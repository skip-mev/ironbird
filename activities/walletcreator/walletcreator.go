package walletcreator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	petriutil "github.com/skip-mev/petri/core/v3/util"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
	"github.com/skip-mev/petri/cosmos/v3/wallet"
	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/core/activities/testnet"
	"github.com/skip-mev/ironbird/core/messages"
	"github.com/skip-mev/ironbird/core/util"
)

type Activity struct {
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
	TelemetrySettings digitalocean.TelemetrySettings
}

func (a *Activity) CreateWallets(ctx context.Context, req messages.CreateWalletsRequest) (messages.CreateWalletsResponse, error) {
	logger, _ := zap.NewDevelopment()
	logger.Info("Creating wallets", zap.Int("numWallets", req.NumWallets))

	p, err := util.RestoreProvider(ctx, logger, messages.RunnerType(req.RunnerType), req.ProviderState, util.ProviderOptions{
		DOToken: a.DOToken, TailscaleSettings: a.TailscaleSettings, TelemetrySettings: a.TelemetrySettings})

	if err != nil {
		return messages.CreateWalletsResponse{}, fmt.Errorf("failed to restore provider: %w", err)
	}

	walletConfig := testnet.CosmosWalletConfig
	if req.Evm {
		walletConfig = testnet.EvmCosmosWalletConfig
	}

	chain, err := chain.RestoreChain(ctx, logger, p, req.ChainState, node.RestoreNode, walletConfig)
	if err != nil {
		return messages.CreateWalletsResponse{}, fmt.Errorf("failed to restore chain: %w", err)
	}

	var mnemonics []string
	var addresses []string
	var walletsMutex sync.Mutex

	faucetWallet := chain.GetFaucetWallet()

	var wg sync.WaitGroup

	for i := 0; i < req.NumWallets; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w, err := wallet.NewGeneratedWallet(petriutil.RandomString(5), walletConfig)
			if err != nil {
				logger.Error("failed to create wallet", zap.Error(err))
				return
			}

			walletsMutex.Lock()
			mnemonics = append(mnemonics, w.Mnemonic())
			addresses = append(addresses, w.FormattedAddress())
			walletsMutex.Unlock()
		}()
	}

	wg.Wait()
	logger.Info("successfully created wallets", zap.Int("count", len(mnemonics)))

	validators := chain.GetValidators()
	node := validators[len(validators)-1]
	err = node.RecoverKey(ctx, "faucet", faucetWallet.Mnemonic())
	if err != nil {
		logger.Error("failed to recover faucet wallet key", zap.Error(err))
		return messages.CreateWalletsResponse{}, fmt.Errorf("failed to restore faucet wallet: %w", err)
	}
	time.Sleep(1 * time.Second)

	chainConfig := chain.GetConfig()
	command := []string{
		chainConfig.BinaryName,
		"tx", "bank", "multi-send",
		faucetWallet.FormattedAddress(),
	}

	command = append(command, addresses...)
	command = append(command, fmt.Sprintf("1000000000%s", chainConfig.Denom),
		"--chain-id", chainConfig.ChainId,
		"--keyring-backend", "test",
		"--from", "faucet",
		"--fees", fmt.Sprintf("80000%s", chainConfig.Denom),
		"--gas", "auto",
		"--yes",
		"--home", chainConfig.HomeDir,
	)

	_, stderr, exitCode, err := node.RunCommand(ctx, command)
	if err != nil || exitCode != 0 {
		logger.Error("failed to fund wallets", zap.Error(err), zap.String("stderr", stderr))
		return messages.CreateWalletsResponse{}, fmt.Errorf("failed to restore fund wallets: %w", err)
	}
	time.Sleep(5 * time.Second)

	return messages.CreateWalletsResponse{
		Mnemonics: mnemonics,
	}, nil
}

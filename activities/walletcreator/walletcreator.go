package walletcreator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/core/v3/provider/docker"
	petriutil "github.com/skip-mev/petri/core/v3/util"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
	"github.com/skip-mev/petri/cosmos/v3/wallet"
	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
)

type Activity struct {
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
	TelemetrySettings digitalocean.TelemetrySettings
}

func handleWalletCreationError(ctx context.Context, logger *zap.Logger, p provider.ProviderI, chain *chain.Chain, originalErr error, errMsg string) (messages.CreateWalletsResponse, error) {
	res := messages.CreateWalletsResponse{}
	wrappedErr := fmt.Errorf("%s: %w", errMsg, originalErr)

	newProviderState, serializeErr := p.SerializeProvider(ctx)
	if serializeErr != nil {
		logger.Error("failed to serialize provider after error", zap.Error(wrappedErr), zap.Error(serializeErr))
		return res, fmt.Errorf("%w; also failed to serialize provider: %v", wrappedErr, serializeErr)
	}
	res.ProviderState = newProviderState

	if chain != nil {
		newChainState, chainErr := chain.Serialize(ctx, p)
		if chainErr != nil {
			logger.Error("failed to serialize chain after error", zap.Error(wrappedErr), zap.Error(chainErr))
			return res, fmt.Errorf("%w; also failed to serialize chain: %v", wrappedErr, chainErr)
		}
		res.ChainState = newChainState
	}

	return res, wrappedErr
}

func (a *Activity) CreateWallets(ctx context.Context, req messages.CreateWalletsRequest) (messages.CreateWalletsResponse, error) {
	logger, _ := zap.NewDevelopment()
	logger.Info("Creating wallets", zap.Int("numWallets", req.NumWallets))

	// Restore provider based on runner type
	var p provider.ProviderI
	var err error
	if req.RunnerType == string(testnettypes.Docker) {
		p, err = docker.RestoreProvider(
			ctx,
			logger,
			req.ProviderState,
		)
	} else {
		p, err = digitalocean.RestoreProvider(
			ctx,
			req.ProviderState,
			a.DOToken,
			a.TailscaleSettings,
			digitalocean.WithLogger(logger),
			digitalocean.WithTelemetry(a.TelemetrySettings),
		)
	}

	if err != nil {
		return messages.CreateWalletsResponse{}, fmt.Errorf("failed to restore provider: %w", err)
	}

	// Get wallet config based on GaiaEVM flag
	walletConfig := testnet.CosmosWalletConfig
	if req.GaiaEVM {
		walletConfig = testnet.EVMCosmosWalletConfig
		logger.Info("using EVM wallet config")
	}

	chain, err := chain.RestoreChain(ctx, logger, p, req.ChainState, node.RestoreNode, walletConfig)
	if err != nil {
		return handleWalletCreationError(ctx, logger, p, nil, err, "failed to restore chain")
	}

	// Create wallets
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
	logger.Info("successfully created all wallets", zap.Int("count", len(mnemonics)))

	// Fund the wallets
	validators := chain.GetValidators()
	node := validators[len(validators)-1]
	err = node.RecoverKey(ctx, "faucet", faucetWallet.Mnemonic())
	if err != nil {
		logger.Error("failed to recover faucet wallet key", zap.Error(err))
		return handleWalletCreationError(ctx, logger, p, chain, err, "failed to recover faucet wallet key")
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
		logger.Warn("failed to fund wallets", zap.Error(err), zap.String("stderr", stderr))
	}
	time.Sleep(5 * time.Second)

	// Serialize provider and chain state
	newProviderState, err := p.SerializeProvider(ctx)
	if err != nil {
		logger.Error("failed to serialize provider after successful run", zap.Error(err))
		return messages.CreateWalletsResponse{Mnemonics: mnemonics}, fmt.Errorf("wallet creation succeeded, but failed to serialize provider: %w", err)
	}

	newChainState, err := chain.Serialize(ctx, p)
	if err != nil {
		logger.Error("failed to serialize chain after successful run", zap.Error(err))
		return messages.CreateWalletsResponse{ProviderState: newProviderState, Mnemonics: mnemonics}, fmt.Errorf("wallet creation succeeded, but failed to serialize chain: %w", err)
	}

	return messages.CreateWalletsResponse{
		ProviderState: newProviderState,
		ChainState:    newChainState,
		Mnemonics:     mnemonics,
	}, nil
}

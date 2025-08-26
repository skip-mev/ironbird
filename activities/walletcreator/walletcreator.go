package walletcreator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
	types2 "github.com/skip-mev/ironbird/petri/core/types"
	petriutil "github.com/skip-mev/ironbird/petri/core/util"
	"github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/skip-mev/ironbird/petri/cosmos/node"
	"github.com/skip-mev/ironbird/petri/cosmos/wallet"
	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"
	pb "github.com/skip-mev/ironbird/server/proto"
	"github.com/skip-mev/ironbird/util"
)

const (
	walletFundChunkSize = 100
	maxRetries          = 3
	baseRetryDelay      = 1 * time.Second
)

type Activity struct {
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
	TelemetrySettings digitalocean.TelemetrySettings
	GRPCClient        pb.IronbirdServiceClient
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
	if req.IsEvmChain {
		walletConfig = testnet.EvmCosmosWalletConfig
	}

	chain, err := chain.RestoreChain(ctx, logger, p, req.ChainState, node.RestoreNode, walletConfig)
	if err != nil {
		return messages.CreateWalletsResponse{}, fmt.Errorf("failed to restore chain: %w", err)
	}

	mnemonics := make([]string, req.NumWallets)
	addresses := make([]string, req.NumWallets)
	faucetWallet := chain.GetFaucetWallet()

	var wg sync.WaitGroup
	for i := range req.NumWallets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w, err := wallet.NewGeneratedWallet(petriutil.RandomString(5), walletConfig)
			if err != nil {
				logger.Error("failed to create wallet", zap.Error(err))
				return
			}
			mnemonics[i] = w.Mnemonic()
			addresses[i] = w.FormattedAddress()
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
	commands := getFundWalletCommands(chainConfig, req.NumWallets, faucetWallet, addresses)

	for _, command := range commands {
		for retry := 0; retry < maxRetries; retry++ {
			if retry > 0 {
				time.Sleep(baseRetryDelay)
			}

			stdout, stderr, exitCode, err := node.RunCommand(ctx, command)
			if err == nil && exitCode == 0 {
				logger.Info("fund result", zap.String("stdout", stdout))
				break
			}

			if retry == maxRetries-1 {
				logger.Error("failed to fund wallets", zap.Error(err), zap.String("stderr", stderr))
				return messages.CreateWalletsResponse{}, fmt.Errorf("failed to fund wallets: %w", err)
			}
		}
	}
	time.Sleep(5 * time.Second)

	if a.GRPCClient != nil {
		walletInfo := &pb.WalletInfo{
			FaucetAddress:  faucetWallet.FormattedAddress(),
			FaucetMnemonic: faucetWallet.Mnemonic(),
			UserAddresses:  addresses,
			UserMnemonics:  mnemonics,
		}

		updateReq := &pb.UpdateWorkflowDataRequest{
			WorkflowId: req.WorkflowID,
			Wallets:    walletInfo,
		}

		_, err = a.GRPCClient.UpdateWorkflowData(ctx, updateReq)
		if err != nil {
			logger.Error("Failed to update workflow wallets", zap.Error(err))
		} else {
			logger.Info("Successfully updated workflow wallets", zap.String("workflowID", req.WorkflowID))
		}
	}

	return messages.CreateWalletsResponse{
		Mnemonics: mnemonics,
	}, nil
}

func getFundWalletCommands(chainConfig types2.ChainConfig, numWallets int, faucet types2.WalletI, addresses []string) [][]string {
	commands := make([][]string, 0)
	for i := 0; i <= numWallets/walletFundChunkSize; i++ {
		command := []string{
			chainConfig.BinaryName,
			"tx", "bank", "multi-send",
			faucet.FormattedAddress(),
		}
		first := i * walletFundChunkSize
		last := first + walletFundChunkSize
		if last > len(addresses) {
			last = len(addresses)
		}

		receivers := addresses[first:last]

		if len(receivers) == 0 {
			return commands
		}

		var gasPrices string
		var amount string
		if chainConfig.IsEVMChain {
			amount = "10000000000000000"
			gasPrices = "770000000"
		} else {
			amount = "1000000000"
			gasPrices = "160000"
		}

		command = append(command, receivers...)
		command = append(command, fmt.Sprintf("%s%s", amount, chainConfig.Denom),
			"--chain-id", chainConfig.ChainId,
			"--keyring-backend", "test",
			"--from", "faucet",
			"--gas-prices", fmt.Sprintf("%s%s", gasPrices, chainConfig.Denom),
			"--gas", "auto",
			"--gas-adjustment", "1.5",
			"--yes",
			"--home", chainConfig.HomeDir,
		)
		commands = append(commands, command)
	}
	return commands
}

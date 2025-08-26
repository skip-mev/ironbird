package node

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/petri/core/types"
	"github.com/skip-mev/ironbird/petri/core/util"
	"github.com/skip-mev/ironbird/petri/cosmos/wallet"
)

// CreateWallet creates a new wallet on the node using a randomly generated mnemonic
func (n *Node) CreateWallet(ctx context.Context, name string, walletConfig types.WalletConfig) (types.WalletI, error) {
	n.logger.Info("creating wallet", zap.String("name", name))

	keyWallet, err := wallet.NewGeneratedWallet(name, walletConfig) // todo: fix this to depend on WalletI
	if err != nil {
		return nil, err
	}

	err = n.RecoverKey(ctx, name, keyWallet.Mnemonic())
	if err != nil {
		return nil, err
	}

	return keyWallet, nil
}

// RecoverWallet recovers a wallet on the node using a mnemonic
func (n *Node) RecoverKey(ctx context.Context, name, mnemonic string) error {
	n.logger.Info("recovering wallet", zap.String("name", name), zap.String("mnemonic", mnemonic))

	command := []string{
		"sh",
		"-c",
		fmt.Sprintf(`echo %q | %s keys add %s --recover --keyring-backend %s --coin-type %s --home %s --output json`, mnemonic, n.GetChainConfig().BinaryName, name, keyring.BackendTest, n.GetChainConfig().CoinType, n.GetChainConfig().HomeDir),
	}

	stdout, stderr, exitCode, err := n.RunCommand(ctx, command)
	n.logger.Debug("RecoverKey", zap.String("name", name), zap.String("stdout", stdout),
		zap.String("stderr", stderr), zap.Any("exitCode", exitCode))

	return err
}

// KeyBech32 returns the bech32 address of a key on the node using the app's binary
func (n *Node) KeyBech32(ctx context.Context, name, bech string) (string, error) {
	command := []string{
		n.GetChainConfig().BinaryName,
		"keys", "show", name, "-a", "--keyring-backend", keyring.BackendTest, "--home", n.GetChainConfig().HomeDir,
	}

	if bech != "" {
		command = append(command, "--bech", bech)
	}

	stdout, stderr, exitCode, err := n.RunCommand(ctx, command)
	n.logger.Debug("show key", zap.String("name", name), zap.String("stdout", stdout), zap.String("stderr", stderr))

	if err != nil {
		return "", fmt.Errorf("failed to show key %q (stderr=%q): %w", name, stderr, err)
	}

	if exitCode != 0 {
		return "", fmt.Errorf("failed to show key %q (exitcode=%d): %s", name, exitCode, stderr)
	}

	return util.CleanDockerOutput(stdout), nil
}

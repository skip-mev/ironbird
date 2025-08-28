package node

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/zap"

	petritypes "github.com/skip-mev/ironbird/petri/core/types"
)

// GenesisFileContent returns the genesis file on the node in byte format
func (n *Node) GenesisFileContent(ctx context.Context) ([]byte, error) {
	n.logger.Info("reading genesis file", zap.String("node", n.GetDefinition().Name))

	bz, err := n.ReadFile(ctx, "config/genesis.json")
	if err != nil {
		return nil, err
	}

	return bz, err
}

// CopyGenTx retrieves the genesis transaction from the node and copy it to the destination node
func (n *Node) CopyGenTx(ctx context.Context, dstNode petritypes.NodeI) error {
	n.logger.Info("copying gen tx", zap.String("from", n.GetConfig().Name), zap.String("to", dstNode.GetConfig().Name))

	nid, err := n.NodeId(ctx)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("config/gentx/gentx-%s.json", nid)

	n.logger.Debug("reading gen tx", zap.String("node", n.GetConfig().Name))
	gentx, err := n.ReadFile(context.Background(), path)
	if err != nil {
		return err
	}

	n.logger.Debug("writing gen tx", zap.String("node", dstNode.GetConfig().Name))
	return dstNode.WriteFile(context.Background(), path, gentx)
}

// AddGenesisAccount adds a genesis account to the node's local genesis file
func (n *Node) AddGenesisAccount(ctx context.Context, address string, genesisAmounts []types.Coin) error {
	n.logger.Debug("adding genesis account", zap.String("node", n.GetDefinition().Name), zap.String("address", address))

	amount := ""

	for i, coin := range genesisAmounts {
		if i != 0 {
			amount += ","
		}

		amount += fmt.Sprintf("%s%s", coin.Amount.String(), coin.Denom)
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	var command []string

	if n.GetChainConfig().UseGenesisSubCommand {
		command = append(command, "genesis")
	}

	command = append(command, "add-genesis-account", address, amount, "--keyring-backend", keyring.BackendTest)
	command = n.BinCommand(command...)

	stdout, stderr, exitCode, err := n.RunCommand(ctx, command)
	n.logger.Debug("add-genesis-account", zap.String("stdout", stdout), zap.String("stderr", stderr), zap.Int("exitCode", exitCode))

	if err != nil {
		return fmt.Errorf("failed to add genesis account: %w", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("failed to add genesis account (exitcode=%d): %s", exitCode, stderr)
	}

	return nil
}

// GenerateGenTx generates a genesis transaction for the node
func (n *Node) GenerateGenTx(ctx context.Context, genesisSelfDelegation types.Coin) error {
	n.logger.Info("generating genesis transaction", zap.String("node", n.GetDefinition().Name))

	var command []string

	if n.GetChainConfig().UseGenesisSubCommand {
		command = append(command, "genesis")
	}

	command = append(command, "gentx", petritypes.ValidatorKeyName, fmt.Sprintf("%s%s", genesisSelfDelegation.Amount.String(), genesisSelfDelegation.Denom),
		"--keyring-backend", keyring.BackendTest,
		"--chain-id", n.GetChainConfig().ChainId)

	command = n.BinCommand(command...)

	stdout, stderr, exitCode, err := n.RunCommand(ctx, command)
	n.logger.Debug("gentx", zap.String("stdout", stdout), zap.String("stderr", stderr),
		zap.Int("exitCode", exitCode), zap.Strings("command", command))

	if err != nil {
		return fmt.Errorf("failed to generate genesis transaction: %w", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("failed to generate genesis transaction (exitcode=%d): %s", exitCode, stderr)
	}

	return nil
}

// CollectGenTxs collects the genesis transactions from the node and create a finalized genesis file
func (n *Node) CollectGenTxs(ctx context.Context) error {
	n.logger.Info("collecting genesis transactions", zap.String("node", n.GetDefinition().Name))

	command := []string{}

	if n.GetChainConfig().UseGenesisSubCommand {
		command = append(command, "genesis")
	}

	command = append(command, "collect-gentxs")

	stdout, stderr, exitCode, err := n.RunCommand(ctx, n.BinCommand(command...))
	n.logger.Debug("collect-gentxs", zap.String("stdout", stdout), zap.String("stderr", stderr), zap.Int("exitCode", exitCode))

	if err != nil {
		return fmt.Errorf("failed to collect genesis transactions: %w", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("failed to collect genesis transactions (exitcode=%d): %s", exitCode, stderr)
	}

	return nil
}

// OverwriteGenesisFile overwrites the genesis file on the node with the provided genesis file
func (n *Node) OverwriteGenesisFile(ctx context.Context, bz []byte) error {
	n.logger.Info("overwriting genesis file", zap.String("node", n.GetDefinition().Name))

	return n.WriteFile(ctx, "config/genesis.json", bz)
}

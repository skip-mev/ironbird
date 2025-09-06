package node

import (
	"context"
	"fmt"

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

// OverwriteGenesisFile overwrites the genesis file on the node with the provided genesis file
func (n *Node) OverwriteGenesisFile(ctx context.Context, bz []byte) error {
	n.logger.Info("overwriting genesis file", zap.String("node", n.GetDefinition().Name))

	return n.WriteFile(ctx, "config/genesis.json", bz)
}

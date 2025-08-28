package node_test

import (
	"context"
	"testing"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/docker"
	"github.com/skip-mev/ironbird/petri/core/types"
	"github.com/skip-mev/ironbird/petri/cosmos/node"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const idAlphabet = "abcdefghijklqmnoqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

var defaultChainConfig = types.ChainConfig{
	Denom:         "stake",
	Decimals:      6,
	NumValidators: 1,
	NumNodes:      0,
	BinaryName:    "/usr/bin/simd",
	Image: provider.ImageDefinition{
		Image: "ghcr.io/cosmos/simapp:v0.47",
		UID:   "1000",
		GID:   "1000",
	},
	GasPrices:            "0.0005stake",
	Bech32Prefix:         "stake",
	HomeDir:              "/gaia",
	CoinType:             "118",
	ChainId:              "stake-1",
	UseGenesisSubCommand: true,
}

func TestNodeLifecycle(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()
	providerName := gonanoid.MustGenerate(idAlphabet, 10)
	chainName := gonanoid.MustGenerate(idAlphabet, 5)

	p, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)
	defer func(p provider.ProviderI, ctx context.Context) {
		require.NoError(t, p.Teardown(ctx))
	}(p, ctx)

	chainConfig := defaultChainConfig
	chainConfig.Name = chainName

	n, err := node.CreateNode(ctx, logger, p, types.NodeConfig{
		Name:        "test",
		Index:       0,
		ChainConfig: chainConfig,
	}, types.NodeOptions{})
	require.NoError(t, err)

	defer func(n types.NodeI, ctx context.Context) {
		require.NoError(t, n.Stop(ctx))
	}(n, ctx)

	status, err := n.GetStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_STOPPED, status)

	err = n.Start(ctx)
	require.NoError(t, err)
}

func TestNodeSerialization(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()
	providerName := gonanoid.MustGenerate(idAlphabet, 10)
	chainName := gonanoid.MustGenerate(idAlphabet, 5)

	p, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)
	defer func(p provider.ProviderI, ctx context.Context) {
		require.NoError(t, p.Teardown(ctx))
	}(p, ctx)

	chainConfig := defaultChainConfig
	chainConfig.Name = chainName

	n, err := node.CreateNode(ctx, logger, p, types.NodeConfig{
		Name:        "test",
		Index:       0,
		ChainConfig: chainConfig,
	}, types.NodeOptions{})
	require.NoError(t, err)
	defer func(n types.NodeI, ctx context.Context) {
		require.NoError(t, n.Stop(ctx))
	}(n, ctx)

	status, err := n.GetStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_STOPPED, status)

	err = n.Start(ctx)
	require.NoError(t, err)

	state, err := n.Serialize(ctx, p)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	n2, err := node.RestoreNode(ctx, logger, state, p)
	require.NoError(t, err)

	status, err = n2.GetStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_RUNNING, status)
}

package e2e

import (
	"context"
	"flag"
	"testing"

	"github.com/skip-mev/ironbird/petri/core/util"

	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/docker"
	"github.com/skip-mev/ironbird/petri/core/types"
	cosmoschain "github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/skip-mev/ironbird/petri/cosmos/node"
	"github.com/skip-mev/ironbird/petri/cosmos/tests/e2e"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	defaultChainConfig = types.ChainConfig{
		Denom:         "stake",
		Decimals:      6,
		NumValidators: 1,
		NumNodes:      1,
		BinaryName:    "/usr/bin/simd",
		Image: provider.ImageDefinition{
			Image: "ghcr.io/cosmos/simapp:v0.47",
			UID:   "1000",
			GID:   "1000",
		},
		GasPrices:            "0.0005stake",
		Bech32Prefix:         "cosmos",
		HomeDir:              "/gaia",
		CoinType:             "118",
		ChainId:              "stake-1",
		UseGenesisSubCommand: true,
		SetPersistentPeers:   true,
	}

	defaultChainOptions = types.ChainOptions{
		NodeCreator: node.CreateNode,
		WalletConfig: types.WalletConfig{
			SigningAlgorithm: string(hd.Secp256k1.Name()),
			Bech32Prefix:     "cosmos",
			HDPath:           hd.CreateHDPath(118, 0, 0),
			DerivationFn:     hd.Secp256k1.Derive(),
			GenerationFn:     hd.Secp256k1.Generate(),
		},
	}

	numTestChains = flag.Int("num-chains", 3, "number of chains to create for concurrent testing")
	numNodes      = flag.Int("num-nodes", 1, "number of nodes per chain")
	numValidators = flag.Int("num-validators", 1, "number of validators per chain")
)

func TestDockerE2E(t *testing.T) {
	if !flag.Parsed() {
		flag.Parse()
	}

	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	defer func() {
		dockerClient, err := client.NewClientWithOpts()
		if err != nil {
			t.Logf("Failed to create Docker client for volume cleanup: %v", err)
			return
		}
		_, err = dockerClient.VolumesPrune(ctx, filters.Args{})
		if err != nil {
			t.Logf("Failed to prune volumes: %v", err)
		}
	}()

	providerName := util.RandomString(5)

	p, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)

	chains := make([]*cosmoschain.Chain, *numTestChains)

	// Create first half of chains
	defaultChainConfig.NumNodes = *numNodes
	defaultChainConfig.NumValidators = *numValidators
	e2e.CreateChainsConcurrently(ctx, t, logger, p, 0, *numTestChains/2, chains,
		defaultChainConfig, "chain-%d", defaultChainOptions)

	// Restore provider before creating second half of chains
	serializedProvider, err := p.SerializeProvider(ctx)
	require.NoError(t, err)
	restoredProvider, err := docker.RestoreProvider(ctx, logger, serializedProvider)
	require.NoError(t, err)

	// Restore the existing chains with the restored provider
	restoredChains := make([]*cosmoschain.Chain, *numTestChains)
	for i := 0; i < *numTestChains/2; i++ {
		chainState, err := chains[i].Serialize(ctx, restoredProvider)
		require.NoError(t, err)

		restoredChain, err := cosmoschain.RestoreChain(ctx, logger, restoredProvider, chainState, node.RestoreNode, defaultChainOptions.WalletConfig)
		require.NoError(t, err)

		require.Equal(t, chains[i].GetConfig(), restoredChain.GetConfig())
		require.Equal(t, len(chains[i].GetValidators()), len(restoredChain.GetValidators()))

		restoredChains[i] = restoredChain
	}

	// Create second half of chains with restored provider
	e2e.CreateChainsConcurrently(ctx, t, logger, restoredProvider, *numTestChains/2, *numTestChains,
		restoredChains, defaultChainConfig, "chain-%d", defaultChainOptions)

	// Test and teardown half the chains individually
	for i := 0; i < *numTestChains/2; i++ {
		originalChain := restoredChains[i]
		validators := originalChain.GetValidators()
		nodes := originalChain.GetNodes()

		for _, validator := range validators {
			e2e.AssertNodeRunning(t, ctx, validator)
		}

		for _, node := range nodes {
			e2e.AssertNodeRunning(t, ctx, node)
		}

		err = originalChain.WaitForBlocks(ctx, 2)
		require.NoError(t, err)

		// Test individual chain teardown
		err = originalChain.Teardown(ctx)
		require.NoError(t, err)

		for _, validator := range validators {
			e2e.AssertNodeShutdown(t, ctx, validator)
		}

		for _, node := range nodes {
			e2e.AssertNodeShutdown(t, ctx, node)
		}
	}

	// Test the remaining chains but let the provider teardown handle their cleanup
	remainingChains := make([]*cosmoschain.Chain, 0)
	for i := *numTestChains / 2; i < *numTestChains; i++ {
		originalChain := restoredChains[i]
		remainingChains = append(remainingChains, originalChain)
		validators := originalChain.GetValidators()
		nodes := originalChain.GetNodes()
		for _, validator := range validators {
			e2e.AssertNodeRunning(t, ctx, validator)
		}
		for _, node := range nodes {
			e2e.AssertNodeRunning(t, ctx, node)
		}

		err = originalChain.WaitForBlocks(ctx, 2)
		require.NoError(t, err)
	}

	require.NoError(t, restoredProvider.Teardown(ctx))
	// Verify all remaining chains are properly torn down
	for _, chain := range remainingChains {
		validators := chain.GetValidators()
		nodes := chain.GetNodes()

		for _, validator := range validators {
			e2e.AssertNodeShutdown(t, ctx, validator)
		}

		for _, node := range nodes {
			e2e.AssertNodeShutdown(t, ctx, node)
		}
	}
}

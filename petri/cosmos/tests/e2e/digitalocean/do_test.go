package e2e

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/skip-mev/ironbird/petri/core/util"

	"tailscale.com/tsnet"

	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
	"github.com/skip-mev/ironbird/petri/core/types"
	cosmoschain "github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/skip-mev/ironbird/petri/cosmos/node"
	"github.com/skip-mev/ironbird/petri/cosmos/tests/e2e"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	defaultChainConfig = types.ChainConfig{
		Denom:    "stake",
		Decimals: 6,
		RegionConfig: []types.RegionConfig{
			{
				Name:          "nyc1",
				NumValidators: 1,
				NumNodes:      1,
			},
			{
				Name:          "fra1",
				NumValidators: 1,
				NumNodes:      0,
			},
		},
		BinaryName: "/usr/bin/simd",
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
		NodeOptions: types.NodeOptions{
			NodeDefinitionModifier: func(def provider.TaskDefinition, nodeConfig types.NodeConfig) provider.TaskDefinition {
				doConfig := digitalocean.DigitalOceanTaskConfig{
					"size":     "s-2vcpu-4gb",
					"region":   "ams3",
					"image_id": os.Getenv("DO_IMAGE_ID"),
				}
				def.ProviderSpecificConfig = doConfig
				return def
			},
		},
	}

	numTestChains = flag.Int("num-chains", 3, "number of chains to create for concurrent testing")
)

func TestDOE2E(t *testing.T) {
	if !flag.Parsed() {
		flag.Parse()
	}

	var restoredProvider provider.ProviderI
	providerTornDown := false

	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	doToken := os.Getenv("DO_API_TOKEN")
	if doToken == "" {
		logger.Fatal("DO_API_TOKEN environment variable not set")
	}

	imageID := os.Getenv("DO_IMAGE_ID")
	if imageID == "" {
		logger.Fatal("DO_IMAGE_ID environment variable not set")
	}

	clientAuthKey := os.Getenv("TS_CLIENT_AUTH_KEY")
	if clientAuthKey == "" {
		logger.Fatal("TS_CLIENT_AUTH_KEY environment variable not set")
	}

	serverOauthSecret := os.Getenv("TS_SERVER_OAUTH_SECRET")
	if serverOauthSecret == "" {
		logger.Fatal("TS_SERVER_AUTH_KEY environment variable not set")
	}

	serverAuthKey, err := digitalocean.GenerateTailscaleAuthKey(ctx, serverOauthSecret, []string{"petri-e2e"})
	require.NoError(t, err)

	tsServer := tsnet.Server{
		AuthKey:   serverAuthKey,
		Ephemeral: true,
		Hostname:  "petri-e2e",
	}

	localClient, err := tsServer.LocalClient()
	require.NoError(t, err)

	tailscaleSettings := digitalocean.TailscaleSettings{
		AuthKey:     clientAuthKey,
		Server:      &tsServer,
		Tags:        []string{"petri-e2e"},
		LocalClient: localClient,
	}
	providerName := util.RandomString(5)

	p, err := digitalocean.NewProvider(ctx, providerName, doToken, tailscaleSettings, digitalocean.WithLogger(logger))
	defer func() {
		if restoredProvider != nil {
			return
		}

		require.NoError(t, p.Teardown(ctx))
		providerTornDown = true
	}()
	require.NoError(t, err)

	chains := make([]*cosmoschain.Chain, *numTestChains)

	// Create first half of chains
	e2e.CreateChainsConcurrently(ctx, t, logger, p, 0, *numTestChains/2, chains,
		defaultChainConfig, "chain-%d", defaultChainOptions)

	// Restore provider before creating second half of chains
	serializedProvider, err := p.SerializeProvider(ctx)
	require.NoError(t, err)
	restoredProvider, err = digitalocean.RestoreProvider(ctx, serializedProvider, doToken, tailscaleSettings, digitalocean.WithLogger(logger))
	require.NoError(t, err)
	defer func() {
		if providerTornDown {
			return
		}

		require.NoError(t, restoredProvider.Teardown(ctx))
	}()

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
	e2e.CreateChainsConcurrently(ctx, t, logger, restoredProvider, *numTestChains/2,
		*numTestChains, restoredChains, defaultChainConfig, "chain-%d", defaultChainOptions)

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

		// wait for status to update on DO client side
		time.Sleep(15 * time.Second)

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
	providerTornDown = true
	// wait for status to update on DO client side
	time.Sleep(15 * time.Second)

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

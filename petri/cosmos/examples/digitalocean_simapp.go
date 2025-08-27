package examples

import (
	"context"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/skip-mev/ironbird/petri/cosmos/node"

	"github.com/skip-mev/ironbird/petri/core/provider"
	petritypes "github.com/skip-mev/ironbird/petri/core/types"
	"go.uber.org/zap"
)

func main() {
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

	tailscaleSettings, err := digitalocean.SetupTailscale(ctx, serverOauthSecret,
		clientAuthKey, "ironbird-tests", []string{"ironbird-e2e"}, []string{"ironbird-e2e"})
	if err != nil {
		logger.Fatal("failed to generate Tailscale auth key", zap.Error(err))
	}

	doProvider, err := digitalocean.NewProvider(
		ctx,
		"cosmos-hub",
		doToken,
		tailscaleSettings,
		digitalocean.WithLogger(logger),
	)

	if err != nil {
		logger.Fatal("failed to create DigitalOcean provider", zap.Error(err))
	}

	chainConfig := petritypes.ChainConfig{
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
	}

	chainOptions := petritypes.ChainOptions{
		NodeCreator: node.CreateNode,
		NodeOptions: petritypes.NodeOptions{
			NodeDefinitionModifier: func(def provider.TaskDefinition, nodeConfig petritypes.NodeConfig) provider.TaskDefinition {
				doConfig := digitalocean.DigitalOceanTaskConfig{
					"size":     "s-2vcpu-4gb",
					"region":   "ams3",
					"image_id": imageID,
				}
				def.ProviderSpecificConfig = doConfig
				return def
			},
		},
		WalletConfig: petritypes.WalletConfig{
			SigningAlgorithm: string(hd.Secp256k1.Name()),
			Bech32Prefix:     "cosmos",
			HDPath:           hd.CreateHDPath(118, 0, 0),
			DerivationFn:     hd.Secp256k1.Derive(),
			GenerationFn:     hd.Secp256k1.Generate(),
		},
	}

	logger.Info("Creating chain")
	cosmosChain, err := chain.CreateChain(ctx, logger, doProvider, chainConfig, chainOptions)
	if err != nil {
		logger.Fatal("failed to create chain", zap.Error(err))
	}

	logger.Info("Initializing chain")
	err = cosmosChain.Init(ctx, chainOptions)
	if err != nil {
		logger.Fatal("failed to initialize chain", zap.Error(err))
	}

	logger.Info("Chain is successfully running! Waiting for chain to produce blocks")
	err = cosmosChain.WaitForBlocks(ctx, 1)
	if err != nil {
		logger.Fatal("failed waiting for blocks", zap.Error(err))
	}

	// Comment out section below if you want to persist your Digital Ocean resources
	logger.Info("Chain has successfully produced required number of blocks. Tearing down Digital Ocean resources.")
	err = doProvider.Teardown(ctx)
	if err != nil {
		logger.Fatal("failed to teardown provider", zap.Error(err))
	}

	logger.Info("All Digital Ocean resources created have been successfully deleted!")
}

func getExternalIP() (string, error) {
	resp, err := http.Get("https://ifconfig.me")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ip)), nil
}

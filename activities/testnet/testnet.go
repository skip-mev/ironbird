package testnet

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/skip-mev/ironbird/activities"
	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/core/v3/types"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/skip-mev/petri/cosmos/v3/node"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
	"net/http"
)

type TestnetOptions struct {
	Name                    string
	Image                   string
	UID                     string
	GID                     string
	BinaryName              string
	HomeDir                 string
	ProviderSpecificOptions map[string]string

	ValidatorCount uint64
	NodeCount      uint64

	ProviderState []byte
	ChainState    []byte
}

type PackagedState struct {
	ProviderState []byte
	ChainState    []byte
	Nodes         []Node
}

type Node struct {
	Name    string
	Rpc     string
	Lcd     string
	Metrics string
}

type Activity struct {
	TailscaleServer *activities.TailscaleServer
}

func (a *Activity) CreateProvider(ctx context.Context, opts TestnetOptions) (string, error) {
	logger, _ := zap.NewDevelopment()

	p, err := digitalocean.NewProvider(
		ctx,
		opts.Name,
		"<DO_TOKEN>",
		digitalocean.WithLogger(logger),
		digitalocean.WithTailscale(a.TailscaleServer.Server, a.TailscaleServer.NodeAuthkey, a.TailscaleServer.Tags),
	)

	if err != nil {
		return "", err
	}

	state, err := p.SerializeProvider(ctx)

	return string(state), err
}

func (a *Activity) TeardownProvider(ctx context.Context, opts TestnetOptions) (string, error) {
	logger, _ := zap.NewDevelopment()
	p, err := digitalocean.RestoreProvider(
		ctx,
		opts.ProviderState,
		"<DO_TOKEN>",
		digitalocean.WithLogger(logger),
		digitalocean.WithTailscale(a.TailscaleServer.Server, a.TailscaleServer.NodeAuthkey, a.TailscaleServer.Tags),
	)

	if err != nil {
		return "", err
	}

	err = p.Teardown(ctx)
	return "", err
}

func (a *Activity) LaunchTestnet(ctx context.Context, opts TestnetOptions) (PackagedState, error) {
	logger, _ := zap.NewDevelopment()

	p, err := digitalocean.RestoreProvider(
		ctx,
		opts.ProviderState,
		"<DO_TOKEN>",
		digitalocean.WithLogger(logger),
		digitalocean.WithTailscale(a.TailscaleServer.Server, a.TailscaleServer.NodeAuthkey, a.TailscaleServer.Tags),
	)

	if err != nil {
		return PackagedState{}, err
	}

	chain, err := chain.CreateChain(
		ctx,
		logger,
		p,
		types.ChainConfig{
			Denom:         "stake",
			Decimals:      6,
			NumValidators: int(opts.ValidatorCount),
			NumNodes:      int(opts.NodeCount),
			BinaryName:    opts.BinaryName,
			Image: provider.ImageDefinition{
				Image: opts.Image,
				UID:   opts.UID,
				GID:   opts.GID,
			},
			GasPrices:            "0.0005stake",
			Bech32Prefix:         "cosmos",
			HomeDir:              opts.HomeDir,
			CoinType:             "118",
			ChainId:              opts.Name,
			UseGenesisSubCommand: true,
		},
		types.ChainOptions{
			NodeCreator: node.CreateNode,
			NodeOptions: types.NodeOptions{
				NodeDefinitionModifier: func(definition provider.TaskDefinition, config types.NodeConfig) provider.TaskDefinition {
					definition.ProviderSpecificConfig = opts.ProviderSpecificOptions
					return definition
				},
			},
			WalletConfig: types.WalletConfig{
				SigningAlgorithm: "secp256k1",
				Bech32Prefix:     "cosmos",
				HDPath:           hd.CreateHDPath(118, 0, 0),
				DerivationFn:     hd.Secp256k1.Derive(),
				GenerationFn:     hd.Secp256k1.Generate(),
			},
		},
	)

	if err != nil {
		return PackagedState{}, temporal.NewApplicationErrorWithOptions("failed to create chain", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	err = chain.Init(ctx, types.ChainOptions{
		NodeCreator: node.CreateNode,
		WalletConfig: types.WalletConfig{
			SigningAlgorithm: "secp256k1",
			Bech32Prefix:     "cosmos",
			HDPath:           hd.CreateHDPath(118, 0, 0),
			DerivationFn:     hd.Secp256k1.Derive(),
			GenerationFn:     hd.Secp256k1.Generate(),
		},
	})

	if err != nil {
		return PackagedState{}, temporal.NewApplicationErrorWithOptions("failed to init chain", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	providerState, err := p.SerializeProvider(ctx)
	if err != nil {
		return PackagedState{}, temporal.NewApplicationErrorWithOptions("failed to serialize provider", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	state, err := chain.Serialize(ctx, p)
	if err != nil {
		return PackagedState{}, temporal.NewApplicationErrorWithOptions("failed to serialize chain", err.Error(), temporal.ApplicationErrorOptions{NonRetryable: true})
	}

	var testnetNodes []Node

	for _, validator := range chain.GetValidators() {
		cosmosIp, err := validator.GetExternalAddress(ctx, "1317")
		if err != nil {
			return PackagedState{}, err
		}

		cometIp, err := validator.GetExternalAddress(ctx, "26657")
		if err != nil {
			return PackagedState{}, err
		}

		metricsIp, err := validator.GetIP(ctx)
		if err != nil {
			return PackagedState{}, err
		}

		testnetNodes = append(testnetNodes, Node{
			Name:    validator.GetDefinition().Name,
			Rpc:     fmt.Sprintf("http://%s", cometIp),
			Lcd:     fmt.Sprintf("http://%s", cosmosIp),
			Metrics: fmt.Sprintf("%s:26660", metricsIp),
		})
	}

	return PackagedState{
		ProviderState: providerState,
		ChainState:    state,
		Nodes:         testnetNodes,
	}, err
}

func (a *Activity) MonitorTestnet(ctx context.Context, opts TestnetOptions) (string, error) {
	logger, _ := zap.NewDevelopment()

	p, err := digitalocean.RestoreProvider(
		ctx,
		opts.ProviderState,
		"<DO_TOKEN>",
		digitalocean.WithLogger(logger),
		digitalocean.WithTailscale(a.TailscaleServer.Server, a.TailscaleServer.NodeAuthkey, a.TailscaleServer.Tags),
	)

	if err != nil {
		return "", err
	}

	chain, err := chain.RestoreChain(ctx, logger, p, opts.ChainState, node.RestoreNode)

	if err != nil {
		return "", err
	}

	addr, err := chain.GetValidators()[0].GetExternalAddress(ctx, "26657")

	if err != nil {
		return "", err
	}

	resp, err := http.Get("http://" + addr + "/status")

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", err
	}

	return "ok", nil
}

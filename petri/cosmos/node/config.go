package node

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/pelletier/go-toml/v2"

	"reflect"
	"time"

	petritypes "github.com/skip-mev/ironbird/petri/core/types"
)

// recursiveModifyToml will apply toml modifications at the current depth,
// then recurse for new depths
func recursiveModifyToml(c map[string]interface{}, modifications map[string]interface{}) error {
	for key, value := range modifications {
		if reflect.ValueOf(value).Kind() == reflect.Map {
			cV, ok := c[key]
			if !ok {
				// Did not find section in existing config, populating fresh.
				cV = make(map[string]interface{})
			}
			// Retrieve existing config to apply overrides to.
			cVM, ok := cV.(map[string]interface{})

			if !ok {
				cVM = make(map[string]interface{})
			}
			if err := recursiveModifyToml(cVM, value.(map[string]interface{})); err != nil {
				return err
			}
			c[key] = cVM
		} else {
			// Not a map, so we can set override value directly.
			c[key] = value
		}
	}
	return nil
}

func GenerateDefaultClientConfig(chainID string) map[string]interface{} {
	clientConfig := make(map[string]interface{})

	clientConfig["chain-id"] = chainID
	clientConfig["keyring-backend"] = keyring.BackendTest
	clientConfig["output"] = "text"
	clientConfig["node"] = "http://localhost:26657"
	clientConfig["broadcast-mode"] = "sync"

	return clientConfig
}

// GenerateDefaultConsensusConfig returns a default / sensible config for CometBFT
func GenerateDefaultConsensusConfig(externalAddr string) map[string]interface{} {
	cometBftConfig := make(map[string]interface{})

	// Set Log Level to info
	cometBftConfig["log_level"] = "info"

	p2p := make(map[string]interface{})

	// Allow p2p strangeness
	p2p["allow_duplicate_ip"] = true
	p2p["addr_book_strict"] = false

	p2p["external_address"] = externalAddr

	cometBftConfig["p2p"] = p2p

	consensusConfig := make(map[string]interface{})

	blockTime := (time.Duration(2) * time.Second).String() // todo(zygimantass): make configurable
	consensusConfig["timeout_commit"] = blockTime
	consensusConfig["timeout_propose"] = blockTime

	cometBftConfig["consensus"] = consensusConfig

	instrumentationConfig := make(map[string]interface{})
	instrumentationConfig["prometheus"] = true

	cometBftConfig["instrumentation"] = instrumentationConfig

	rpc := make(map[string]interface{})

	// Enable public RPC
	rpc["laddr"] = "tcp://0.0.0.0:26657"
	rpc["allowed_origins"] = []string{"*"}

	cometBftConfig["rpc"] = rpc

	return cometBftConfig
}

// GenerateDefaultAppConfig returns a default / sensible config for the Cosmos SDK
func GenerateDefaultAppConfig(c petritypes.ChainConfig) map[string]interface{} {
	sdkConfig := make(map[string]interface{})
	sdkConfig["minimum-gas-prices"] = c.GasPrices

	grpc := make(map[string]interface{})

	// Enable public GRPC
	grpc["address"] = "0.0.0.0:9090"

	sdkConfig["grpc"] = grpc

	api := make(map[string]interface{})

	// Enable public REST API
	api["enable"] = true
	api["swagger"] = true
	api["address"] = "tcp://0.0.0.0:1317"

	sdkConfig["api"] = api

	telemetry := make(map[string]interface{})
	telemetry["enabled"] = true
	telemetry["prometheus-retention-time"] = 3600

	sdkConfig["telemetry"] = telemetry

	if c.IsEVMChain {
		evm := make(map[string]interface{})
		evm["tracer"] = ""
		evm["max-tx-gas-wanted"] = 0
		evm["cache-preimage"] = false

		sdkConfig["evm"] = evm

		jsonRPC := make(map[string]interface{})
		jsonRPC["enable"] = true
		jsonRPC["address"] = "127.0.0.1:8545"
		jsonRPC["ws-address"] = "127.0.0.1:8546"
		jsonRPC["api"] = "eth,net,web3"
		jsonRPC["gas-cap"] = 25000000
		jsonRPC["allow-insecure-unlock"] = true
		jsonRPC["evm-timeout"] = "5s"
		jsonRPC["txfee-cap"] = 1
		jsonRPC["filter-cap"] = 200
		jsonRPC["feehistory-cap"] = 100
		jsonRPC["logs-cap"] = 10000
		jsonRPC["block-range-cap"] = 10000
		jsonRPC["http-timeout"] = "30s"
		jsonRPC["http-idle-timeout"] = "2m0s"
		jsonRPC["allow-unprotected-txs"] = false
		jsonRPC["max-open-connections"] = 0
		jsonRPC["enable-indexer"] = false
		jsonRPC["metrics-address"] = "127.0.0.1:6065"
		jsonRPC["fix-revert-gas-refund-height"] = 0

		sdkConfig["json-rpc"] = jsonRPC
	}

	return sdkConfig
}

// ModifyTomlConfigFile will modify a TOML config file at filePath with the provided modifications.
// If a certain path defined in modifications is not found in the existing config, it will return an error
func (n *Node) ModifyTomlConfigFile(
	ctx context.Context,
	filePath string,
	modifications map[string]interface{},
) error {
	config, err := n.ReadFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to retrieve %s: %w", filePath, err)
	}

	var c map[string]interface{}
	if err := toml.Unmarshal(config, &c); err != nil {
		return fmt.Errorf("failed to unmarshal %s: %w", filePath, err)
	}

	if err := recursiveModifyToml(c, modifications); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(c); err != nil {
		return err
	}

	if err := n.WriteFile(ctx, filePath, buf.Bytes()); err != nil {
		return fmt.Errorf("overwriting %s: %w", filePath, err)
	}

	return nil
}

// SetChainConfigs will generate the default configs for CometBFT and the app, apply custom configs, and write them to disk
func (n *Node) SetChainConfigs(ctx context.Context, chainID string, p2pExternalAddr string) error {
	appConfig := GenerateDefaultAppConfig(n.GetChainConfig())

	consensusConfig := GenerateDefaultConsensusConfig(p2pExternalAddr)
	clientConfig := GenerateDefaultClientConfig(chainID)

	if err := n.ModifyTomlConfigFile(
		ctx,
		"config/app.toml",
		appConfig,
	); err != nil {
		return err
	}

	if err := n.ModifyTomlConfigFile(
		ctx,
		"config/config.toml",
		consensusConfig,
	); err != nil {
		return err
	}

	if err := n.ModifyTomlConfigFile(
		ctx,
		"config/client.toml",
		clientConfig,
	); err != nil {
		return err
	}

	if err := n.ApplyCustomConfigs(ctx); err != nil {
		return err
	}

	return nil
}

// ApplyCustomConfigs applies custom configurations from ChainConfig to the respective config files
func (n *Node) ApplyCustomConfigs(ctx context.Context) error {
	chainConfig := n.GetChainConfig()

	if len(chainConfig.CustomAppConfig) > 0 {
		if err := n.ModifyTomlConfigFile(
			ctx,
			"config/app.toml",
			chainConfig.CustomAppConfig,
		); err != nil {
			return fmt.Errorf("failed to apply custom app config: %w", err)
		}
	}

	if len(chainConfig.CustomConsensusConfig) > 0 {
		if err := n.ModifyTomlConfigFile(
			ctx,
			"config/config.toml",
			chainConfig.CustomConsensusConfig,
		); err != nil {
			return fmt.Errorf("failed to apply custom consensus config: %w", err)
		}
	}

	if len(chainConfig.CustomClientConfig) > 0 {
		if err := n.ModifyTomlConfigFile(
			ctx,
			"config/client.toml",
			chainConfig.CustomClientConfig,
		); err != nil {
			return fmt.Errorf("failed to apply custom client config: %w", err)
		}
	}

	return nil
}

// SetPersistentPeers will set the node's persistent peers in the CometBFT config
func (n *Node) SetPersistentPeers(ctx context.Context, peers string) error {
	cometBftConfig := make(map[string]interface{})

	p2pConfig := make(map[string]interface{})
	p2pConfig["persistent_peers"] = peers

	cometBftConfig["p2p"] = p2pConfig

	return n.ModifyTomlConfigFile(
		ctx,
		"config/config.toml",
		cometBftConfig,
	)
}

// SetLibP2PBootstrapPeers will set the node's libp2p bootstrap peers in the CometBFT config
// @see https://github.com/cometbft/cometbft/blob/6837f04ce6c122a1c575f5281c8ba171df8dd9d4/config/config.go#L631
func (n *Node) SetLibP2PBootstrapPeers(ctx context.Context, peers []map[string]any) error {
	// sanity checks
	for _, bp := range peers {
		if _, ok := bp["host"]; !ok {
			return fmt.Errorf("bootstrap peer missing host: %v", bp)
		}
		if _, ok := bp["id"]; !ok {
			return fmt.Errorf("bootstrap peer missing id: %v", bp)
		}
	}

	consensusSubConfig := map[string]any{
		"p2p": map[string]any{
			"libp2p": map[string]any{
				"bootstrap_peers": peers,
			},
		},
	}

	return n.ModifyTomlConfigFile(
		ctx,
		"config/config.toml",
		consensusSubConfig,
	)
}

// SetSeedNode will set a given node as seed for the network
func (n *Node) SetSeedNode(ctx context.Context, seedNode string) error {
	cometBftConfig := make(map[string]interface{})

	p2pConfig := make(map[string]interface{})
	p2pConfig["seeds"] = seedNode

	cometBftConfig["p2p"] = p2pConfig

	return n.ModifyTomlConfigFile(
		ctx,
		"config/config.toml",
		cometBftConfig,
	)
}

// SetSeedMode will configure this node to operate in seed mode
func (n *Node) SetSeedMode(ctx context.Context) error {
	cometBftConfig := make(map[string]interface{})

	p2pConfig := make(map[string]interface{})
	p2pConfig["seed_mode"] = true
	p2pConfig["seeds"] = ""

	cometBftConfig["p2p"] = p2pConfig

	return n.ModifyTomlConfigFile(
		ctx,
		"config/config.toml",
		cometBftConfig,
	)
}

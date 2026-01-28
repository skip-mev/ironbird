package types

import (
	"context"
	"fmt"
	"math/big"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"google.golang.org/grpc"

	"github.com/skip-mev/ironbird/petri/core/provider"
)

var (
	validRegions = map[string]bool{
		"nyc1": true,
		"sfo2": true,
		"ams3": true,
		"fra1": true,
		"sgp1": true,
	}
)

const (
	DigitalOcean = "DigitalOcean"
	Docker       = "Docker"
)

// GenesisModifier is a function that takes in genesis bytes and returns modified genesis bytes
type GenesisModifier func([]byte) ([]byte, error)

// RegionConfig defines the number of validators and nodes for a specific region
type RegionConfig struct {
	Name          string `json:"name"`
	NumValidators int    `json:"num_validators"`
	NumNodes      int    `json:"num_nodes"`
}

func (r RegionConfig) ValidateBasic() error {
	if r.Name == "" {
		return fmt.Errorf("region cannot be empty")
	}

	if !validRegions[r.Name] {
		return fmt.Errorf("region '%s' is not supported. Valid regions are: nyc1, sfo2, ams3, fra1, sgp1", r.Name)
	}

	if r.NumValidators < 0 {
		return fmt.Errorf("num validators cannot be negative")
	}
	if r.NumNodes < 0 {
		return fmt.Errorf("num nodes cannot be negative")
	}
	return nil
}

// ChainI is an interface for a logical chain
type ChainI interface {
	Init(context.Context, ChainOptions) error
	Teardown(context.Context) error

	GetConfig() ChainConfig
	GetGRPCClient(context.Context) (*grpc.ClientConn, error)
	GetTMClient(context.Context) (*rpchttp.HTTP, error)

	GetValidators() []NodeI
	GetFaucetWallet() WalletI
	GetValidatorWallets() []WalletI

	GetNodes() []NodeI

	Height(context.Context) (uint64, error)
	WaitForBlocks(ctx context.Context, delta uint64) error
	WaitForHeight(ctx context.Context, desiredHeight uint64) error

	Serialize(ctx context.Context, p provider.ProviderI) ([]byte, error)
}

type ChainOptions struct {
	ModifyGenesis GenesisModifier // ModifyGenesis is a function that modifies the genesis bytes of the chain
	NodeOptions   NodeOptions     // NodeOptions is the options for creating a node
	NodeCreator   NodeCreator     // NodeCreator is a function that creates a node

	WalletConfig       WalletConfig // WalletConfig is the default configuration of a chain's wallet
	AdditionalAccounts int
	BaseMnemonic       string
}

func (o ChainOptions) ValidateBasic() error {
	if err := o.WalletConfig.ValidateBasic(); err != nil {
		return fmt.Errorf("wallet config is invalid: %w", err)
	}

	if o.NodeCreator == nil {
		return fmt.Errorf("node creator cannot be nil")
	}

	return nil
}

// ChainConfig is the configuration structure for a logical chain.
// It contains all the relevant details needed to create a Cosmos chain
type ChainConfig struct {
	Name          string
	Denom         string // Denom is the denomination of the native staking token
	Decimals      uint64 // Decimals is the number of decimals of the native staking token
	NumValidators int    // NumValidators is the number of validators to create for Docker deployments
	NumNodes      int    // NumNodes is the number of nodes to create for Docker deployments

	// RegionConfig defines how validators and nodes should be distributed across regions for DigitalOcean deployments
	RegionConfig []RegionConfig

	BinaryName string   // BinaryName is the name of the chain binary in the Docker image
	Entrypoint []string // Entrypoint is the list of arguments to invoke in the entrypoint of the Docker image

	Image provider.ImageDefinition // Image is the Docker ImageDefinition of the chain

	GasPrices string // GasPrices are the minimum gas prices to set on the chain

	Bech32Prefix string // Bech32Prefix is the Bech32 prefix of the on-chain addresses

	HomeDir string // HomeDir is the home directory of the chain

	CoinType string // CoinType is the coin type of the chain (e.g. 118)
	ChainId  string // ChainId is the chain ID of the chain

	UseGenesisSubCommand bool     // UseGenesisSubCommand is a flag that indicates whether to use the 'genesis' subcommand to initialize the chain. Set to true if Cosmos SDK >v0.50
	AdditionalStartFlags []string // AdditionalStartFlags are additional flags to pass to the chain binary when starting the chain

	AdditionalPorts []string // AdditionalPorts are additional ports to expose for the chain

	// number of tokens to allocate per account in the genesis state (unscaled). This value defaults to 10_000_000 if not set.
	// if not set.
	GenesisDelegation *big.Int
	// number of tokens to allocate to the genesis account. This value defaults to 5_000_000 if not set.
	GenesisBalance *big.Int

	IsEVMChain            bool                   // IsEVMChain is used to set evm specific configs during chain creation
	CustomAppConfig       map[string]interface{} // CustomAppConfig is the configuration for the chain's app.toml
	CustomClientConfig    map[string]interface{} // CustomClientConfig is the configuration for the chain's client.toml
	CustomConsensusConfig map[string]interface{} // CustomConsensusConfig is the configuration for the chain's config.toml

	// SetPersistentPeers is used to determine whether nodes and validators of the network are added as persistent
	// peers to the consensus config
	SetPersistentPeers bool
	// SetPersistentPeers is used to determine whether a seed node is added to the consensus config
	SetSeedNode bool
}

func (c ChainConfig) GetGenesisBalance() *big.Int {
	if c.GenesisBalance == nil {
		return big.NewInt(100_000_000)
	}
	return c.GenesisBalance
}

func (c ChainConfig) GetGenesisDelegation() *big.Int {
	if c.GenesisDelegation == nil {
		return big.NewInt(5_000_000)
	}
	return c.GenesisDelegation
}

func (c ChainConfig) ValidateBasic(providerType string) error {
	if c.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if c.Denom == "" {
		return fmt.Errorf("denom cannot be empty")
	}

	if c.Decimals == 0 {
		return fmt.Errorf("decimals cannot be 0")
	}

	if providerType == DigitalOcean {
		if len(c.RegionConfig) == 0 {
			return fmt.Errorf("regional distribution cannot be empty")
		}

		hasValidators := false
		for i, region := range c.RegionConfig {
			if err := region.ValidateBasic(); err != nil {
				return fmt.Errorf("regional config %d is invalid: %w", i, err)
			}
			if !hasValidators && region.NumValidators > 0 {
				hasValidators = true
			}
		}

		if !hasValidators {
			return fmt.Errorf("at least one region must have validators")
		}
	} else if providerType == Docker {
		if c.NumValidators == 0 {
			return fmt.Errorf("num validators cannot be 0")
		}
	} else {
		return fmt.Errorf("invalid provider type: %s", providerType)
	}

	if c.BinaryName == "" {
		return fmt.Errorf("binary name cannot be empty")
	}

	if c.GasPrices == "" {
		return fmt.Errorf("gas prices cannot be empty")
	}

	if err := c.Image.ValidateBasic(); err != nil {
		return fmt.Errorf("image definition is invalid: %w", err)
	}

	if c.Bech32Prefix == "" {
		return fmt.Errorf("bech32 prefix cannot be empty")
	}

	if c.CoinType == "" {
		return fmt.Errorf("coin type cannot be empty")
	}

	if c.ChainId == "" {
		return fmt.Errorf("chain ID cannot be empty")
	}

	return nil
}

// UseLibP2P returns true if libp2p is enabled in the custom consensus config.
// Bootstrap peers are now configured inline in config.toml under [p2p.libp2p].
func (c ChainConfig) UseLibP2P() bool {
	if len(c.CustomConsensusConfig) == 0 {
		return false
	}

	if _, ok := c.CustomConsensusConfig["p2p"]; !ok {
		return false
	}

	p2p, ok := c.CustomConsensusConfig["p2p"].(map[string]any)
	if !ok {
		return false
	}

	// https://github.com/cometbft/cometbft/blob/main/config/config.go LibP2PConfig
	libp2p, ok := p2p["libp2p"].(map[string]any)
	if !ok {
		return false
	}

	enabled, ok := libp2p["enabled"].(bool)
	if !ok {
		return false
	}

	return enabled
}

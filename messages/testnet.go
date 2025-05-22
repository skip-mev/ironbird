package messages

import (
	"fmt"
	"time"

	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/types/testnet"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
)

type CreateProviderRequest struct {
	RunnerType testnet.RunnerType
	Name       string
}

type CreateProviderResponse struct {
	ProviderState []byte
}

type TeardownProviderRequest struct {
	RunnerType    testnet.RunnerType
	ProviderState []byte
}

type TeardownProviderResponse struct{}

type LaunchTestnetRequest struct {
	Name                    string
	Image                   string
	UID                     string
	GID                     string
	BinaryName              string
	HomeDir                 string
	ProviderSpecificOptions map[string]string
	GenesisModifications    []petrichain.GenesisKV
	RunnerType              testnet.RunnerType

	NumOfValidators uint64
	NumOfNodes      uint64

	ProviderState []byte
}

type LaunchTestnetResponse struct {
	ProviderState []byte
	ChainState    []byte
	ChainID       string
	Nodes         []testnet.Node
}

type TestnetWorkflowRequest struct {
	Repo               string
	SHA                string
	ChainConfig        types.ChainsConfig
	RunnerType         testnet.RunnerType
	LoadTestSpec       *catalysttypes.LoadTestSpec
	GrafanaConfig      types.GrafanaConfig
	LongRunningTestnet bool
	TestnetDuration    time.Duration
}

func (r TestnetWorkflowRequest) Validate() error {
	if r.Repo == "" {
		return fmt.Errorf("repo is required")
	}

	if r.SHA == "" {
		return fmt.Errorf("SHA is required")
	}

	if r.ChainConfig.Name == "" {
		return fmt.Errorf("chain name is required")
	}

	if r.ChainConfig.Image.BinaryName == "" {
		return fmt.Errorf("binary name is required")
	}

	if r.ChainConfig.Image.HomeDir == "" {
		return fmt.Errorf("home directory is required")
	}

	if r.RunnerType != testnet.DigitalOcean && r.RunnerType != testnet.Docker {
		return fmt.Errorf("runner type must be one of: %s, %s", testnet.DigitalOcean, testnet.Docker)
	}

	if r.LongRunningTestnet && r.TestnetDuration > 0 {
		return fmt.Errorf("can not set duration on long-running testnet")
	}

	return nil
}

type TestnetWorkflowResponse string

package messages

import (
	"fmt"
	"time"

	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/types"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
)

const (
	DigitalOcean RunnerType = "DigitalOcean"
	Docker       RunnerType = "Docker"
	TaskQueue               = "TESTNET_TASK_QUEUE"
)

type RunnerType string

type Node struct {
	Name    string
	Address string
	RPC     string
	LCD     string
}

type CreateProviderRequest struct {
	RunnerType RunnerType
	Name       string
}

type CreateProviderResponse struct {
	ProviderState []byte
}

type TeardownProviderRequest struct {
	RunnerType    RunnerType
	ProviderState []byte
}

type TeardownProviderResponse struct{}

type LaunchTestnetRequest struct {
	Name                    string
	GaiaEVM                 bool
	Repo                    string
	SHA                     string
	Image                   string
	ProviderSpecificOptions map[string]string
	GenesisModifications    []petrichain.GenesisKV
	RunnerType              RunnerType

	NumOfValidators uint64
	NumOfNodes      uint64

	ProviderState []byte
}

type LaunchTestnetResponse struct {
	ProviderState []byte
	ChainState    []byte
	ChainID       string
	Nodes         []Node
	Validators    []Node
}

type TestnetWorkflowRequest struct {
	Repo               string
	SHA                string
	GaiaEVM            bool
	ChainConfig        types.ChainsConfig
	RunnerType         RunnerType
	LoadTestSpec       *catalysttypes.LoadTestSpec
	LongRunningTestnet bool
	TestnetDuration    time.Duration
	NumWallets         int
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

	if r.RunnerType != DigitalOcean && r.RunnerType != Docker {
		return fmt.Errorf("runner type must be one of: %s, %s", DigitalOcean, Docker)
	}

	if r.LongRunningTestnet && r.TestnetDuration > 0 {
		return fmt.Errorf("can not set duration on long-running testnet")
	}

	return nil
}

type TestnetWorkflowResponse string

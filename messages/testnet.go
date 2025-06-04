package messages

import (
	"fmt"
	pb "github.com/skip-mev/ironbird/server/proto"
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

var (
	DigitalOceanDefaultOpts = map[string]string{"region": "nyc1", "size": "s-4vcpu-8gb",
		"image_id": "185517855"}
)

type RunnerType string

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
	Evm                     bool
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
	Nodes         []pb.Node
	Validators    []pb.Node
}

type TestnetWorkflowRequest struct {
	Repo               string
	SHA                string
	Evm                bool
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

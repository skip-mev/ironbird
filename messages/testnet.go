package messages

import (
	"fmt"

	ctlttypes "github.com/skip-mev/catalyst/chains/types"
	pb "github.com/skip-mev/ironbird/server/proto"
	"github.com/skip-mev/ironbird/types"
	petritypes "github.com/skip-mev/petri/core/v3/types"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
)

const (
	DigitalOcean RunnerType = "DigitalOcean"
	Docker       RunnerType = "Docker"
	TaskQueue               = "TESTNET_TASK_QUEUE"
)

var (
	DigitalOceanDefaultOpts = map[string]string{"region": "nyc1", "size": "s-4vcpu-8gb",
		"image_id": "197014352"}
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
	Name                 string
	IsEvmChain           bool
	Repo                 string
	SHA                  string
	Image                string // tag of image e.g.  public.ecr.aws/n7v2p5f8/skip-mev/ironbird-local:gaia-evmv23.3.0-gaia-b84ff4c1702d3cc7756209a6de81ab95b3e6c6e5
	BaseImage            string // base image used e.g. simapp-v53, gaia (defined in worker.yaml chains map)
	GenesisModifications []petrichain.GenesisKV
	RunnerType           RunnerType

	RegionConfigs   []petritypes.RegionConfig
	NumOfValidators uint64
	NumOfNodes      uint64

	CustomAppConfig       map[string]interface{}
	CustomConsensusConfig map[string]interface{}
	CustomClientConfig    map[string]interface{}

	ProviderState      []byte
	SetPersistentPeers bool
	SetSeedNode        bool
}

type LaunchTestnetResponse struct {
	ProviderState []byte
	ChainState    []byte
	ChainID       string
	Nodes         []*pb.Node
	Validators    []*pb.Node
}

type TestnetWorkflowRequest struct {
	Repo        string
	SHA         string
	IsEvmChain  bool
	ChainConfig types.ChainsConfig
	RunnerType  RunnerType

	EthereumLoadTestSpec *ctlttypes.LoadTestSpec
	CosmosLoadTestSpec   *ctlttypes.LoadTestSpec

	LongRunningTestnet bool
	LaunchLoadBalancer bool
	TestnetDuration    string
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

	if r.LongRunningTestnet && r.TestnetDuration != "" {
		return fmt.Errorf("can not set duration on long-running testnet")
	}

	if !r.ChainConfig.SetSeedNode && !r.ChainConfig.SetPersistentPeers {
		return fmt.Errorf("at least one of SetSeedNode or SetPersistentPeers must be set to true")
	}

	if r.RunnerType == Docker && r.LaunchLoadBalancer {
		return fmt.Errorf("load balancer is not supported for docker runners")
	}

	if r.EthereumLoadTestSpec != nil && r.CosmosLoadTestSpec != nil {
		return fmt.Errorf("only one of ethereum of cosmos load test can be specified")
	}

	if r.IsEvmChain {
		if r.CosmosLoadTestSpec != nil {
			return fmt.Errorf("can not run cosmos load tests for evm chain")
		}
	} else {
		if r.EthereumLoadTestSpec != nil {
			return fmt.Errorf("can not run ethereum load tests for cosmos chain")
		}
	}

	return nil
}

type TestnetWorkflowResponse string

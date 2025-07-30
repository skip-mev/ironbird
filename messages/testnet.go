package messages

import (
	"fmt"
	pb "github.com/skip-mev/ironbird/server/proto"

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
	DigitalOceanDefaultOpts = []map[string]string{
		{
			"region":   "nyc1",
			"size":     "s-4vcpu-8gb",
			"image_id": "194382907",
		},
		{
			"region":   "sfo2",
			"size":     "s-4vcpu-8gb",
			"image_id": "194382907",
		},
		{
			"region":   "ams3",
			"size":     "s-4vcpu-8gb",
			"image_id": "194382907",
		},
		{
			"region":   "fra1",
			"size":     "s-4vcpu-8gb",
			"image_id": "194382907",
		},
		{
			"region":   "sgp1",
			"size":     "s-4vcpu-8gb",
			"image_id": "194382907",
		},
	}
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
	IsEvmChain              bool
	Repo                    string
	SHA                     string
	Image                   string // tag of image e.g.  public.ecr.aws/n7v2p5f8/skip-mev/ironbird-local:gaia-evmv23.3.0-gaia-b84ff4c1702d3cc7756209a6de81ab95b3e6c6e5
	BaseImage               string // base image used e.g. simapp-v53, gaia (defined in worker.yaml chains map)
	ProviderSpecificOptions map[string]string
	GenesisModifications    []petrichain.GenesisKV
	RunnerType              RunnerType

	NumOfValidators uint64
	NumOfNodes      uint64

	CustomAppConfig       map[string]interface{}
	CustomConsensusConfig map[string]interface{}
	CustomClientConfig    map[string]interface{}

	ProviderState []byte
}

type LaunchTestnetResponse struct {
	ProviderState []byte
	ChainState    []byte
	ChainID       string
	Nodes         []*pb.Node
	Validators    []*pb.Node
}

type TestnetWorkflowRequest struct {
	Repo               string
	SHA                string
	IsEvmChain         bool
	ChainConfig        types.ChainsConfig
	RunnerType         RunnerType
	LoadTestSpec       *catalysttypes.LoadTestSpec
	LongRunningTestnet bool
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

	return nil
}

type TestnetWorkflowResponse string

package testnet

import (
	"context"
	"fmt"

	ethtypes "github.com/skip-mev/catalyst/chains/ethereum/types"

	petritypes "github.com/skip-mev/ironbird/petri/core/types"

	"github.com/skip-mev/ironbird/activities/loadbalancer"
	petriutil "github.com/skip-mev/ironbird/petri/core/util"

	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"

	"github.com/aws/aws-sdk-go-v2/config"
	cosmostypes "github.com/skip-mev/catalyst/chains/cosmos/types"
	catalysttypes "github.com/skip-mev/catalyst/chains/types"
	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/loadtest"
	testnettypes "github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"
	petrichain "github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/skip-mev/ironbird/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type TestnetWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

var (
	simappReq = messages.TestnetWorkflowRequest{
		TestnetDuration: "1m",
		CatalystVersion: "technicallyty-refactor-wallets",
		ChainConfig: types.ChainsConfig{
			Name:  "stake-1",
			Image: "simapp",
			GenesisModifications: []petrichain.GenesisKV{
				{
					Key:   "consensus.params.block.max_gas",
					Value: "75000000",
				},
			},
			NumOfValidators:    1,
			NumOfNodes:         1,
			SetPersistentPeers: true,
		},
		CosmosLoadTestSpec: &catalysttypes.LoadTestSpec{
			Kind:        "cosmos",
			Name:        "e2e-test",
			ChainID:     "stake-1",
			Description: "e2e test",
			NumOfBlocks: 5,
			NumOfTxs:    100,
			Msgs: []catalysttypes.LoadTestMsg{
				{Weight: 1, Type: cosmostypes.MsgSend},
			},
		},
		BaseMnemonic: "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom",
		NumWallets:   20,
	}
	evmReq = messages.TestnetWorkflowRequest{
		TestnetDuration: "20m",
		Repo:            "evm",
		CatalystVersion: "technicallyty-refactor-wallets",
		IsEvmChain:      true,
		SHA:             "2d3df2ba510c978d785f2151132e9ed70e1605ec",
		RunnerType:      messages.Docker,
		BaseMnemonic:    "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom",
		NumWallets:      1500,
		ChainConfig: types.ChainsConfig{
			Name:  "evmd",
			Image: "evm",
			GenesisModifications: []petrichain.GenesisKV{
				{
					Key:   "app_state.staking.params.bond_denom",
					Value: "atest",
				},
				{
					Key:   "app_state.gov.params.expedited_voting_period",
					Value: "120s",
				},
				{
					Key:   "app_state.gov.params.voting_period",
					Value: "300s",
				},
				{
					Key:   "app_state.gov.params.expedited_min_deposit.0.amount",
					Value: "1",
				},
				{
					Key:   "app_state.gov.params.expedited_min_deposit.0.denom",
					Value: "atest",
				},
				{
					Key:   "app_state.gov.params.min_deposit.0.amount",
					Value: "1",
				},
				{
					Key:   "app_state.gov.params.min_deposit.0.denom",
					Value: "atest",
				},
				{
					Key:   "app_state.evm.params.evm_denom",
					Value: "atest",
				},
				{
					Key:   "app_state.mint.params.mint_denom",
					Value: "atest",
				},
				{
					Key: "app_state.bank.denom_metadata",
					Value: []map[string]interface{}{
						{
							"description": "The native staking token for evmd.",
							"denom_units": []map[string]interface{}{
								{
									"denom":    "atest",
									"exponent": 0,
									"aliases":  []string{"attotest"},
								},
								{
									"denom":    "test",
									"exponent": 18,
									"aliases":  []string{},
								},
							},
							"base":     "atest",
							"display":  "test",
							"name":     "Test Token",
							"symbol":   "TEST",
							"uri":      "",
							"uri_hash": "",
						},
					},
				},
				{
					Key: "app_state.evm.params.active_static_precompiles",
					Value: []string{
						"0x0000000000000000000000000000000000000100",
						"0x0000000000000000000000000000000000000400",
						"0x0000000000000000000000000000000000000800",
						"0x0000000000000000000000000000000000000801",
						"0x0000000000000000000000000000000000000802",
						"0x0000000000000000000000000000000000000803",
						"0x0000000000000000000000000000000000000804",
						"0x0000000000000000000000000000000000000805",
					},
				},
				{
					Key: "app_state.erc20.token_pairs",
					Value: []map[string]interface{}{
						{
							"contract_owner": 1,
							"erc20_address":  "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
							"denom":          "atest",
							"enabled":        true,
						},
					},
				},
				{
					Key:   "consensus.params.block.max_gas",
					Value: "75000000",
				},
			},
			NumOfValidators:    1,
			NumOfNodes:         0,
			SetPersistentPeers: true,
		},
		EthereumLoadTestSpec: &catalysttypes.LoadTestSpec{
			Kind:        "eth",
			Name:        "e2e-test",
			ChainID:     "262144",
			Description: "e2e test",
			NumOfBlocks: 5,
			NumOfTxs:    100,
			Msgs: []catalysttypes.LoadTestMsg{
				{NumMsgs: 20, Type: ethtypes.MsgCreateContract},
			},
		},
	}
)

func (s *TestnetWorkflowTestSuite) SetupTest() {
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=buildkitd", "--format", "{{.Names}}")
	output, err := cmd.CombinedOutput()
	if err == nil && !strings.Contains(string(output), "buildkitd") {
		buildkitCmd := exec.Command("docker", "run", "-d", "--name", "buildkitd", "--privileged",
			"-p", "1234:1234",
			"-v", "/var/run/docker.sock:/var/run/docker.sock",
			"-v", "buildkitd:/var/lib/buildkit",
			"-v", os.Getenv("HOME")+"/.docker/config.json:/root/.docker/config.json",
			"moby/buildkit:latest",
			"--addr", "tcp://0.0.0.0:1234")

		if buildkitOutput, buildkitErr := buildkitCmd.CombinedOutput(); buildkitErr != nil {
			log.Printf("Failed to start buildkitd container: %v\nOutput: %s", buildkitErr, buildkitOutput)
		} else {
			log.Println("Started buildkitd container successfully")
		}
	} else if strings.Contains(string(output), "buildkitd") {
		log.Println("buildkitd container already exists")
	}

	cosmostypes.Register()
	ethtypes.Register()
	s.env = s.NewTestWorkflowEnvironment()
	s.env.SetTestTimeout(2 * time.Hour)
}

func (s *TestnetWorkflowTestSuite) AfterTest() {
	s.env.AssertExpectations(s.T())
}

func (s *TestnetWorkflowTestSuite) TearDownSuite() {
	stopCmd := exec.Command("docker", "stop", "buildkitd")
	if _, err := stopCmd.CombinedOutput(); err != nil {
		log.Printf("Failed to stop buildkitd container: %v", err)
	}

	rmCmd := exec.Command("docker", "rm", "buildkitd")
	if _, err := rmCmd.CombinedOutput(); err != nil {
		log.Printf("Failed to remove buildkitd container: %v", err)
	} else {
		log.Println("Successfully cleaned up buildkitd container")
	}
}

func (s *TestnetWorkflowTestSuite) setupMockActivitiesDocker() {
	cfg, err := types.ParseWorkerConfig("../../conf/worker.yaml")
	if err != nil {
		s.T().Fatal(err)
	}
	testnetActivity := &testnettypes.Activity{
		Chains:       cfg.Chains,
		RegistryType: "local",
	}
	s.env.RegisterActivity(testnetActivity.CreateProvider)
	s.env.RegisterActivity(testnetActivity.TeardownProvider)
	s.env.RegisterActivity(testnetActivity.LaunchTestnet)

	loadTestActivity := &loadtest.Activity{}
	s.env.RegisterActivity(loadTestActivity.RunLoadTest)

	builderConfig := types.BuilderConfig{
		BuildKitAddress: "tcp://localhost:1234",
		Local: types.LocalRegistryConfig{
			ImageName: "ironbird",
		},
	}
	builderActivity := &builder.Activity{
		BuilderConfig: builderConfig,
		Registry: types.RegistryConfig{
			Type:      "local",
			ImageName: "ironbird",
		},
	}
	s.env.RegisterActivity(builderActivity.BuildDockerImage)

	testnetActivities = testnetActivity
	loadTestActivities = loadTestActivity

	s.env.OnActivity(loadTestActivity.RunLoadTest, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.RunLoadTestRequest) (messages.RunLoadTestResponse, error) {
			return loadTestActivities.RunLoadTest(ctx, req)
		})

	s.env.OnActivity(testnetActivity.CreateProvider, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.CreateProviderRequest) (messages.CreateProviderResponse, error) {
			return testnetActivity.CreateProvider(ctx, req)
		})

	s.env.OnActivity(testnetActivity.LaunchTestnet, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.LaunchTestnetRequest) (messages.LaunchTestnetResponse, error) {
			return testnetActivity.LaunchTestnet(ctx, req)
		})

	s.env.OnActivity(testnetActivity.TeardownProvider, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.TeardownProviderRequest) (messages.TeardownProviderResponse, error) {
			return testnetActivity.TeardownProvider(ctx, req)
		})

	s.env.OnActivity(builderActivity.BuildDockerImage, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.BuildDockerImageRequest) (messages.BuildDockerImageResponse, error) {
			if req.Repo == "evm" {
				evmTag := "public.ecr.aws/n7v2p5f8/skip-mev/ironbird-local:test-1main-evm-e01cc5077dc05796362af724fe0c9281c478f94a"

				cmd := exec.Command("docker", "pull", evmTag)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return messages.BuildDockerImageResponse{}, fmt.Errorf("failed to pull EVM Docker image: %w, output: %s", err, output)
				}

				return messages.BuildDockerImageResponse{
					FQDNTag: evmTag,
					Logs:    output,
				}, nil
			} else {
				originalTag := "ghcr.io/cosmos/simapp:v0.50"
				newTag := "simapp-v53"

				cmd := exec.Command("docker", "pull", originalTag)
				output, err := cmd.CombinedOutput()
				if err != nil {
					return messages.BuildDockerImageResponse{}, err
				}

				tagCmd := exec.Command("docker", "tag", originalTag, newTag)
				tagOutput, err := tagCmd.CombinedOutput()
				if err != nil {
					return messages.BuildDockerImageResponse{}, err
				}

				return messages.BuildDockerImageResponse{
					FQDNTag: newTag,
					Logs:    append(output, tagOutput...),
				}, nil
			}
		})
}

func (s *TestnetWorkflowTestSuite) setupMockActivitiesDigitalOcean() {
	ctx := context.Background()
	doToken := os.Getenv("DIGITALOCEAN_TOKEN")

	nodeAuthKey := os.Getenv("TS_NODE_AUTH_KEY")
	tsServerOauthSecret := os.Getenv("TS_SERVER_OAUTH_SECRET")
	tailscaleSettings, err := digitalocean.SetupTailscale(ctx, tsServerOauthSecret,
		nodeAuthKey, "ironbird-tests", []string{"ironbird-e2e"}, []string{"ironbird-e2e"})
	if err != nil {
		panic(err)
	}

	awsConfig, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		log.Fatalln(err)
	}

	cfg, err := types.ParseWorkerConfig("../../conf/worker.yaml")
	if err != nil {
		s.T().Fatal(err)
	}
	testnetActivity := &testnettypes.Activity{
		DOToken:           doToken,
		TailscaleSettings: tailscaleSettings,
		Chains:            cfg.Chains,
		AwsConfig:         &awsConfig,
		RegistryType:      "ecr",
	}
	loadBalancerActivity := &loadbalancer.Activity{
		RootDomain:        "ib-local.dev.skip.build",
		DOToken:           doToken,
		TailscaleSettings: tailscaleSettings,
	}

	s.env.RegisterActivity(testnetActivity.CreateProvider)
	s.env.RegisterActivity(testnetActivity.TeardownProvider)
	s.env.RegisterActivity(testnetActivity.LaunchTestnet)
	s.env.RegisterActivity(loadBalancerActivity.LaunchLoadBalancer)

	loadTestActivity := &loadtest.Activity{
		DOToken:           doToken,
		TailscaleSettings: tailscaleSettings,
	}
	s.env.RegisterActivity(loadTestActivity.RunLoadTest)

	builderConfig := types.BuilderConfig{
		BuildKitAddress: "tcp://localhost:1234",
		ECR: types.ECRRegistryConfig{
			URL:       "public.ecr.aws",
			ImageName: "skip-mev/n7v2p5f8/n7v2p5f8/skip-mev/ironbird-local",
		},
	}

	builderActivity := builder.Activity{
		BuilderConfig: builderConfig,
		AwsConfig:     &awsConfig,
		Registry: types.RegistryConfig{
			Type:      "ecr",
			URL:       "public.ecr.aws",
			ImageName: "skip-mev/n7v2p5f8/n7v2p5f8/skip-mev/ironbird-local",
		},
	}
	s.env.RegisterActivity(builderActivity.BuildDockerImage)

	testnetActivities = testnetActivity
	loadTestActivities = loadTestActivity
	loadBalancerActivities = loadBalancerActivity

	s.env.OnActivity(loadTestActivity.RunLoadTest, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.RunLoadTestRequest) (messages.RunLoadTestResponse, error) {
			return loadTestActivities.RunLoadTest(ctx, req)
		})

	s.env.OnActivity(builderActivity.BuildDockerImage, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.BuildDockerImageRequest) (messages.BuildDockerImageResponse, error) {
			tag := "ghcr.io/cosmos/simapp:v0.50"
			cmd := exec.Command("docker", "pull", tag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return messages.BuildDockerImageResponse{}, err
			}

			return messages.BuildDockerImageResponse{
				FQDNTag: tag,
				Logs:    output,
			}, nil
		})

	s.env.OnActivity(loadBalancerActivity.LaunchLoadBalancer, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.LaunchLoadBalancerRequest) (messages.LaunchLoadBalancerResponse, error) {
			return loadBalancerActivities.LaunchLoadBalancer(ctx, req)
		})

	s.env.OnActivity(testnetActivity.TeardownProvider, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.TeardownProviderRequest) (messages.TeardownProviderResponse, error) {
			return testnetActivity.TeardownProvider(ctx, req)
		})
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowDocker() {
	s.setupMockActivitiesDocker()

	// use sdk repo here to test skipping replace workflow
	dockerReq := simappReq
	dockerReq.Repo = "cosmos-sdk"
	dockerReq.SHA = "acb1d65cdc1e0fc36d93f3c5bb6aaf919a1321e2"
	dockerReq.RunnerType = messages.Docker
	dockerReq.ChainConfig.Name = "stake"
	dockerReq.CatalystVersion = "technicallyty-refactor-wallets"
	dockerReq.BaseMnemonic = "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"
	dockerReq.NumWallets = 1500

	s.env.ExecuteWorkflow(Workflow, dockerReq)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.env.AssertActivityNumberOfCalls(s.T(), "RunLoadTest", 1)
	s.env.AssertActivityNumberOfCalls(s.T(), "TeardownProvider", 1)
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowDigitalOcean() {
	s.setupMockActivitiesDigitalOcean()

	// use cometbft repo here to test replace workflow
	doReq := simappReq
	doReq.Repo = "cometbft"
	doReq.SHA = "54602e1c1a39943c2d460c23757e33f1d24f85f0"
	doReq.RunnerType = messages.DigitalOcean
	doReq.ChainConfig.Name = fmt.Sprintf("stake-%s", petriutil.RandomString(3))
	doReq.CatalystVersion = "technicallyty-refactor-wallets"
	doReq.LaunchLoadBalancer = false
	doReq.BaseMnemonic = "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"
	doReq.NumWallets = 1500
	doReq.ChainConfig.RegionConfigs = []petritypes.RegionConfig{
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
	}

	s.env.ExecuteWorkflow(Workflow, doReq)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.env.AssertActivityNumberOfCalls(s.T(), "RunLoadTest", 1)
	s.env.AssertActivityNumberOfCalls(s.T(), "TeardownProvider", 1)
	s.env.AssertActivityNumberOfCalls(s.T(), "LaunchLoadBalancer", 0)
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowDigitalOceanWithLoadBalancer() {
	s.setupMockActivitiesDigitalOcean()

	doReq := simappReq
	doReq.Repo = "cometbft"
	doReq.SHA = "54602e1c1a39943c2d460c23757e33f1d24f85f0"
	doReq.RunnerType = messages.DigitalOcean
	doReq.ChainConfig.Name = fmt.Sprintf("stake-%s", petriutil.RandomString(3))
	doReq.CatalystVersion = "technicallyty-refactor-wallets"
	doReq.LaunchLoadBalancer = true
	doReq.ChainConfig.RegionConfigs = []petritypes.RegionConfig{
		{
			Name:          "nyc1",
			NumValidators: 1,
			NumNodes:      0,
		},
	}
	doReq.BaseMnemonic = "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"
	doReq.NumWallets = 1500

	s.env.ExecuteWorkflow(Workflow, doReq)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.env.AssertActivityNumberOfCalls(s.T(), "RunLoadTest", 1)
	s.env.AssertActivityNumberOfCalls(s.T(), "TeardownProvider", 1)
	s.env.AssertActivityNumberOfCalls(s.T(), "LaunchLoadBalancer", 1)
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowEVM() {
	s.setupMockActivitiesDocker()

	s.env.ExecuteWorkflow(Workflow, evmReq)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.env.AssertActivityNumberOfCalls(s.T(), "RunLoadTest", 1)
	s.env.AssertActivityNumberOfCalls(s.T(), "TeardownProvider", 1)
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowCustomDurationNoLoadTest() {
	s.setupMockActivitiesDocker()

	dockerReq := simappReq
	dockerReq.Repo = "cosmos-sdk"
	dockerReq.SHA = "acb1d65cdc1e0fc36d93f3c5bb6aaf919a1321e2"
	dockerReq.RunnerType = messages.Docker
	dockerReq.ChainConfig.Name = "stake"
	dockerReq.CatalystVersion = "technicallyty-refactor-wallets"
	dockerReq.CosmosLoadTestSpec = nil
	dockerReq.LongRunningTestnet = false
	dockerReq.TestnetDuration = ""
	dockerReq.BaseMnemonic = "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"
	dockerReq.NumWallets = 1500

	s.env.ExecuteWorkflow(Workflow, dockerReq)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.env.AssertActivityNumberOfCalls(s.T(), "RunLoadTest", 0)
	s.env.AssertActivityNumberOfCalls(s.T(), "TeardownProvider", 1)
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowLongRunningCancelled() {
	s.setupMockActivitiesDocker()

	dockerReq := simappReq
	dockerReq.Repo = "cosmos-sdk"
	dockerReq.SHA = "acb1d65cdc1e0fc36d93f3c5bb6aaf919a1321e2"
	dockerReq.RunnerType = messages.Docker
	dockerReq.ChainConfig.Name = "stake"
	dockerReq.CatalystVersion = "technicallyty-refactor-wallets"
	dockerReq.LongRunningTestnet = true
	dockerReq.TestnetDuration = ""
	dockerReq.BaseMnemonic = "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"
	dockerReq.NumWallets = 1500

	done := make(chan struct{})
	s.env.RegisterDelayedCallback(func() {
		s.env.CancelWorkflow()
		time.Sleep(5 * time.Second)
		close(done)
	}, 15*time.Second)

	s.env.ExecuteWorkflow(Workflow, dockerReq)

	<-done
	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.env.AssertActivityNumberOfCalls(s.T(), "RunLoadTest", 0)
	s.env.AssertActivityNumberOfCalls(s.T(), "TeardownProvider", 1)
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowWaitsForLoadTestOnDurationExpiry() {
	loadTestCompleted := false

	cfg, err := types.ParseWorkerConfig("../../conf/worker.yaml")
	if err != nil {
		s.T().Fatal(err)
	}
	testnetActivity := &testnettypes.Activity{
		Chains: cfg.Chains,
	}
	loadTestActivity := &loadtest.Activity{}
	builderActivity := &builder.Activity{}

	s.env.RegisterActivity(testnetActivity.CreateProvider)
	s.env.RegisterActivity(testnetActivity.TeardownProvider)
	s.env.RegisterActivity(testnetActivity.LaunchTestnet)
	s.env.RegisterActivity(loadTestActivity.RunLoadTest)
	s.env.RegisterActivity(builderActivity.BuildDockerImage)

	testnetActivities = testnetActivity
	loadTestActivities = loadTestActivity

	s.env.OnActivity(loadTestActivity.RunLoadTest, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.RunLoadTestRequest) (messages.RunLoadTestResponse, error) {
			time.Sleep(5 * time.Second)
			loadTestCompleted = true
			return messages.RunLoadTestResponse{
				Result: catalysttypes.LoadTestResult{},
			}, nil
		})

	s.env.OnActivity(testnetActivity.TeardownProvider, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.TeardownProviderRequest) (messages.TeardownProviderResponse, error) {
			// fail if teardown is called before loadtest completes
			if !loadTestCompleted {
				s.T().Errorf("TeardownProvider called before LoadTest completed instead of waiting for load test to complete")
			}
			return messages.TeardownProviderResponse{}, nil
		})

	s.env.OnActivity(builderActivity.BuildDockerImage, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.BuildDockerImageRequest) (messages.BuildDockerImageResponse, error) {
			originalTag := "ghcr.io/cosmos/simapp:v0.50"
			newTag := "simapp-v53"

			cmd := exec.Command("docker", "pull", originalTag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return messages.BuildDockerImageResponse{}, err
			}

			tagCmd := exec.Command("docker", "tag", originalTag, newTag)
			tagOutput, err := tagCmd.CombinedOutput()
			if err != nil {
				return messages.BuildDockerImageResponse{}, err
			}

			return messages.BuildDockerImageResponse{
				FQDNTag: newTag,
				Logs:    append(output, tagOutput...),
			}, nil
		})

	dockerReq := simappReq
	dockerReq.Repo = "cosmos-sdk"
	dockerReq.SHA = "acb1d65cdc1e0fc36d93f3c5bb6aaf919a1321e2"
	dockerReq.RunnerType = messages.Docker
	dockerReq.ChainConfig.Name = "stake"
	dockerReq.CatalystVersion = "technicallyty-refactor-wallets"
	dockerReq.TestnetDuration = "3s"
	dockerReq.LongRunningTestnet = false
	dockerReq.BaseMnemonic = "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"
	dockerReq.NumWallets = 1500

	s.env.ExecuteWorkflow(Workflow, dockerReq)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
	s.env.AssertActivityNumberOfCalls(s.T(), "RunLoadTest", 1)
	s.env.AssertActivityNumberOfCalls(s.T(), "TeardownProvider", 1)
}

func TestTestnetWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(TestnetWorkflowTestSuite))
}

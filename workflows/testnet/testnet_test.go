package testnet

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/skip-mev/petri/core/v3/provider/digitalocean"

	"github.com/aws/aws-sdk-go-v2/config"
	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/loadtest"
	testnettypes "github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	testnettype "github.com/skip-mev/ironbird/types/testnet"
	petriutil "github.com/skip-mev/petri/core/v3/util"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
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
	req = messages.TestnetWorkflowRequest{
		InstallationID: 57729708,
		Owner:          "skip-mev",
		ChainConfig: types.ChainsConfig{
			Image: types.ImageConfig{
				UID:        "1000",
				GID:        "1000",
				BinaryName: "/usr/bin/simd",
				HomeDir:    "/simapp",
				Dockerfile: "../../hack/simapp.Dockerfile",
			},
			Version: "v0.50.10",
			GenesisModifications: []petrichain.GenesisKV{
				{
					Key:   "consensus.params.block.max_gas",
					Value: "75000000",
				},
			},
			Dependencies: map[string]string{
				"skip-mev/ironbird-cosmos-sdk": "github.com/cosmos/cosmos-sdk",
				"skip-mev/ironbird-cometbft":   "github.com/cometbft/cometbft",
			},
			NumOfValidators: 1,
			NumOfNodes:      1,
		},
		LoadTestSpec: &catalysttypes.LoadTestSpec{
			Name:        "e2e-test",
			Description: "e2e test",
			NumOfBlocks: 5,
			NumOfTxs:    10,
			Msgs: []catalysttypes.LoadTestMsg{
				{Weight: 1, Type: catalysttypes.MsgSend},
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
	githubActivity := &github.NotifierActivity{}
	s.env.RegisterActivity(githubActivity.CreateGitHubCheck)
	s.env.RegisterActivity(githubActivity.UpdateGitHubCheck)

	testnetActivity := &testnettypes.Activity{}
	s.env.RegisterActivity(testnetActivity.CreateProvider)
	s.env.RegisterActivity(testnetActivity.TeardownProvider)
	s.env.RegisterActivity(testnetActivity.LaunchTestnet)
	s.env.RegisterActivity(testnetActivity.MonitorTestnet)

	loadTestActivity := &loadtest.Activity{}
	s.env.RegisterActivity(loadTestActivity.RunLoadTest)

	builderActivity := &builder.Activity{}
	s.env.RegisterActivity(builderActivity.BuildDockerImage)

	s.env.OnActivity(githubActivity.CreateGitHubCheck, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.CreateGitHubCheckRequest) (messages.CreateGitHubCheckResponse, error) {
			return messages.CreateGitHubCheckResponse(123), nil
		})

	s.env.OnActivity(githubActivity.UpdateGitHubCheck, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.UpdateGitHubCheckRequest) (messages.UpdateGitHubCheckResponse, error) {
			return messages.UpdateGitHubCheckResponse(123), nil
		})

	s.env.OnActivity(builderActivity.BuildDockerImage, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.BuildDockerImageRequest) (messages.BuildDockerImageResponse, error) {
			imageTag := "ghcr.io/cosmos/simapp:v0.50"

			cmd := exec.Command("docker", "pull", imageTag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return messages.BuildDockerImageResponse{}, err
			}

			return messages.BuildDockerImageResponse{
				FQDNTag: imageTag,
				Logs:    output,
			}, nil
		})
}

func (s *TestnetWorkflowTestSuite) setupMockActivitiesDigitalOcean() {
	ctx := context.Background()
	githubActivity := &github.NotifierActivity{}
	s.env.RegisterActivity(githubActivity.CreateGitHubCheck)
	s.env.RegisterActivity(githubActivity.UpdateGitHubCheck)

	doToken := os.Getenv("DIGITALOCEAN_TOKEN")

	nodeAuthKey := os.Getenv("TS_NODE_AUTH_KEY")
	tsServerOauthSecret := os.Getenv("TS_SERVER_OAUTH_SECRET")
	tailscaleSettings, err := digitalocean.SetupTailscale(ctx, tsServerOauthSecret,
		nodeAuthKey, "ironbird-tests", []string{"ironbird-e2e"}, []string{"ironbird-e2e"})
	if err != nil {
		panic(err)
	}

	testnetActivity := &testnettypes.Activity{
		DOToken:           doToken,
		TailscaleSettings: tailscaleSettings,
	}
	s.env.RegisterActivity(testnetActivity.CreateProvider)
	s.env.RegisterActivity(testnetActivity.TeardownProvider)
	s.env.RegisterActivity(testnetActivity.LaunchTestnet)
	s.env.RegisterActivity(testnetActivity.MonitorTestnet)

	loadTestActivity := &loadtest.Activity{
		DOToken:           doToken,
		TailscaleSettings: tailscaleSettings,
	}
	s.env.RegisterActivity(loadTestActivity.RunLoadTest)

	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalln(err)
	}

	builderConfig := types.BuilderConfig{
		BuildKitAddress: "tcp://localhost:1234",
		Registry: types.RegistryConfig{
			URL:       "public.ecr.aws",
			ImageName: "skip-mev/n7v2p5f8/n7v2p5f8/skip-mev/ironbird-local",
		},
	}

	builderActivity := builder.Activity{BuilderConfig: builderConfig, AwsConfig: &awsConfig}
	s.env.RegisterActivity(builderActivity.BuildDockerImage)

	s.env.OnActivity(githubActivity.CreateGitHubCheck, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.CreateGitHubCheckRequest) (messages.CreateGitHubCheckResponse, error) {
			return messages.CreateGitHubCheckResponse(123), nil
		})

	s.env.OnActivity(githubActivity.UpdateGitHubCheck, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req messages.UpdateGitHubCheckRequest) (messages.UpdateGitHubCheckResponse, error) {
			return messages.UpdateGitHubCheckResponse(123), nil
		})
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowDocker() {
	s.setupMockActivitiesDocker()

	// use sdk repo here to test skipping replace workflow
	dockerReq := req
	dockerReq.Repo = "ironbird-cosmos-sdk"
	dockerReq.SHA = "3de8d67d5feb33fad8d3e54236bec1428af3fe6b"
	dockerReq.RunnerType = testnettype.Docker
	dockerReq.ChainConfig.Name = "stake"

	s.env.ExecuteWorkflow(Workflow, dockerReq)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *TestnetWorkflowTestSuite) Test_TestnetWorkflowDigitalOcean() {
	s.setupMockActivitiesDigitalOcean()

	// use cometbft repo here to test replace workflow
	doReq := req
	doReq.Repo = "ironbird-cometbft"
	doReq.SHA = "e5fd4c0cacdb4a338e031083ac6d2b16e404b006"
	doReq.RunnerType = testnettype.DigitalOcean
	doReq.ChainConfig.Name = "stake-" + petriutil.RandomString(3)

	s.env.ExecuteWorkflow(Workflow, doReq)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func TestTestnetWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(TestnetWorkflowTestSuite))
}

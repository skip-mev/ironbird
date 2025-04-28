package testnet

import (
	"errors"
	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/github"
	testnetactivity "github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"testing"
)

var fakeWorkflowOptions = messages.TestnetWorkflowRequest{
	ChainConfig: types.ChainsConfig{
		Name: "testnet",
		Image: types.ImageConfig{
			Dockerfile: "./fake",
			GID:        "1",
			UID:        "1",
			BinaryName: "faked",
			HomeDir:    "/faked",
			GasPrices:  "0.05ufake",
		},
		Dependencies: map[string]string{
			"github.com/fake/fake": "github.com/faker/faker",
		},
		Version:         "v1.1.1",
		NumOfNodes:      1,
		NumOfValidators: 1,
	},
	RunnerType:     testnet.Docker,
	InstallationID: 1,
	Owner:          "fake",
	Repo:           "fake-repo",
	SHA:            "fake-sha",
	LoadTestSpec:   nil,
}

var fakeGithubNotifier = github.NotifierActivity{}
var fakeTailscaleSettings = digitalocean.TailscaleSettings{
	AuthKey:     "",
	Tags:        []string{},
	Server:      nil,
	LocalClient: nil,
}

var fakeTestnetActivity = testnetactivity.Activity{
	TailscaleSettings: fakeTailscaleSettings,
	DOToken:           "",
}

var fakeBuilderActivity = builder.Activity{}

type TestnetTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *TestnetTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *TestnetTestSuite) AfterTest() {
	s.env.AssertExpectations(s.T())
}

func TestTestnetTestSuite(t *testing.T) {
	suite.Run(t, new(TestnetTestSuite))
}

func (s *TestnetTestSuite) TestCreateCheckFailure() {
	s.env.OnActivity(fakeGithubNotifier.CreateGitHubCheck, mock.Anything, mock.Anything).Return(int64(1), errors.New("CreateCheckFailure"))
	s.env.ExecuteWorkflow(Workflow, fakeWorkflowOptions)
	s.Require().True(s.env.IsWorkflowCompleted())

	err := s.env.GetWorkflowError()
	s.Require().Error(err)
	var applicationErr *temporal.ApplicationError
	s.Require().True(errors.As(err, &applicationErr))
	s.Require().Equal("CreateCheckFailure", applicationErr.Error())

	//w.RegisterActivity(testnetActivity.LaunchTestnet)
	//w.RegisterActivity(testnetActivity.MonitorTestnet)
	//w.RegisterActivity(testnetActivity.CreateProvider)
	//w.RegisterActivity(testnetActivity.TeardownProvider)
	//w.RegisterActivity(loadTestActivity.RunLoadTest)
	//
	//w.RegisterActivity(notifier.UpdateGitHubCheck)
	//w.RegisterActivity(notifier.CreateGitHubCheck)
	//
	//w.RegisterActivity(builderActivity.BuildDockerImage)
}

func (s *TestnetTestSuite) TestBuildDockerImageFailure() {
	s.env.OnActivity(fakeGithubNotifier.CreateGitHubCheck, mock.Anything, mock.Anything).Return(int64(1), nil)
	s.env.OnActivity(fakeBuilderActivity.BuildDockerImage, mock.Anything, mock.Anything).Return(builder.BuildResult{}, errors.New("BuildDockerImageFailure"))
	s.env.ExecuteWorkflow(Workflow, fakeWorkflowOptions)
	s.Require().True(s.env.IsWorkflowCompleted())

	err := s.env.GetWorkflowError()
	s.Require().Error(err)
	var applicationErr *temporal.ApplicationError
	s.Require().True(errors.As(err, &applicationErr))
	s.Require().Equal("BuildDockerImageFailure", applicationErr.Error())
}

func (s *TestnetTestSuite) TestCreateProviderFailure() {
	s.env.OnActivity(fakeGithubNotifier.CreateGitHubCheck, mock.Anything, mock.Anything).Return(int64(1), nil)
	s.env.OnActivity(fakeBuilderActivity.BuildDockerImage, mock.Anything, mock.Anything).Return(builder.BuildResult{}, nil)
	s.env.OnActivity(fakeTestnetActivity.CreateProvider, mock.Anything, mock.Anything).Return(nil, errors.New("CreateProviderFailure"))
	s.env.ExecuteWorkflow(Workflow, fakeWorkflowOptions)
	s.Require().True(s.env.IsWorkflowCompleted())

	err := s.env.GetWorkflowError()
	s.Require().Error(err)
	var applicationErr *temporal.ApplicationError
	s.Require().True(errors.As(err, &applicationErr))
	s.Require().Equal("CreateProviderFailure", applicationErr.Error())
}

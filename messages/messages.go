package messages

import (
	"fmt"
	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/types/testnet"
)

type BuildDockerImageRequest struct {
	Tag            string
	Files          map[string][]byte
	BuildArguments map[string]string
}

type BuildDockerImageResponse struct {
	FQDNTag string
	Logs    []byte
}

type CreateGitHubCheckRequest struct {
	InstallationID int64
	Owner          string
	Repo           string
	Name           string
	SHA            string
	Status         *string
	Conclusion     *string
	Title          *string
	Summary        *string
}

type CreateGitHubCheckResponse int64

type UpdateGitHubCheckRequest struct {
	CheckID        int64
	InstallationID int64
	Owner          string
	Repo           string
	Name           string
	Status         *string
	Conclusion     *string
	Title          *string
	Summary        *string
	Text           string
}

type UpdateGitHubCheckResponse int64

type RunLoadTestRequest struct {
	ChainState    []byte
	ProviderState []byte
	LoadTestSpec  catalysttypes.LoadTestSpec
	RunnerType    string
}

type RunLoadTestResponse struct {
	ProviderState []byte
	ChainState    []byte
	Result        catalysttypes.LoadTestResult
}

type LaunchObservabilityStackRequest struct {
	ProviderState          []byte
	ProviderSpecificConfig map[string]string
	PrometheusTargets      []string
	RunnerType             string
}

type LaunchObservabilityStackResponse struct {
	ExternalGrafanaURL string
	GrafanaURL         string
	PrometheusState    []byte
	GrafanaState       []byte
	ProviderState      []byte
}

type CreateProviderRequest struct {
	RunnerType string
	Name       string
}

type CreateProviderResponse struct {
	State []byte
}

type TestnetWorkflowRequest struct {
	InstallationID int64
	Owner          string
	Repo           string
	SHA            string
	ChainConfig    types.ChainsConfig
	RunnerType     testnet.RunnerType
	LoadTestSpec   *catalysttypes.LoadTestSpec
}

func (r TestnetWorkflowRequest) Validate() error {
	if r.InstallationID == 0 {
		return fmt.Errorf("installationID is required")
	}

	if r.Owner == "" {
		return fmt.Errorf("owner is required")
	}

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

	return nil
}

type TestnetWorkflowResponse string

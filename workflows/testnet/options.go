package testnet

import (
	"fmt"
	catalyst_types "github.com/skip-mev/catalyst/pkg/types"

	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/ironbird/util"
)

const TaskQueue = "TESTNET_TASK_QUEUE"

type WorkflowOptions struct {
	InstallationID int64
	Owner          string
	Repo           string
	SHA            string
	ChainConfig    types.ChainsConfig
	RunnerType     testnet.RunnerType
	LoadTestSpec   *catalyst_types.LoadTestSpec
}

func (o *WorkflowOptions) GenerateCheckOptions(name, status, title, summary, text string, conclusion *string) github.CheckRunOptions {
	return github.CheckRunOptions{
		InstallationID: o.InstallationID,
		Owner:          o.Owner,
		Repo:           o.Repo,
		SHA:            o.SHA,
		Name:           name,
		Status:         util.StringPtr(status),
		Title:          util.StringPtr(title),
		Summary:        util.StringPtr(summary),
		Text:           text,
		Conclusion:     conclusion,
	}
}

func (o *WorkflowOptions) Validate() error {
	if o.InstallationID == 0 {
		return fmt.Errorf("installationID is required")
	}

	if o.Owner == "" {
		return fmt.Errorf("owner is required")
	}

	if o.Repo == "" {
		return fmt.Errorf("repo is required")
	}

	if o.SHA == "" {
		return fmt.Errorf("SHA is required")
	}

	if o.ChainConfig.Name == "" {
		return fmt.Errorf("chain name is required")
	}

	if o.ChainConfig.Image.BinaryName == "" {
		return fmt.Errorf("binary name is required")
	}

	if o.ChainConfig.Image.HomeDir == "" {
		return fmt.Errorf("home directory is required")
	}

	if o.RunnerType != testnet.DigitalOcean && o.RunnerType != testnet.Docker {
		return fmt.Errorf("runner type must be one of: %s, %s", testnet.DigitalOcean, testnet.Docker)
	}

	return nil
}

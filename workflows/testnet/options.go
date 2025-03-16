package testnet

import (
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/util"
)

const TaskQueue = "TESTNET_TASK_QUEUE"

type WorkflowOptions struct {
	InstallationID int64
	Owner          string
	Repo           string
	SHA            string
	ChainConfig    types.ChainsConfig
	LoadTestConfig *types.LoadTestConfig
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

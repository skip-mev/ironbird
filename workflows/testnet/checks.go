package testnet

import (
	"github.com/skip-mev/ironbird/activities/github"
	"go.temporal.io/sdk/workflow"
)

func updateCheck(ctx workflow.Context, checkId int64, options github.CheckRunOptions) error {
	return workflow.ExecuteActivity(ctx, githubActivities.UpdateCheck, checkId, options).Get(ctx, nil)
}

func createInitialCheck(ctx workflow.Context, opts WorkflowOptions, name string) (int64, error) {
	var checkID int64

	err := workflow.ExecuteActivity(ctx, githubActivities.CreateCheck, opts.GenerateCheckOptions(
		name,
		"queued",
		"Launching the testnet",
		"Launching the testnet",
		"",
		nil,
	)).Get(ctx, &checkID)

	return checkID, err
}

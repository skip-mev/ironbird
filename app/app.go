package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/workflows/testnet"
	temporalclient "go.temporal.io/sdk/client"
)

type App struct {
	cc             githubapp.ClientCreator
	server         *http.Server
	temporalClient temporalclient.Client
	cfg            types.AppConfig
}

func NewApp(cfg types.AppConfig, temporalClient temporalclient.Client) (*App, error) {
	app := &App{
		temporalClient: temporalClient,
		cfg:            cfg,
	}
	cc, err := githubapp.NewDefaultCachingClientCreator(
		cfg.Github,
	)

	if err != nil {
		return nil, err
	}

	app.cc = cc

	mux := http.NewServeMux()
	webhookHandler := githubapp.NewDefaultEventDispatcher(cfg.Github, app)

	mux.Handle(githubapp.DefaultWebhookRoute, webhookHandler)
	app.server = &http.Server{
		Handler: mux,
		Addr:    ":3000",
	}

	return app, nil
}

func (a *App) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		if err := a.server.Shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	if err := a.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (a *App) handleOpenedPullRequest(ctx context.Context, pr *ValidatedPullRequest) error {
	chain := a.cfg.Chains[0]
	id := fmt.Sprintf("%s/%s/%s/pr-%d", chain.Name, pr.Owner, pr.Repo, pr.Number)

	_, err := a.temporalClient.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		ID:        id,
		TaskQueue: testnet.TaskQueue,
	}, testnet.Workflow, testnet.WorkflowOptions{
		InstallationID: pr.InstallationID,
		Owner:          pr.Owner,
		Repo:           pr.Repo,
		SHA:            pr.HeadSHA,
		ChainConfig:    chain,
	})

	if err != nil {
		fmt.Println("failed to execute workflow", err)
		return err
	}

	//for _, chain := range a.cfg.Chains {
	//	id := fmt.Sprintf("%s/%s/%s/pr-%d", chain.Name, pr.Owner, pr.Repo, pr.Number)
	//
	//	if _, ok := chain.Dependencies[fmt.Sprintf("%s/%s", pr.Owner, pr.Repo)]; !ok {
	//		continue
	//	}

	//_, err := a.temporalClient.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
	//	ID:        id,
	//	TaskQueue: workflows.FullNodeTaskQueue,
	//}, workflows.FullNodeWorkflow, workflows.FullNodeWorkflowOptions{
	//	InstallationID: pr.InstallationID,
	//	Owner:          pr.Owner,
	//	Repo:           pr.Repo,
	//	SHA:            pr.HeadSHA,
	//	ChainConfig:    chain,
	//})

	//	if err != nil {
	//		fmt.Println("failed to execute workflow", err)
	//		continue
	//	}
	//}

	return nil
}

func (a *App) handleClosedPullRequest(ctx context.Context, pr *ValidatedPullRequest) error {
	id := fmt.Sprintf("%s/%s/pr-%d", pr.Owner, pr.Repo, pr.Number)
	workflow := a.temporalClient.GetWorkflow(ctx, id, "")

	if workflow.GetRunID() == "" {
		return fmt.Errorf("no workflow for id %s", id)
	}

	if err := a.temporalClient.CancelWorkflow(ctx, workflow.GetID(), workflow.GetRunID()); err != nil {
		return err
	}

	return nil
}

func (a *App) handlePullRequest(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	validatedPullRequest, err := validatePullRequest(eventType, deliveryID, payload)

	if err != nil {
		return err
	}

	switch validatedPullRequest.Action {
	case "opened":
		return a.handleOpenedPullRequest(ctx, validatedPullRequest)
	}

	return nil
}

func (a *App) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	switch eventType {
	case "pull_request":
		return a.handlePullRequest(ctx, eventType, deliveryID, payload)
	}

	return nil
}

func (a *App) Handles() []string {
	return []string{
		"installation",
		"pull_request",
	}
}

package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/skip-mev/ironbird/types"
	temporalclient "go.temporal.io/sdk/client"
)

type App struct {
	cc             githubapp.ClientCreator
	server         *http.Server
	temporalClient temporalclient.Client
	cfg            types.AppConfig

	commands map[string]Command
}

func NewApp(cfg types.AppConfig) (*App, error) {
	app := &App{
		cfg: cfg,
	}

	temporalClient, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
	})

	if err != nil {
		return nil, err
	}

	app.temporalClient = temporalClient

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

	app.commands = make(map[string]Command)
	app.commands["start"] = Command{
		Description: "",
		Usage:       "/ironbird start <chain> <loadtest>",
		Func:        app.commandStart,
	}
	app.commands["chains"] = Command{
		Func: app.commandChains,
	}
	app.commands["loadtests"] = Command{
		Func: app.commandLoadTests,
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
	return a.SendInitialComment(ctx, pr)
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
	case "reopened":
		return a.handleOpenedPullRequest(ctx, validatedPullRequest)
	case "closed":
		return a.handleClosedPullRequest(ctx, validatedPullRequest)
	}

	return nil
}

func (a *App) handleComment(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	validatedComment, err := validateComment(eventType, deliveryID, payload)

	if err != nil {
		return err
	}

	if validatedComment.Action != "created" {
		return nil
	}

	if strings.HasPrefix(validatedComment.Body, "/ironbird") {
		return a.HandleCommand(ctx, validatedComment, validatedComment.Body)
	}

	return nil
}

func (a *App) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var err error
	switch eventType {
	case "pull_request":
		err = a.handlePullRequest(ctx, eventType, deliveryID, payload)
	case "issue_comment":
		err = a.handleComment(ctx, eventType, deliveryID, payload)
	}

	fmt.Printf("handled %s with err %w\n", eventType, err)

	return err
}

func (a *App) Handles() []string {
	return []string{
		"installation",
		"pull_request",
		"issue_comment",
	}
}

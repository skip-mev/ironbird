package main

import (
	"context"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/skip-mev/ironbird/builder"
	"github.com/skip-mev/ironbird/pipeline"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/petri/core/v2/provider"
	"github.com/skip-mev/petri/core/v2/provider/digitalocean"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"log"
)

func main() {
	cfg, err := types.ParseWorkerConfig("./conf/worker.yaml")

	if err != nil {
		panic(err)
	}

	cc, err := githubapp.NewDefaultCachingClientCreator(cfg.Github)

	if err != nil {
		panic(err)
	}

	notifier := pipeline.GithubNotifierActivity{GithubClient: cc}

	c, err := client.Dial(client.Options{})

	if err != nil {
		log.Fatalln(err)
	}

	defer c.Close()

	builderActivity := builder.Activity{BuilderConfig: cfg.Builder}

	sshKeyPair, err := digitalocean.ParseSSHKeyPair([]byte(cfg.SSHAuth.PrivateKey))

	if err != nil {
		panic(err)
	}

	nodeActivity := pipeline.NodeActivity{
		ProviderCreator: func(ctx context.Context, logger *zap.Logger, name string) (provider.Provider, error) {
			return digitalocean.NewDigitalOceanProvider(ctx, logger, name, cfg.DigitalOcean.Token, []string{}, sshKeyPair)
		},
	}

	w := worker.New(c, pipeline.FullNodeTaskQueue, worker.Options{})

	w.RegisterWorkflow(pipeline.FullNodeWorkflow)

	w.RegisterActivity(nodeActivity.MonitorContainer)
	w.RegisterActivity(nodeActivity.ShutdownNode)
	w.RegisterActivity(nodeActivity.LaunchNode)

	w.RegisterActivity(notifier.UpdateCheck)
	w.RegisterActivity(notifier.CreateCheck)

	w.RegisterActivity(builderActivity.BuildDockerImage)

	err = w.Run(worker.InterruptCh())

	if err != nil {
		log.Fatalln(err)
	}
}

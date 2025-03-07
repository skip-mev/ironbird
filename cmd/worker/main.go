package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"tailscale.com/tsnet"
	"time"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/observability"
	testnetactivity "github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/types"
	testnetworkflow "github.com/skip-mev/ironbird/workflows/testnet"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"log"
)

func main() {
	ctx := context.Background()

	cfg, err := types.ParseWorkerConfig("./conf/worker.yaml")

	if err != nil {
		panic(err)
	}

	cc, err := githubapp.NewDefaultCachingClientCreator(cfg.Github)

	if err != nil {
		panic(err)
	}

	notifier := github.NotifierActivity{GithubClient: cc}

	c, err := client.Dial(client.Options{
		HostPort: "127.0.0.1:7233",
	})

	if err != nil {
		log.Fatalln(err)
	}

	defer c.Close()

	awsConfig, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		log.Fatalln(err)
	}

	builderActivity := builder.Activity{BuilderConfig: cfg.Builder}

	authKey, err := digitalocean.GenerateTailscaleAuthKey(ctx, cfg.Tailscale.ServerOauthSecret, cfg.Tailscale.ServerTags)

	if err != nil {
		panic(err)
	}

	ts := tsnet.Server{
		AuthKey:   authKey,
		Ephemeral: true,
		Hostname:  "ironbird-tests",
	}

	if err := ts.Start(); err != nil {
		panic(err)
	}

	lc, err := ts.LocalClient()

	if err != nil {
		panic(err)
	}

	for {
		status, err := lc.Status(context.Background())
		if err != nil {
			panic(err)
		}

		if status.BackendState == "Running" {
			break
		}

		time.Sleep(1 * time.Second)
	}

	tsLocalClient, err := ts.LocalClient()

	if err != nil {
		panic(err)
	}

	tailscaleSettings := digitalocean.TailscaleSettings{
		AuthKey:     cfg.Tailscale.NodeAuthKey,
		Tags:        cfg.Tailscale.NodeTags,
		Server:      &ts,
		LocalClient: tsLocalClient,
	}

	testnetActivity := testnetactivity.Activity{
		TailscaleSettings: tailscaleSettings,
		DOToken:           cfg.DigitalOcean.Token,
	}

	observabilityActivity := observability.Activity{
		TailscaleSettings:    tailscaleSettings,
		AwsConfig:            &awsConfig,
		ScreenshotBucketName: "ironbird-demo-screenshots",
		DOToken:              cfg.DigitalOcean.Token,
	}
	w := worker.New(c, testnetworkflow.TaskQueue, worker.Options{})

	w.RegisterWorkflow(testnetworkflow.Workflow)

	w.RegisterActivity(testnetActivity.LaunchTestnet)
	w.RegisterActivity(testnetActivity.MonitorTestnet)
	w.RegisterActivity(testnetActivity.CreateProvider)
	w.RegisterActivity(testnetActivity.TeardownProvider)
	w.RegisterActivity(observabilityActivity.LaunchObservabilityStack)
	w.RegisterActivity(observabilityActivity.GrabGraphScreenshot)
	w.RegisterActivity(observabilityActivity.UploadScreenshot)

	w.RegisterActivity(notifier.UpdateCheck)
	w.RegisterActivity(notifier.CreateCheck)

	w.RegisterActivity(builderActivity.BuildDockerImage)

	err = w.Run(worker.InterruptCh())

	if err != nil {
		log.Fatalln(err)
	}
}

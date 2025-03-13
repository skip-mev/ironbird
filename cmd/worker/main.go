package main

import (
	"context"
	"flag"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"tailscale.com/tsnet"

	"log"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/loadtest"
	"github.com/skip-mev/ironbird/activities/observability"
	testnetactivity "github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/types"
	testnetworkflow "github.com/skip-mev/ironbird/workflows/testnet"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

var (
	configFlag = flag.String("config", "./conf/worker.yaml", "Path to the worker configuration file")
)

func main() {
	ctx := context.Background()

	flag.Parse()

	cfg, err := types.ParseWorkerConfig(*configFlag)

	if err != nil {
		panic(err)
	}

	cc, err := githubapp.NewDefaultCachingClientCreator(cfg.Github)

	if err != nil {
		panic(err)
	}

	notifier := github.NotifierActivity{GithubClient: cc}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
	})

	if err != nil {
		log.Fatalln(err)
	}

	defer c.Close()

	awsConfig, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		log.Fatalln(err)
	}

	builderActivity := builder.Activity{BuilderConfig: cfg.Builder, AwsConfig: &awsConfig}

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
		ScreenshotBucketName: cfg.ScreenshotBucketName,
		DOToken:              cfg.DigitalOcean.Token,
	}

	loadTestActivity := loadtest.Activity{
		DOToken:           cfg.DigitalOcean.Token,
		TailscaleSettings: tailscaleSettings,
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
	w.RegisterActivity(loadTestActivity.RunLoadTest)

	w.RegisterActivity(notifier.UpdateCheck)
	w.RegisterActivity(notifier.CreateCheck)

	w.RegisterActivity(builderActivity.BuildDockerImage)

	err = w.Run(worker.InterruptCh())

	if err != nil {
		log.Fatalln(err)
	}
}

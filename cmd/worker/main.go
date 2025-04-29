package main

import (
	"context"
	"flag"
	"github.com/skip-mev/ironbird/util"
	sdktally "go.temporal.io/sdk/contrib/tally"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/uber-go/tally/v4/prometheus"
	"log"

	"github.com/palantir/go-githubapp/githubapp"
	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/loadtest"
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
		MetricsHandler: sdktally.NewMetricsHandler(util.NewPrometheusScope(prometheus.Configuration{
			ListenAddress: "0.0.0.0:9090",
			TimerType:     "histogram",
		})),
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

	tailscaleSettings, err := digitalocean.SetupTailscale(ctx, cfg.Tailscale.ServerOauthSecret,
		cfg.Tailscale.NodeAuthKey, "ironbird", cfg.Tailscale.ServerTags, cfg.Tailscale.NodeTags)
	if err != nil {
		panic(err)
	}

	telemetrySettings := digitalocean.TelemetrySettings{
		Prometheus: digitalocean.PrometheusSettings{
			URL:      cfg.Telemetry.Prometheus.URL,
			Username: cfg.Telemetry.Prometheus.Username,
			Password: cfg.Telemetry.Prometheus.Password,
		},
		Loki: digitalocean.LokiSettings{
			URL:      cfg.Telemetry.Loki.URL,
			Username: cfg.Telemetry.Loki.Username,
			Password: cfg.Telemetry.Loki.Password,
		},
	}

	testnetActivity := testnetactivity.Activity{
		TailscaleSettings: tailscaleSettings,
		TelemetrySettings: telemetrySettings,
		DOToken:           cfg.DigitalOcean.Token,
	}

	loadTestActivity := loadtest.Activity{
		DOToken:           cfg.DigitalOcean.Token,
		TailscaleSettings: tailscaleSettings,
		TelemetrySettings: telemetrySettings,
	}

	w := worker.New(c, testnetworkflow.TaskQueue, worker.Options{})

	w.RegisterWorkflow(testnetworkflow.Workflow)

	w.RegisterActivity(testnetActivity.LaunchTestnet)
	w.RegisterActivity(testnetActivity.MonitorTestnet)
	w.RegisterActivity(testnetActivity.CreateProvider)
	w.RegisterActivity(testnetActivity.TeardownProvider)
	w.RegisterActivity(loadTestActivity.RunLoadTest)

	w.RegisterActivity(notifier.UpdateGitHubCheck)
	w.RegisterActivity(notifier.CreateGitHubCheck)

	w.RegisterActivity(builderActivity.BuildDockerImage)

	err = w.Run(worker.InterruptCh())

	if err != nil {
		log.Fatalln(err)
	}
}

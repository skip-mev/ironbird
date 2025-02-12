package main

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/davecgh/go-spew/spew"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"golang.org/x/oauth2/clientcredentials"
	"strings"
	"tailscale.com/client/tailscale"
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

func getAuthkey(ctx context.Context) (string, error) {
	baseURL := "https://api.tailscale.com"

	credentials := clientcredentials.Config{
		ClientSecret: "tskey-client-ke9ha9iyrm11CNTRL-Ky8rd9KZ5737w5yw36xA23HDqPBtA2hG",
		TokenURL:     baseURL + "/api/v2/oauth/token",
	}

	tsClient := tailscale.NewClient("-", nil)
	tailscale.I_Acknowledge_This_API_Is_Unstable = true
	tsClient.UserAgent = "tailscale-cli"
	tsClient.HTTPClient = credentials.Client(ctx)
	tsClient.BaseURL = baseURL

	caps := tailscale.KeyCapabilities{
		Devices: tailscale.KeyDeviceCapabilities{
			Create: tailscale.KeyDeviceCreateCapabilities{
				Reusable:      false,
				Ephemeral:     true,
				Preauthorized: true,
				Tags:          strings.Split("tag:ironbird", ","),
			},
		},
	}
	authkey, _, err := tsClient.CreateKey(ctx, caps)
	if err != nil {
		return "", err
	}
	return authkey, nil
}

func main() {
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

	awsConfig, err := config.LoadDefaultConfig(context.Background())

	if err != nil {
		log.Fatalln(err)
	}

	builderActivity := builder.Activity{BuilderConfig: cfg.Builder}

	authkey, err := getAuthkey(context.Background())

	if err != nil {
		panic(err)
	}

	ts := tsnet.Server{
		AuthKey:   authkey,
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

		spew.Dump(status)

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
		AuthKey:     "tskey-client-kQLqHQSaEA11CNTRL-Ng7ZgVhjtghcfd8j6r8xmhreMwRpZhWw?ephemeral=true&preauthorized=true",
		Tags:        []string{"ironbird-nodes"},
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

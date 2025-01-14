package main

import (
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/builder"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
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

	notifier := github.NotifierActivity{GithubClient: cc}

	c, err := client.Dial(client.Options{})

	if err != nil {
		log.Fatalln(err)
	}

	defer c.Close()

	builderActivity := builder.Activity{BuilderConfig: cfg.Builder}

	//sshKeyPair, err := digitalocean.ParseSSHKeyPair([]byte(cfg.SSHAuth.PrivateKey))

	//if err != nil {
	//	panic(err)
	//}

	testnetActivity := testnet.Activity{}

	w := worker.New(c, workflows.TestnetTaskQueue, worker.Options{})

	w.RegisterWorkflow(workflows.TestnetWorkflow)

	w.RegisterActivity(testnetActivity.LaunchTestnet)
	w.RegisterActivity(testnetActivity.MonitorTestnet)
	w.RegisterActivity(testnetActivity.CreateProvider)
	w.RegisterActivity(testnetActivity.TeardownProvider)

	w.RegisterActivity(notifier.UpdateCheck)
	w.RegisterActivity(notifier.CreateCheck)

	w.RegisterActivity(builderActivity.BuildDockerImage)

	err = w.Run(worker.InterruptCh())

	if err != nil {
		log.Fatalln(err)
	}
}

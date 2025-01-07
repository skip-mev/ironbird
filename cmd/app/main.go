package main

import (
	"context"
	"github.com/skip-mev/ironbird/app"
	"github.com/skip-mev/ironbird/types"
	"go.temporal.io/sdk/client"
)

func main() {
	ctx := context.Background()
	temporalClient, err := client.Dial(client.Options{})

	if err != nil {
		panic(err)
	}

	defer temporalClient.Close()

	cfg, err := types.ParseAppConfig("./conf/app.yaml")

	if err != nil {
		panic(err)
	}

	app, err := app.NewApp(cfg, temporalClient)

	if err != nil {
		panic(err)
	}

	app.Start(ctx)
}

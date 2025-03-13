package main

import (
	"context"
	"github.com/skip-mev/ironbird/app"
	"github.com/skip-mev/ironbird/types"
)

func main() {
	ctx := context.Background()

	cfg, err := types.ParseAppConfig("./conf/app.yaml")

	if err != nil {
		panic(err)
	}

	app, err := app.NewApp(cfg)

	if err != nil {
		panic(err)
	}

	app.Start(ctx)
}

package main

import (
	"context"
	"flag"

	"github.com/skip-mev/ironbird/app"
	"github.com/skip-mev/ironbird/types"
)

var (
	configFlag = flag.String("config", "./conf/app.yaml", "Path to the app configuration file")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	cfg, err := types.ParseAppConfig(*configFlag)

	if err != nil {
		panic(err)
	}

	app, err := app.NewApp(cfg)

	if err != nil {
		panic(err)
	}

	if err := app.Start(ctx); err != nil {
		panic(err)
	}
}

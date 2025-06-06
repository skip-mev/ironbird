package util

import (
	"context"
	"log"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/core/v3/provider/docker"
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"go.uber.org/zap"
)

func StringPtr(s string) *string {
	return &s
}

func NewPrometheusScope(c prometheus.Configuration) tally.Scope {
	reporter, err := c.NewReporter(
		prometheus.ConfigurationOptions{
			Registry: prom.NewRegistry(),
			OnError: func(err error) {
				log.Println("error in prometheus reporter", err)
			},
		},
	)
	if err != nil {
		log.Fatalln("error creating prometheus reporter", err)
	}
	scopeOpts := tally.ScopeOptions{
		CachedReporter:  reporter,
		Separator:       prometheus.DefaultSeparator,
		SanitizeOptions: &sdktally.PrometheusSanitizeOptions,
		Prefix:          "ironbird",
	}
	scope, _ := tally.NewRootScope(scopeOpts, time.Second)
	scope = sdktally.NewPrometheusNamingScope(scope)

	log.Println("prometheus metrics scope created")
	return scope
}

type ProviderOptions struct {
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
	TelemetrySettings digitalocean.TelemetrySettings
}

func RestoreProvider(ctx context.Context, logger *zap.Logger, runnerType messages.RunnerType, providerState []byte, opts ProviderOptions) (provider.ProviderI, error) {
	if runnerType == messages.Docker {
		return docker.RestoreProvider(ctx, logger, providerState)
	}

	return digitalocean.RestoreProvider(ctx, providerState, opts.DOToken, opts.TailscaleSettings,
		digitalocean.WithLogger(logger), digitalocean.WithTelemetry(opts.TelemetrySettings))
}

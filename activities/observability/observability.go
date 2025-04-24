package observability

import (
	"context"
	"fmt"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/types/testnet"

	"github.com/skip-mev/petri/core/v3/monitoring"
	"github.com/skip-mev/petri/core/v3/provider"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/skip-mev/petri/core/v3/provider/docker"
)

type Options struct {
	ProviderState          []byte
	ProviderSpecificConfig map[string]string
	PrometheusTargets      []string
	RunnerType             string
}

type PackagedState struct {
	ExternalGrafanaURL string
	GrafanaURL         string
	PrometheusState    []byte
	GrafanaState       []byte
	ProviderState      []byte
}

type Activity struct {
	TailscaleSettings digitalocean.TailscaleSettings
	DOToken           string
}

func (a *Activity) LaunchObservabilityStack(ctx context.Context, opts Options) (PackagedState, error) {
	logger, _ := zap.NewDevelopment()

	var p provider.ProviderI
	var err error

	if opts.RunnerType == string(testnet.Docker) {
		p, err = docker.RestoreProvider(
			ctx,
			logger,
			opts.ProviderState,
		)
	} else {
		p, err = digitalocean.RestoreProvider(
			ctx,
			opts.ProviderState,
			a.DOToken,
			a.TailscaleSettings,
			digitalocean.WithLogger(logger),
		)
	}

	if err != nil {
		return PackagedState{}, err
	}

	prometheusOptions := monitoring.PrometheusOptions{
		Targets: opts.PrometheusTargets,
	}

	if opts.RunnerType == string(testnet.DigitalOcean) {
		prometheusOptions.ProviderSpecificConfig = opts.ProviderSpecificConfig
	}

	prometheusTask, err := monitoring.SetupPrometheusTask(ctx, logger, p, prometheusOptions)

	if err != nil {
		return PackagedState{}, err
	}

	if err := prometheusTask.Start(ctx); err != nil {
		return PackagedState{}, err
	}

	prometheusIp, err := prometheusTask.GetIP(ctx)

	if err != nil {
		return PackagedState{}, err
	}

	grafanaOptions := monitoring.GrafanaOptions{
		PrometheusURL: fmt.Sprintf("http://%s:9090", prometheusIp),
		DashboardJSON: monitoring.DefaultDashboardJSON,
	}

	if opts.RunnerType == string(testnet.DigitalOcean) {
		grafanaOptions.ProviderSpecificConfig = opts.ProviderSpecificConfig
	}

	grafanaTask, err := monitoring.SetupGrafanaTask(ctx, logger, p, grafanaOptions)

	if err != nil {
		return PackagedState{}, err
	}

	if err := grafanaTask.Start(ctx); err != nil {
		return PackagedState{}, err
	}

	grafanaIp, err := grafanaTask.GetIP(ctx)

	if err != nil {
		return PackagedState{}, err
	}

	externalGrafanaIp, err := grafanaTask.GetExternalAddress(ctx, "3000")

	if err != nil {
		return PackagedState{}, err
	}

	prometheusState, err := p.SerializeTask(ctx, prometheusTask)

	if err != nil {
		return PackagedState{}, err
	}

	grafanaState, err := p.SerializeTask(ctx, grafanaTask)

	if err != nil {
		return PackagedState{}, err
	}

	providerState, err := p.SerializeProvider(ctx)

	if err != nil {
		return PackagedState{}, err
	}

	return PackagedState{
		GrafanaURL:         fmt.Sprintf("http://%s:3000", grafanaIp),
		ExternalGrafanaURL: fmt.Sprintf("http://%s", externalGrafanaIp),
		PrometheusState:    prometheusState,
		GrafanaState:       grafanaState,
		ProviderState:      providerState,
	}, nil
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
		Prefix:          "temporal_samples",
	}
	scope, _ := tally.NewRootScope(scopeOpts, time.Second)
	scope = sdktally.NewPrometheusNamingScope(scope)

	log.Println("prometheus metrics scope created")
	return scope
}

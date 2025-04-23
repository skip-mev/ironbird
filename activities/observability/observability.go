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
	"github.com/skip-mev/ironbird/messages"

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

type Activity struct {
	TailscaleSettings digitalocean.TailscaleSettings
	DOToken           string
}

func (a *Activity) LaunchObservabilityStack(ctx context.Context, req messages.LaunchObservabilityStackRequest) (messages.LaunchObservabilityStackResponse, error) {
	logger, _ := zap.NewDevelopment()

	var p provider.ProviderI
	var err error

	if req.RunnerType == string(testnet.Docker) {
		p, err = docker.RestoreProvider(
			ctx,
			logger,
			req.ProviderState,
		)
	} else {
		p, err = digitalocean.RestoreProvider(
			ctx,
			req.ProviderState,
			a.DOToken,
			a.TailscaleSettings,
			digitalocean.WithLogger(logger),
		)
	}

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	prometheusOptions := monitoring.PrometheusOptions{
		Targets: req.PrometheusTargets,
	}

	if req.RunnerType == string(testnet.DigitalOcean) {
		prometheusOptions.ProviderSpecificConfig = req.ProviderSpecificConfig
	}

	prometheusTask, err := monitoring.SetupPrometheusTask(ctx, logger, p, prometheusOptions)

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	if err := prometheusTask.Start(ctx); err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	prometheusIp, err := prometheusTask.GetIP(ctx)

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	grafanaOptions := monitoring.GrafanaOptions{
		PrometheusURL: fmt.Sprintf("http://%s:9090", prometheusIp),
		DashboardJSON: monitoring.DefaultDashboardJSON,
	}

	if req.RunnerType == string(testnet.DigitalOcean) {
		grafanaOptions.ProviderSpecificConfig = req.ProviderSpecificConfig
	}

	grafanaTask, err := monitoring.SetupGrafanaTask(ctx, logger, p, grafanaOptions)

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	if err := grafanaTask.Start(ctx); err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	grafanaIp, err := grafanaTask.GetIP(ctx)

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	externalGrafanaIp, err := grafanaTask.GetExternalAddress(ctx, "3000")

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	prometheusState, err := p.SerializeTask(ctx, prometheusTask)

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	grafanaState, err := p.SerializeTask(ctx, grafanaTask)

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	providerState, err := p.SerializeProvider(ctx)

	if err != nil {
		return messages.LaunchObservabilityStackResponse{}, err
	}

	return messages.LaunchObservabilityStackResponse{
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
		Prefix:          "",
	}
	scope, _ := tally.NewRootScope(scopeOpts, time.Second)
	scope = sdktally.NewPrometheusNamingScope(scope)

	log.Println("prometheus metrics scope created")
	return scope
}

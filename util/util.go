package util

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
	"github.com/skip-mev/ironbird/petri/core/provider/docker"
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

func CompressData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		gz.Close()
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

func DecompressData(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	return result, nil
}

package observability

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/types/testnet"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	TailscaleSettings    digitalocean.TailscaleSettings
	AwsConfig            *aws.Config
	ScreenshotBucketName string
	DOToken              string
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

func (a *Activity) GrabGraphScreenshot(ctx context.Context, grafanaUrl, dashboardId, dashboardName, panelId, from string) ([]byte, error) {
	httpClient := http.Client{
		Transport: &http.Transport{
			// Set to true to prevent GZIP-bomb DoS attacks
			DisableCompression: true,
			DialContext:        a.TailscaleSettings.Server.Dial,
			Proxy:              http.ProxyFromEnvironment,
		},
	}

	resp, err := httpClient.Get(
		fmt.Sprintf(
			"%s/render/d-solo/%s/%s?orgId=1&panelId=%s&from=%s&to=now",
			grafanaUrl,
			dashboardId,
			dashboardName,
			panelId,
			from,
		),
	)

	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, nil
	}

	return buf.Bytes(), nil
}

func (a *Activity) UploadScreenshot(ctx context.Context, run, screenshot string, bz []byte) (string, error) {
	if a.AwsConfig == nil {
		return "", fmt.Errorf("aws config is required")
	}

	client := s3.NewFromConfig(*a.AwsConfig)
	key := fmt.Sprintf("%s/%s.png", run, screenshot)

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(a.ScreenshotBucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(bz),
		ContentType: aws.String("image/png"),
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", a.ScreenshotBucketName, key), nil
}

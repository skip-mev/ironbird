package observability

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/skip-mev/ironbird/activities"
	"github.com/skip-mev/petri/core/v3/monitoring"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"go.uber.org/zap"
	"net/http"
)

type Options struct {
	ProviderState          []byte
	ProviderSpecificConfig map[string]string
	PrometheusTargets      []string
}

type PackagedState struct {
	ExternalGrafanaURL string
	GrafanaURL         string
	PrometheusState    []byte
	GrafanaState       []byte
	ProviderState      []byte
}

type Activity struct {
	TailscaleServer      *activities.TailscaleServer
	AwsConfig            *aws.Config
	ScreenshotBucketName string
	DOToken              string
}

func (a *Activity) LaunchObservabilityStack(ctx context.Context, opts Options) (PackagedState, error) {
	logger, _ := zap.NewDevelopment()

	p, err := digitalocean.RestoreProvider(
		ctx,
		opts.ProviderState,
		a.DOToken,
		digitalocean.WithLogger(logger),
		digitalocean.WithTailscale(a.TailscaleServer.Server, a.TailscaleServer.NodeAuthkey, a.TailscaleServer.Tags),
	)

	if err != nil {
		return PackagedState{}, err
	}

	prometheusTask, err := monitoring.SetupPrometheusTask(ctx, logger, p, monitoring.PrometheusOptions{
		Targets:                opts.PrometheusTargets,
		ProviderSpecificConfig: opts.ProviderSpecificConfig,
	})

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

	grafanaTask, err := monitoring.SetupGrafanaTask(ctx, logger, p, monitoring.GrafanaOptions{
		PrometheusURL:          fmt.Sprintf("http://%s:9090", prometheusIp),
		DashboardJSON:          monitoring.DefaultDashboardJSON,
		ProviderSpecificConfig: opts.ProviderSpecificConfig,
	})

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

func (*Activity) GrabGraphScreenshot(ctx context.Context, grafanaUrl, dashboardId, dashboardName, panelId, from string) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("%s/render/d-solo/%s/%s?orgId=1&panelId=%s&from=%s&to=now", grafanaUrl, dashboardId, dashboardName, panelId, from))

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

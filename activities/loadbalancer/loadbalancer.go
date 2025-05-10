package loadbalancer

import (
	"context"
	"fmt"
	"github.com/skip-mev/ironbird/messages"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/petri/core/v3/apps"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"go.uber.org/zap"
)

type Activity struct {
	RootDomain        string
	SSLCertificate    []byte
	SSLKey            []byte
	DOToken           string
	TailscaleSettings digitalocean.TailscaleSettings
	TelemetrySettings digitalocean.TelemetrySettings
}

func (a *Activity) LaunchLoadBalancer(ctx context.Context, req messages.LaunchLoadBalancerRequest) (messages.LaunchLoadBalancerResponse, error) {
	logger, _ := zap.NewDevelopment()

	if req.RunnerType != testnettypes.DigitalOcean {
		return messages.LaunchLoadBalancerResponse{}, fmt.Errorf("only digitalocean provider supported for load balancer")
	}

	p, err := digitalocean.RestoreProvider(
		ctx,
		req.ProviderState,
		a.DOToken,
		a.TailscaleSettings,
		digitalocean.WithLogger(logger),
		digitalocean.WithTelemetry(a.TelemetrySettings),
		digitalocean.WithDomain(a.RootDomain),
	)

	if err != nil {
		return messages.LaunchLoadBalancerResponse{}, fmt.Errorf("failed to restore provider: %w", err)
	}

	lb, err := apps.LaunchLoadBalancer(ctx, p, a.RootDomain, apps.LoadBalancerDefinition{
		SSLKey:         a.SSLKey,
		SSLCertificate: a.SSLCertificate,
		ProviderSpecificOptions: map[string]string{
			"region":   "nyc1",
			"size":     "s-1vcpu-1gb",
			"image_id": "185517855",
		},
		Domains: req.Domains,
	})

	if err != nil {
		return messages.LaunchLoadBalancerResponse{}, fmt.Errorf("failed to launch load balancer: %w", err)
	}

	newProviderState, err := p.SerializeProvider(ctx)
	if err != nil {
		return messages.LaunchLoadBalancerResponse{}, fmt.Errorf("failed to serialize provider: %w", err)
	}

	loadBalancerState, err := p.SerializeTask(ctx, lb)

	if err != nil {
		return messages.LaunchLoadBalancerResponse{
			ProviderState: newProviderState,
		}, fmt.Errorf("failed to serialize load balancer task: %w", err)
	}

	return messages.LaunchLoadBalancerResponse{ProviderState: newProviderState, LoadBalancerState: loadBalancerState}, nil
}

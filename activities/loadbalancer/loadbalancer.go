package loadbalancer

import (
	"context"
	"fmt"
	"github.com/skip-mev/ironbird/db"
	"strings"

	"github.com/skip-mev/ironbird/messages"
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
	DatabaseService   *db.DatabaseService
}

func (a *Activity) LaunchLoadBalancer(ctx context.Context, req messages.LaunchLoadBalancerRequest) (messages.LaunchLoadBalancerResponse, error) {
	logger, _ := zap.NewDevelopment()

	if req.RunnerType != messages.DigitalOcean {
		return messages.LaunchLoadBalancerResponse{}, fmt.Errorf("only digitalocean provider supported for load balancer")
	}

	p, err := digitalocean.RestoreProvider(ctx, req.ProviderState, a.DOToken, a.TailscaleSettings,
		digitalocean.WithLogger(logger), digitalocean.WithDomain(a.RootDomain))

	if err != nil {
		return messages.LaunchLoadBalancerResponse{}, fmt.Errorf("failed to restore provider: %w", err)
	}

	lb, err := apps.LaunchLoadBalancer(ctx, p, a.RootDomain, apps.LoadBalancerDefinition{
		SSLKey:                  a.SSLKey,
		SSLCertificate:          a.SSLCertificate,
		ProviderSpecificOptions: messages.DigitalOceanDefaultOpts,
		Domains:                 req.Domains,
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
		return messages.LaunchLoadBalancerResponse{ProviderState: newProviderState},
			fmt.Errorf("failed to serialize load balancer task: %w", err)
	}

	workflowID := req.WorkflowID

	if a.DatabaseService != nil {
		nodeNames := make(map[string]bool)
		for _, domain := range req.Domains {
			parts := strings.Split(domain.Domain, "-")
			if len(parts) >= 1 {
				nodeNames[parts[0]] = true
			}
		}

		var loadBalancers []messages.Node
		for nodeName := range nodeNames {
			loadBalancers = append(loadBalancers, messages.Node{
				Name:    nodeName,
				Address: a.RootDomain,
				RPC:     fmt.Sprintf("https://%s-rpc.%s", nodeName, a.RootDomain),
				LCD:     fmt.Sprintf("https://%s-lcd.%s", nodeName, a.RootDomain),
			})
		}

		if len(loadBalancers) > 0 {
			logger.Info("updating database with loadbalancers",
				zap.Any("loadBalancers", loadBalancers))

			if err := a.DatabaseService.UpdateWorkflowLoadBalancers(workflowID, loadBalancers); err != nil {
				logger.Error("Failed to update workflow loadbalancers", zap.Error(err))
			}
		} else {
			logger.Warn("No loadbalancers to update in database")
		}
	}

	return messages.LaunchLoadBalancerResponse{ProviderState: newProviderState, LoadBalancerState: loadBalancerState, RootDomain: a.RootDomain}, nil
}

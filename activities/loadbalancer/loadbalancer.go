package loadbalancer

import (
	"context"
	"fmt"
	"regexp"

	pb "github.com/skip-mev/ironbird/server/proto"

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
	GRPCClient        pb.IronbirdServiceClient
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

	if a.GRPCClient != nil {
		nodeNames := make(map[string]bool)
		nodeNameRegex := regexp.MustCompile(`^(.+?)(?:-(?:rpc|lcd|grpc))`)

		for _, domain := range req.Domains {
			matches := nodeNameRegex.FindStringSubmatch(domain.Domain)
			if len(matches) >= 2 {
				nodeNames[matches[1]] = true
			}
		}

		var loadBalancers []*pb.Node
		for nodeName := range nodeNames {
			loadBalancers = append(loadBalancers, &pb.Node{
				Name:    nodeName,
				Address: a.RootDomain,
				Rpc:     fmt.Sprintf("https://%s-rpc.%s", nodeName, a.RootDomain),
				Lcd:     fmt.Sprintf("https://%s-lcd.%s", nodeName, a.RootDomain),
				Grpc:    fmt.Sprintf("%s-grpc.%s", nodeName, a.RootDomain),
			})
		}

		if len(loadBalancers) > 0 {
			updateReq := &pb.UpdateWorkflowDataRequest{
				WorkflowId:    workflowID,
				LoadBalancers: loadBalancers,
			}

			_, err = a.GRPCClient.UpdateWorkflowData(ctx, updateReq)
			if err != nil {
				logger.Error("Failed to update workflow loadbalancers", zap.Error(err))
			}
		}
	}

	return messages.LaunchLoadBalancerResponse{ProviderState: newProviderState, LoadBalancerState: loadBalancerState, RootDomain: a.RootDomain}, nil
}

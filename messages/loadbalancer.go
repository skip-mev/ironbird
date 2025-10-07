package messages

import (
	"github.com/skip-mev/ironbird/petri/core/apps"
)

type LaunchLoadBalancerRequest struct {
	ProviderState []byte
	RunnerType    RunnerType
	Domains       []apps.LoadBalancerDomain
	WorkflowID    string
	IsEvmChain    bool
}

type LaunchLoadBalancerResponse struct {
	ProviderState     []byte
	LoadBalancerState []byte
	RootDomain        string
}

package messages

import (
	"github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/petri/core/v3/apps"
)

type LaunchLoadBalancerRequest struct {
	ProviderState []byte
	RunnerType    testnet.RunnerType
	Domains       []apps.LoadBalancerDomain
	WorkflowID    string
}

type LaunchLoadBalancerResponse struct {
	ProviderState     []byte
	LoadBalancerState []byte
	RootDomain        string
}

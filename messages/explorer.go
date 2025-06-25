package messages

import "github.com/skip-mev/ironbird/types"

type LaunchAutoScoutRequest struct {
	ChainId   string
	ChainName string
	EvmRpcUrl string
	EvmWsUrl  string
	Token     string
}

type LaunchAutoScoutResponse struct {
	InstanceId string
}

type GetAutoScoutInstanceRequest struct {
	InstanceId string
}

type GetAutoScoutInstanceResponse struct {
	InstanceId string
	Config     types.AutoscoutConfig
	URL        string
}

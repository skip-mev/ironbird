package messages

import (
	catalysttypes "github.com/skip-mev/catalyst/chains/types"
)

type RunLoadTestRequest struct {
	ChainState      []byte
	ProviderState   []byte
	LoadTestSpec    catalysttypes.LoadTestSpec
	RunnerType      RunnerType
	IsEvmChain      bool
	Mnemonics       []string
	CatalystVersion string
}

type RunLoadTestResponse struct {
	ProviderState []byte
	ChainState    []byte
	Result        catalysttypes.LoadTestResult
}

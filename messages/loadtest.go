package messages

import (
	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
)

type RunLoadTestRequest struct {
	ChainState    []byte
	ProviderState []byte
	LoadTestSpec  catalysttypes.LoadTestSpec
	RunnerType    RunnerType
	IsEvmChain    bool
	Mnemonics     []string
}

type RunLoadTestResponse struct {
	ProviderState []byte
	ChainState    []byte
	Result        catalysttypes.LoadTestResult
}

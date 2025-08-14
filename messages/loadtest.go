package messages

import (
	"github.com/skip-mev/catalyst/chains/types"
	catalysttypes "github.com/skip-mev/catalyst/chains/types"
)

type RunLoadTestRequest struct {
	ChainState    []byte
	ProviderState []byte
	LoadTestSpec  types.LoadTestSpec
	RunnerType    RunnerType
	IsEvmChain    bool
	Mnemonics     []string
}

type RunLoadTestResponse struct {
	ProviderState []byte
	ChainState    []byte
	Result        catalysttypes.LoadTestResult
}

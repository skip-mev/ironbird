package messages

import (
	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/types/testnet"
)

type RunLoadTestRequest struct {
	ChainState    []byte
	ProviderState []byte
	LoadTestSpec  catalysttypes.LoadTestSpec
	RunnerType    testnet.RunnerType
	GaiaEVM       bool
}

type RunLoadTestResponse struct {
	ProviderState []byte
	ChainState    []byte
	Result        catalysttypes.LoadTestResult
}

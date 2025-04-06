package loadtest

import (
	"context"
	"testing"

	testnetAct "github.com/skip-mev/ironbird/activities/testnet"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	petriutil "github.com/skip-mev/petri/core/v3/util"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/stretchr/testify/require"
)

var testnetOptions = testnetAct.TestnetOptions{
	Name:                    petriutil.RandomString(10),
	Image:                   "ghcr.io/cosmos/simapp",
	UID:                     "1025",
	GID:                     "1025",
	BinaryName:              "simd",
	HomeDir:                 "/simd",
	ProviderSpecificOptions: nil,
	GenesisModifications: []petrichain.GenesisKV{
		{
			Key:   "consensus.params.block.max_gas",
			Value: "75000000",
		},
	},
	RunnerType:      string(testnettypes.Docker),
	NumOfValidators: 1,
	NumOfNodes:      0,
	ProviderState:   nil,
	ChainState:      nil,
}

var loadTestConfig = &LoadTestConfig{
	BlockGasLimitTarget: 0.8,
	NumOfBlocks:         5,
	Msgs: []Message{
		{
			Type:   "bank_send",
			Weight: 1.0,
		},
	},
}

func TestLoadTestLifecycle(t *testing.T) {
	testnetActivity := testnetAct.Activity{}
	loadTestActivity := Activity{}
	options := testnetOptions

	defer func() {
		if options.ProviderState == nil {
			return
		}

		_, err := testnetActivity.TeardownProvider(context.Background(), options)
		require.NoError(t, err)
	}()

	state, err := testnetActivity.CreateProvider(context.Background(), options)
	options.ProviderState = []byte(state)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	packagedState, err := testnetActivity.LaunchTestnet(context.Background(), options)
	options.ProviderState = packagedState.ProviderState
	options.ChainState = packagedState.ChainState
	require.NoError(t, err)
	require.NotNil(t, packagedState)

	loadTestResult, err := loadTestActivity.RunLoadTest(
		context.Background(),
		options.ChainState,
		loadTestConfig,
		options.RunnerType,
		options.ProviderState,
	)

	options.ProviderState = loadTestResult.ProviderState
	options.ChainState = loadTestResult.ChainState

	require.NoError(t, err)
	require.NotNil(t, loadTestResult)
	require.NotEmpty(t, loadTestResult.ProviderState)
	require.NotEmpty(t, loadTestResult.ChainState)
}

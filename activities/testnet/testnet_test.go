package testnet

import (
	"context"
	petriutil "github.com/skip-mev/petri/core/v3/util"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/stretchr/testify/require"
	"testing"
)

var testnetOptions = TestnetOptions{
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
	RunnerType:      "Docker",
	NumOfValidators: 1,
	NumOfNodes:      0,
	ProviderState:   nil,
	ChainState:      nil,
}

func TestProviderLifecycle(t *testing.T) {
	activity := Activity{}
	options := testnetOptions

	state, err := activity.CreateProvider(context.Background(), options)

	require.NoError(t, err)
	require.NotEmpty(t, state)

	options.ProviderState = []byte(state)

	state, err = activity.TeardownProvider(context.Background(), options)
	require.NoError(t, err)
}

func TestChainLifecycle(t *testing.T) {
	activity := Activity{}
	options := testnetOptions

	defer func() {
		if options.ProviderState == nil {
			return
		}

		_, err := activity.TeardownProvider(context.Background(), options)
		require.NoError(t, err)
	}()

	state, err := activity.CreateProvider(context.Background(), options)
	options.ProviderState = []byte(state)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	packagedState, err := activity.LaunchTestnet(context.Background(), options)
	options.ProviderState = packagedState.ProviderState
	options.ChainState = packagedState.ChainState

	require.NoError(t, err)
	require.NotNil(t, packagedState)
}

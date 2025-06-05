package testnet

import (
	"context"
	"github.com/skip-mev/ironbird/messages"
	petriutil "github.com/skip-mev/petri/core/v3/util"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/stretchr/testify/require"
	"testing"
)

var createProviderReq = messages.CreateProviderRequest{
	Name:       petriutil.RandomString(10),
	RunnerType: "Docker",
}

var launchTestnetReq = messages.LaunchTestnetRequest{
	Name:                    petriutil.RandomString(10),
	Image:                   "ghcr.io/cosmos/simapp",
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
}

func TestProviderLifecycle(t *testing.T) {
	activity := Activity{}
	options := createProviderReq

	resp, err := activity.CreateProvider(context.TODO(), options)

	require.NoError(t, err)
	require.NotEmpty(t, resp.ProviderState)

	_, err = activity.TeardownProvider(context.TODO(), messages.TeardownProviderRequest{
		RunnerType:    options.RunnerType,
		ProviderState: resp.ProviderState,
	})
	require.NoError(t, err)
}

func TestChainLifecycle(t *testing.T) {
	activity := Activity{}

	var providerState, chainState []byte
	defer func() {
		if providerState == nil {
			return
		}

		_, err := activity.TeardownProvider(context.TODO(), messages.TeardownProviderRequest{
			RunnerType:    launchTestnetReq.RunnerType,
			ProviderState: providerState,
		})
		require.NoError(t, err)
	}()

	createProviderResp, err := activity.CreateProvider(context.TODO(), createProviderReq)
	require.NoError(t, err)
	providerState = createProviderResp.ProviderState
	require.NotEmpty(t, providerState)

	req := launchTestnetReq
	req.ProviderState = providerState
	packagedState, err := activity.LaunchTestnet(context.TODO(), req)
	providerState = packagedState.ProviderState
	chainState = packagedState.ChainState

	require.NoError(t, err)
	require.NotNil(t, packagedState)
	require.NotNil(t, chainState)
}

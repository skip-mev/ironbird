package observability

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

func TestObservabilityLifecycle(t *testing.T) {
	testnetActivity := testnetAct.Activity{}
	observabilityActivity := Activity{}
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

	var prometheusTargets []string
	for _, node := range packagedState.Nodes {
		prometheusTargets = append(prometheusTargets, node.Metrics)
	}

	observabilityOptions := Options{
		ProviderState:     options.ProviderState,
		PrometheusTargets: prometheusTargets,
		RunnerType:        options.RunnerType,
		ProviderSpecificConfig: map[string]string{
			"region":   "ams3",
			"image_id": "177032231",
			"size":     "s-2vcpu-4gb",
		},
	}

	observabilityResult, err := observabilityActivity.LaunchObservabilityStack(
		context.Background(),
		observabilityOptions,
	)

	options.ProviderState = observabilityResult.ProviderState

	require.NoError(t, err)
	require.NotNil(t, observabilityResult)
	require.NotEmpty(t, observabilityResult.ProviderState)
	require.NotEmpty(t, observabilityResult.GrafanaURL)
	require.NotEmpty(t, observabilityResult.ExternalGrafanaURL)
}

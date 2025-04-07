package observability

import (
	"context"
	"testing"
	"time"

	testnetAct "github.com/skip-mev/ironbird/activities/testnet"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	petriutil "github.com/skip-mev/petri/core/v3/util"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/stretchr/testify/require"
)

var testnetOptions = testnetAct.TestnetOptions{
	Name:                    petriutil.RandomString(10),
	Image:                   "ghcr.io/cosmos/simapp:v0.50",
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

	// Create provider
	state, err := testnetActivity.CreateProvider(context.Background(), options)
	options.ProviderState = []byte(state)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	// Launch testnet
	packagedState, err := testnetActivity.LaunchTestnet(context.Background(), options)
	options.ProviderState = packagedState.ProviderState
	options.ChainState = packagedState.ChainState
	require.NoError(t, err)
	require.NotNil(t, packagedState)

	// Ensure the testnet is running properly
	status, err := testnetActivity.MonitorTestnet(context.Background(), options)
	require.NoError(t, err)
	require.Equal(t, "ok", status)

	// Assert that Nodes array is populated - this is important for proper error handling
	require.NotEmpty(t, packagedState.Nodes, "Testnet nodes should be populated")

	// Extract metrics endpoints from nodes
	var prometheusTargets []string
	for _, node := range packagedState.Nodes {
		prometheusTargets = append(prometheusTargets, node.Metrics)
	}

	// Launch observability stack
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

	// Update provider state from observability stack
	options.ProviderState = observabilityResult.ProviderState

	require.NoError(t, err)
	require.NotNil(t, observabilityResult)
	require.NotEmpty(t, observabilityResult.ProviderState, "Provider state should be preserved")
	require.NotEmpty(t, observabilityResult.GrafanaURL, "Grafana URL should be returned")
	require.NotEmpty(t, observabilityResult.ExternalGrafanaURL, "External Grafana URL should be returned")
	require.NotEmpty(t, observabilityResult.PrometheusState, "Prometheus state should be preserved")
	require.NotEmpty(t, observabilityResult.GrafanaState, "Grafana state should be preserved")
}

// TestObservabilityWithCancelledContext tests that when context is cancelled, proper state is returned
func TestObservabilityWithCancelledContext(t *testing.T) {
	// Skip in short mode as this is a longer test
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

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

	// Create provider
	state, err := testnetActivity.CreateProvider(context.Background(), options)
	options.ProviderState = []byte(state)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	// Launch testnet
	packagedState, err := testnetActivity.LaunchTestnet(context.Background(), options)
	options.ProviderState = packagedState.ProviderState
	options.ChainState = packagedState.ChainState
	require.NoError(t, err)
	require.NotNil(t, packagedState)

	// Extract metrics endpoints from nodes
	var prometheusTargets []string
	for _, node := range packagedState.Nodes {
		prometheusTargets = append(prometheusTargets, node.Metrics)
	}

	// Create options for observability stack with no prometheus targets
	// This will cause an error in prometheus setup
	observabilityOptions := Options{
		ProviderState:     options.ProviderState,
		PrometheusTargets: []string{}, // Empty targets will cause an error
		RunnerType:        options.RunnerType,
		ProviderSpecificConfig: map[string]string{
			"region":   "ams3",
			"image_id": "177032231",
			"size":     "s-2vcpu-4gb",
		},
	}

	// Use a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Launch observability stack with no targets and a context that will time out
	observabilityResult, err := observabilityActivity.LaunchObservabilityStack(
		ctx,
		observabilityOptions,
	)

	// Should return an error
	require.Error(t, err)

	// Even with error, provider state should be returned
	require.NotEmpty(t, observabilityResult.ProviderState, "Provider state should be preserved on error")

	// Update provider state for proper cleanup
	options.ProviderState = observabilityResult.ProviderState
}

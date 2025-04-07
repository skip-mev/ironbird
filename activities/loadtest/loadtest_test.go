package loadtest

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
	Name:       petriutil.RandomString(10),
	Image:      "ghcr.io/cosmos/simapp:v0.50",
	UID:        "1025",
	GID:        "1025",
	BinaryName: "simd",
	HomeDir:    "/simd",
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
	BlockGasLimitTarget: 0.1,
	NumOfBlocks:         3,
	Msgs: []Message{
		{
			Type:   "MsgSend",
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

	time.Sleep(2 * time.Second)
	status, err := testnetActivity.MonitorTestnet(context.Background(), options)
	require.NoError(t, err)
	require.Equal(t, "ok", status)

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
	require.NotEmpty(t, loadTestResult.ProviderState, "Provider state should be preserved")
	require.NotEmpty(t, loadTestResult.ChainState, "Chain state should be preserved")

	require.NotZero(t, loadTestResult.Result.Overall.TotalTransactions, "Load test should report transactions")
}

//func TestLoadTestWithCancelledContext(t *testing.T) {
//	testnetActivity := testnetAct.Activity{}
//	loadTestActivity := Activity{}
//	options := testnetOptions
//
//	defer func() {
//		if options.ProviderState == nil {
//			return
//		}
//
//		_, err := testnetActivity.TeardownProvider(context.Background(), options)
//		require.NoError(t, err)
//	}()
//
//	state, err := testnetActivity.CreateProvider(context.Background(), options)
//	options.ProviderState = []byte(state)
//	require.NoError(t, err)
//	require.NotEmpty(t, state)
//
//	packagedState, err := testnetActivity.LaunchTestnet(context.Background(), options)
//	options.ProviderState = packagedState.ProviderState
//	options.ChainState = packagedState.ChainState
//	require.NoError(t, err)
//	require.NotNil(t, packagedState)
//
//	time.Sleep(2 * time.Second)
//	ctx, cancel := context.WithCancel(context.Background())
//
//	go func() {
//		time.Sleep(2 * time.Second)
//		cancel()
//	}()
//
//	loadTestResult, err := loadTestActivity.RunLoadTest(
//		ctx,
//		options.ChainState,
//		loadTestConfig,
//		options.RunnerType,
//		options.ProviderState,
//	)
//
//	require.Error(t, err)
//	require.Contains(t, err.Error(), "context")
//
//	// Even with error, should still preserve provider & chain state
//	require.NotEmpty(t, loadTestResult.ProviderState, "Provider state should be preserved on context cancellation")
//	require.NotEmpty(t, loadTestResult.ChainState, "Chain state should be preserved on context cancellation")
//
//	options.ProviderState = loadTestResult.ProviderState
//	options.ChainState = loadTestResult.ChainState
//}

// TestLoadTestInvalidInputs tests that proper state is returned when invalid inputs are provided
func TestLoadTestInvalidInputs(t *testing.T) {
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

	// invalid chain state (simulating restore chain error)
	t.Run("Invalid chain state preserves provider state", func(t *testing.T) {
		testCaseOptions := options

		result, err := loadTestActivity.RunLoadTest(
			context.Background(),
			[]byte("invalid-chain-state"),
			loadTestConfig,
			testCaseOptions.RunnerType,
			testCaseOptions.ProviderState,
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to restore chain")

		// Provider state should be preserved
		require.NotEmpty(t, result.ProviderState, "Provider state should be preserved on chain restoration error")

		// The returned provider state should be valid since we didn't corrupt it
		validProviderState := result.ProviderState

		// Verify returned provider state is valid by using it
		_, err = testnetActivity.TeardownProvider(context.Background(), testnetAct.TestnetOptions{
			ProviderState: validProviderState,
			RunnerType:    testCaseOptions.RunnerType,
		})
		require.NoError(t, err, "The preserved provider state should be valid")
	})

	state, err = testnetActivity.CreateProvider(context.Background(), options)
	options.ProviderState = []byte(state)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	packagedState, err := testnetActivity.LaunchTestnet(context.Background(), options)
	options.ProviderState = packagedState.ProviderState
	options.ChainState = packagedState.ChainState
	require.NoError(t, err)
	require.NotNil(t, packagedState)

	res, err := testnetActivity.TeardownProvider(context.Background(), options)
	require.NoError(t, err)
	require.Equal(t, "", res)

	// invalid container image (causing task creation error)
	//t.Run("Invalid container image preserves states", func(t *testing.T) {
	//	tempState, err := testnetActivity.CreateProvider(context.Background(), options)
	//	require.NoError(t, err)
	//	tempOptions := options
	//	tempOptions.ProviderState = []byte(tempState)
	//
	//	tempPackagedState, err := testnetActivity.LaunchTestnet(context.Background(), tempOptions)
	//	require.NoError(t, err)
	//
	//	// Modify the loadTestActivity to use a non-existent image
	//	// this will cause a task creation error
	//	modifiedActivity := Activity{}
	//
	//	result, err := modifiedActivity.RunLoadTest(
	//		context.Background(),
	//		tempPackagedState.ChainState,
	//		loadTestConfig,
	//		"invalid-runner-type", // This will cause the provider restoration to fail
	//		tempPackagedState.ProviderState,
	//	)
	//
	//	require.Error(t, err)
	//
	//	// Even in failure, the provider state should be returned
	//	require.NotEmpty(t, result.ProviderState, "Provider state should be preserved on task creation error")
	//
	//	_, err = testnetActivity.TeardownProvider(context.Background(), testnetAct.TestnetOptions{
	//		ProviderState: result.ProviderState,
	//		RunnerType:    options.RunnerType,
	//	})
	//	require.NoError(t, err)
	//})
}

package testnet

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	petriutil "github.com/skip-mev/petri/core/v3/util"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	resp, err := activity.CreateProvider(context.Background(), options)

	require.NoError(t, err)
	require.NotEmpty(t, resp.ProviderState)

	_, err = activity.TeardownProvider(context.Background(), messages.TeardownProviderRequest{
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

		_, err := activity.TeardownProvider(context.Background(), messages.TeardownProviderRequest{
			RunnerType:    launchTestnetReq.RunnerType,
			ProviderState: providerState,
		})
		require.NoError(t, err)
	}()

	createProviderResp, err := activity.CreateProvider(context.Background(), createProviderReq)
	require.NoError(t, err)
	providerState = createProviderResp.ProviderState
	require.NotEmpty(t, providerState)

	req := launchTestnetReq
	req.ProviderState = providerState
	packagedState, err := activity.LaunchTestnet(context.Background(), req)
	providerState = packagedState.ProviderState
	chainState = packagedState.ChainState

	require.NoError(t, err)
	require.NotNil(t, packagedState)
	require.NotNil(t, chainState)
}

func TestGenerateMonitoringLinks(t *testing.T) {
	dashboardsConfig := &types.DashboardsConfig{
		Grafana: types.GrafanaConfig{
			URL: "https://skipprotocol.grafana.net",
			Dashboards: []types.Dashboard{
				{
					ID:        "b8ff6e6f-5b4b-4d5e-bc50-91bbbf10f436",
					Name:      "comet-performance",
					HumanName: "CometBFT Performance",
				},
			},
		},
	}

	chainID := "test-chain-123"
	startTime := time.Now()

	links := dashboardsConfig.GenerateMonitoringLinks(chainID, startTime)

	require.Len(t, links, 1)
	assert.Contains(t, links, "CometBFT Performance")

	expectedURL := fmt.Sprintf("https://skipprotocol.grafana.net/d/b8ff6e6f-5b4b-4d5e-bc50-91bbbf10f436/comet-performance?orgId=1&var-chain_id=%s&from=%d&to=now&refresh=auto",
		chainID, startTime.UnixMilli())
	assert.Equal(t, expectedURL, links["CometBFT Performance"])
}

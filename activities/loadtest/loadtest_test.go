package loadtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	ethtypes "github.com/skip-mev/catalyst/chains/ethereum/types"
	"github.com/skip-mev/catalyst/chains/types"
	"github.com/skip-mev/ironbird/activities/loadtest/mocks"
	types2 "github.com/skip-mev/ironbird/petri/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gopkg.in/yaml.v3"
)

func TestGenerateSpec(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	chainID := "1234"
	spec := types.LoadTestSpec{
		Name:         "evmtest",
		Description:  "loadtesting stuff",
		Kind:         "eth",
		SendInterval: 500 * time.Millisecond,
		NumBatches:   20,
		Mnemonics:    nil,
		Msgs: []types.LoadTestMsg{
			{
				Type:    "MsgNativeTransferERC20",
				NumMsgs: 1500,
			},
		},
		ChainCfg: &ethtypes.ChainConfig{
			TxOpts: ethtypes.TxOpts{
				GasFeeCap: big.NewInt(100),
				GasTipCap: big.NewInt(109009),
			},
		},
	}

	mnemonics := []string{
		"the coin is for everyone",
		"blockchain liberate us all",
		"all your base are belong to us",
		"hello world i am here to save you",
	}
	chain := mocks.NewMocktheChain(gomock.NewController(t))
	chain.EXPECT().GetConfig().Times(1).Return(types2.ChainConfig{})

	nodes := []types2.NodeI{
		mocks.MockNode{IP: "127.0.0.1"},
		mocks.MockNode{IP: "127.0.0.2"},
		mocks.MockNode{IP: "127.0.0.3"},
		mocks.MockNode{IP: "127.0.0.4"},
	}
	chain.EXPECT().GetNodes().Times(1).Return(nodes)

	specBZ, err := generateLoadTestSpec(ctx, logger, chain, chainID, spec, mnemonics)
	require.NoError(t, err)

	var gotLoadtestSpec types.LoadTestSpec
	err = yaml.Unmarshal(specBZ, &gotLoadtestSpec)
	require.NoError(t, err)

	// the function will set mnemonics.
	spec.Mnemonics = mnemonics
	spec.ChainID = chainID
	spec.ChainCfg.(*ethtypes.ChainConfig).NodesAddresses = gotLoadtestSpec.ChainCfg.(*ethtypes.ChainConfig).NodesAddresses
	require.Equal(t, gotLoadtestSpec, spec)
	require.Equal(t, len(nodes), len(gotLoadtestSpec.ChainCfg.(*ethtypes.ChainConfig).NodesAddresses))
	require.True(t, len(gotLoadtestSpec.Mnemonics) > 0)
}

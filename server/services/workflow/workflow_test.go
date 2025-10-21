package workflow

import (
	"math/big"
	"testing"

	ctlteth "github.com/skip-mev/catalyst/chains/ethereum/types"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeLoadTestSpec(t *testing.T) {
	encodedLoadTestSpec := "{\"name\":\"eth_loadtest\",\"description\":\"testing\",\"kind\":\"eth\",\"chain_id\":\"262144\",\"send_interval\":\"1s\",\"num_batches\":360,\"msgs\":[{\"type\":\"MsgNativeTransferERC20\",\"num_msgs\":800}],\"chain_config\":{\"tx_opts\":{\"gas_fee_cap\":10000000,\"gas_tip_cap\":10000000}}}"
	decodedLoadTestSpec, err := decodeLoadTestSpec(encodedLoadTestSpec)
	require.NoError(t, err)
	cfg, ok := decodedLoadTestSpec.ChainCfg.(*ctlteth.ChainConfig)
	require.True(t, ok)
	require.Equal(t, big.NewInt(10000000), cfg.TxOpts.GasFeeCap)
	require.Equal(t, big.NewInt(10000000), cfg.TxOpts.GasTipCap)
	_, err = encodeLoadTestSpec(decodedLoadTestSpec)
	require.NoError(t, err)
}

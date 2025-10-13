package mocks

import (
	"context"
	"net"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/rpc/client/http"
	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/types"
	"google.golang.org/grpc"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.NodeI = MockNode{}

type MockNode struct {
	IP string
}

func (m MockNode) Start(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) Stop(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) Destroy(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) GetDefinition() provider.TaskDefinition {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) GetStatus(ctx context.Context) (provider.TaskStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) Modify(ctx context.Context, definition provider.TaskDefinition) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) WriteFile(ctx context.Context, s string, bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) ReadFile(ctx context.Context, s string) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) DownloadDir(ctx context.Context, s string, s2 string) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) GetIP(ctx context.Context) (string, error) {
	return m.IP, nil
}

func (m MockNode) GetPrivateIP(ctx context.Context) (string, error) {
	return m.GetIP(ctx)
}

func (m MockNode) GetExternalAddress(ctx context.Context, s string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) DialContext() func(context.Context, string, string) (net.Conn, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) RunCommand(ctx context.Context, strings []string) (string, string, int, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) GetConfig() types.NodeConfig {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) GetTMClient(ctx context.Context) (*http.HTTP, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) GetGRPCClient(ctx context.Context) (*grpc.ClientConn, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) Height(ctx context.Context) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) CopyGenTx(ctx context.Context, i types.NodeI) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) GenesisFileContent(ctx context.Context) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) OverwriteGenesisFile(ctx context.Context, bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) CreateWallet(ctx context.Context, s string, config types.WalletConfig) (types.WalletI, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) RecoverKey(ctx context.Context, s string, s2 string) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) KeyBech32(ctx context.Context, s string, s2 string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) SetChainConfigs(ctx context.Context, s string, s2 string) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) SetPersistentPeers(ctx context.Context, s string) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) SetSeedNode(ctx context.Context, seedNode string) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) SetSeedMode(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) SetupNode(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) SetupValidator(context.Context, types.WalletConfig, []sdk.Coin, sdk.Coin) (types.WalletI, string, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) NodeId(ctx context.Context) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) PubKey(ctx context.Context) (crypto.PubKey, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockNode) Serialize(ctx context.Context, i provider.ProviderI) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

package fullnode

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/skip-mev/petri/core/v2/provider"
	petritypes "github.com/skip-mev/petri/core/v2/types"
	"github.com/skip-mev/petri/cosmos/v2/chain"
	"github.com/skip-mev/petri/cosmos/v2/node"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

type NodeActivity struct {
	ProviderCreator func(ctx context.Context, logger *zap.Logger, name string) (provider.ProviderI, error)
}

type NodeOptions struct {
	Name                    string
	SnapshotURL             string
	Chain                   string
	DockerfilePath          string
	Image                   string
	UID                     string
	GID                     string
	BinaryName              string
	HomeDir                 string
	GasPrices               string
	ProviderSpecificOptions map[string]string
}

func (n *NodeActivity) configureNode(ctx context.Context, opts NodeOptions, nn petritypes.NodeI) error {
	if err := nn.InitHome(ctx); err != nil {
		return err
	}

	snapshotCommand := fmt.Sprintf("wget -O - %s | tar -x -C %s", opts.SnapshotURL, opts.HomeDir)

	if _, _, exitCode, err := nn.RunCommand(ctx, []string{"sh", "-c", snapshotCommand}); err != nil || exitCode != 0 {
		return fmt.Errorf("failed to download and extract snapshot (exit code=%d): %w", exitCode, err)
	}

	genesisFile, err := getGenesisFile(opts.Chain)

	if err != nil {
		return err
	}

	err = nn.OverwriteGenesisFile(ctx, genesisFile)

	if err != nil {
		return err
	}

	if err := nn.SetDefaultConfigs(ctx); err != nil {
		return err
	}

	peers, err := getPersistentPeers(opts.Chain)

	if err != nil {
		return err
	}

	if err := nn.SetPersistentPeers(ctx, peers); err != nil {
		return err
	}

	return nil
}

func (n *NodeActivity) ShutdownNode(ctx context.Context, name string) error {
	p, err := n.ProviderCreator(ctx, zap.NewNop(), name)

	if err != nil {
		return err
	}

	return p.Teardown(ctx)
}

func (n *NodeActivity) LaunchNode(ctx context.Context, opts NodeOptions) (string, error) {
	p, err := n.ProviderCreator(ctx, zap.NewNop(), opts.Name)

	if err != nil {
		return "", err
	}

	c := &chain.Chain{
		State: chain.State{
			Config: petritypes.ChainConfig{
				Image: provider.ImageDefinition{
					Image: opts.Image,
					UID:   opts.UID,
					GID:   opts.GID,
				},
				BinaryName: opts.BinaryName, HomeDir: opts.HomeDir,
				GasPrices: opts.GasPrices,
			},
		},
	}

	nn, err := node.CreateNode(ctx, zap.NewNop(), p, petritypes.NodeConfig{
		Name:        fmt.Sprintf("%s-node", opts.Name),
		Index:       0,
		IsValidator: false,
		ChainConfig: c.State.Config,
	}, petritypes.NodeOptions{})

	if err != nil {
		return "", err
	}

	nn = nn.(*node.Node)

	if err := n.configureNode(ctx, opts, nn); err != nil {
		return "", err
	}

	if err := nn.Start(ctx); err != nil {
		return "", err
	}

	return "", nil
}

func getGenesisFile(chain string) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf("https://snapshots.polkachu.com/genesis/%s/genesis.json"))

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	return io.ReadAll(resp.Body)
}

type PolkachuPeers struct {
	PolkachuPeer string   `json:"polkachu_peer"`
	LivePeers    []string `json:"live_peers"`
}

func getPersistentPeers(chain string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("https://polkachu.com/api/v2/chains/%s/live_peers", chain))

	if err != nil {
		return "", err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	var polkachuPeers PolkachuPeers

	if err := json.NewDecoder(resp.Body).Decode(&polkachuPeers); err != nil {
		return "", err
	}

	polkachuPeers.LivePeers = append(polkachuPeers.LivePeers, polkachuPeers.PolkachuPeer)

	return strings.Join(polkachuPeers.LivePeers, ","), nil
}

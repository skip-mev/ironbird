package chain

import (
	"context"
	"fmt"
	"strings"

	cmcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	petri "github.com/skip-mev/ironbird/petri/core/types"
)

type PeerSet struct {
	peers []petri.NodeI
}

const cometPort = "26656"

func NewPeerSet(peers []petri.NodeI) PeerSet {
	return PeerSet{peers: peers}
}

func (ps *PeerSet) Empty() bool {
	return len(ps.peers) == 0
}

func (ps *PeerSet) Len() int {
	return len(ps.peers)
}

// AsCometPeerString returns a comma-delimited string with the addresses of chain nodes in
// the format of nodeid@host:port.
func (ps *PeerSet) AsCometPeerString(ctx context.Context, useExternal bool) (string, error) {
	resolveHost := peerHostInternal
	if useExternal {
		resolveHost = peerHostExternal
	}

	peerStrings := make([]string, 0, len(ps.peers))

	for _, n := range ps.peers {
		nodeID, err := n.NodeId(ctx)
		if err != nil {
			return "", errors.Wrap(err, "node id")
		}

		host, err := resolveHost(ctx, n)
		if err != nil {
			return "", errors.Wrap(err, "host")
		}

		peerStrings = append(peerStrings, fmt.Sprintf("%s@%s", nodeID, host))
	}

	return strings.Join(peerStrings, ","), nil
}

// AsLibP2PAddressBook returns a map of node IDs to addresses of chain nodes.
// Format it [{host: "1.2.3.4:26656", id: "<lib-p2p-peer-id>"}, {...}, ...]
// @see https://github.com/cometbft/cometbft/blob/608fe92cbc3774c6cdf36c59c56b6c8362489ef1/lp2p/addressbook.go#L16
func (ps *PeerSet) AsLibP2PAddressBook(ctx context.Context, isDocker bool) ([]any, error) {
	peers := make([]any, 0, len(ps.peers))

	resolveHost := peerHostInternal
	if !isDocker {
		// This breaks geo-distributed testnet for go-libp2p.
		//
		// Currently DigitalOcean Droplet are configured via Tailscale, which
		// causes issues go-libp2p connection (tailscale's IP are fetched via peerHostExternal)
		// Thus, we use VMs private IPs.
		//
		// TODO: STACK-1615: come up with a better solution that doesn't use Tailscale, but supports multiple regions.
		resolveHost = peerHostPrivate
	}

	for _, n := range ps.peers {
		host, err := resolveHost(ctx, n)
		if err != nil {
			return nil, errors.Wrap(err, "host")
		}

		peerID, err := peerIDFromNode(ctx, n)
		if err != nil {
			return nil, errors.Wrap(err, "peer id")
		}

		peers = append(peers, map[string]string{
			"host": host,
			"id":   peerID.String(),
		})
	}

	return peers, nil
}

// used for digitalocean
func peerHostExternal(ctx context.Context, n petri.NodeI) (string, error) {
	return n.GetExternalAddress(ctx, cometPort)
}

// used for docker
func peerHostInternal(ctx context.Context, n petri.NodeI) (string, error) {
	ip, err := n.GetIP(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", ip, cometPort), nil
}

func peerHostPrivate(ctx context.Context, n petri.NodeI) (string, error) {
	ip, err := n.GetPrivateIP(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", ip, cometPort), nil
}

// used for digitalocean
func peerIDFromNode(ctx context.Context, n petri.NodeI) (peer.ID, error) {
	cometPubKey, err := n.PubKey(ctx)
	if err != nil {
		return "", errors.Wrap(err, "comet public key")
	}

	pubKey, err := pubKeyFromCosmosKey(cometPubKey)
	if err != nil {
		return "", errors.Wrap(err, "libp2p public key")
	}

	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return "", errors.Wrap(err, "peer id")
	}

	return peerID, nil
}

func pubKeyFromCosmosKey(key cmcrypto.PubKey) (crypto.PubKey, error) {
	var (
		keyType = key.Type()
		raw     = key.Bytes()
	)

	switch keyType {
	case ed25519.KeyType:
		return crypto.UnmarshalEd25519PublicKey(raw)
	case secp256k1.KeyType:
		return crypto.UnmarshalSecp256k1PublicKey(raw)
	default:
		return nil, fmt.Errorf("unsupported public key type %q", keyType)
	}
}

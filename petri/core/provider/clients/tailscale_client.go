package clients

import (
	"context"
	"net"
	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

var _ TailscaleServer = (*tsnet.Server)(nil)
var _ TailscaleLocalClient = (*tailscale.LocalClient)(nil)

type TailscaleServer interface {
	Dial(ctx context.Context, network, address string) (net.Conn, error)
}

type TailscaleLocalClient interface {
	Status(ctx context.Context) (*ipnstate.Status, error)
}

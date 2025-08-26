package digitalocean

import (
	"context"
	"golang.org/x/crypto/ssh"
	"net"
)

func SSHDialWithCustomDial(ctx context.Context, network, addr string, config *ssh.ClientConfig, dialFunc func(ctx context.Context, network, address string) (net.Conn, error)) (*ssh.Client, error) {
	conn, err := dialFunc(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(c, chans, reqs), nil
}

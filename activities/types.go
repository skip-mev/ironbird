package activities

import "tailscale.com/tsnet"

type TailscaleServer struct {
	Server      *tsnet.Server
	NodeAuthkey string
	Tags        []string
}

package main

import (
	"context"
	"golang.org/x/oauth2/clientcredentials"
	"strings"
	"tailscale.com/client/tailscale"
)

const s = "tskey-client-kPdRfVq31321CNTRL-PEzJKVPx9q1jEgS3rkJ4k1b8nUjMq862"

func getAuthkey(ctx context.Context) (string, error) {
	baseURL := "https://api.tailscale.com"

	credentials := clientcredentials.Config{
		ClientSecret: s,
		TokenURL:     baseURL + "/api/v2/oauth/token",
	}

	tsClient := tailscale.NewClient("-", nil)
	tailscale.I_Acknowledge_This_API_Is_Unstable = true
	tsClient.UserAgent = "tailscale-cli"
	tsClient.HTTPClient = credentials.Client(ctx)
	tsClient.BaseURL = baseURL

	caps := tailscale.KeyCapabilities{
		Devices: tailscale.KeyDeviceCapabilities{
			Create: tailscale.KeyDeviceCreateCapabilities{
				Reusable:      false,
				Ephemeral:     true,
				Preauthorized: true,
				Tags:          strings.Split("tag:ironbird", ","),
			},
		},
	}
	authkey, _, err := tsClient.CreateKey(ctx, caps)
	if err != nil {
		return "", err
	}
	return authkey, nil
}

func main() {
	_, s := getAuthkey(context.Background())
	panic(s)
}

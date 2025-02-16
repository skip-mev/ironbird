package builder

import (
	"context"
	"fmt"
	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/staticfs"
	"github.com/skip-mev/ironbird/types"
	"github.com/tonistiigi/fsutil"
	fstypes "github.com/tonistiigi/fsutil/types"
)

type Activity struct {
	BuilderConfig types.BuilderConfig
}

func (a *Activity) BuildDockerImage(ctx context.Context, tag string, files map[string][]byte, buildArgs map[string]string) (string, error) {
	bkClient, err := client.New(ctx, a.BuilderConfig.BuildKitAddress)

	if err != nil {
		return "", err
	}
	defer bkClient.Close()

	fs := staticfs.NewFS()
	for name, content := range files {
		fs.Add(name, &fstypes.Stat{Mode: 0644}, content)
	}

	authProvider := authprovider.NewDockerAuthProvider(&configfile.ConfigFile{
		AuthConfigs: map[string]configtypes.AuthConfig{
			a.BuilderConfig.Registry.URL: {
				Username: a.BuilderConfig.Registry.Username,
				Password: a.BuilderConfig.Registry.Password,
			},
		},
	}, map[string]*authprovider.AuthTLSConfig{})

	frontendAttrs := map[string]string{
		"filename": "Dockerfile",
		"target":   "",
	}

	for k, v := range buildArgs {
		frontendAttrs[fmt.Sprintf("build-arg:%s", k)] = v
	}

	fqdnTag := fmt.Sprintf("%s/%s", a.BuilderConfig.Registry.FQDN, tag)

	solveOpt := client.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,
		LocalMounts: map[string]fsutil.FS{
			"context":    fs,
			"dockerfile": fs,
		},
		Session: []session.Attachable{
			authProvider,
		},
		Exports: []client.ExportEntry{
			{
				Type: client.ExporterImage,
				Attrs: map[string]string{
					"name": fqdnTag,
					"push": "true",
				},
			},
		},
	}

	statusChan := make(chan *client.SolveStatus)

	go func() {
		for status := range statusChan {
			for _, v := range status.Logs {
				fmt.Printf("[%s]: %s\n", v.Timestamp.String(), string(v.Data))
			}
		}
	}()

	_, err = bkClient.Solve(ctx, nil, solveOpt, statusChan)

	if err != nil {
		return "", err
	}

	return fqdnTag, nil
}

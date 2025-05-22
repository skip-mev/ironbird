package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/skip-mev/ironbird/types"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
	"github.com/skip-mev/ironbird/server"
	"google.golang.org/grpc/grpclog"
)

type CaddyModule struct {
	server *server.IronbirdServer
	config types.TemporalConfig
}

func (CaddyModule) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "http.handlers.ironbird",
		New: func() caddy.Module {
			m := &CaddyModule{
				config: types.TemporalConfig{
					Host:      "127.0.0.1:7233", // Use IPv4 address instead of localhost
					Namespace: "default",        // Default Temporal namespace
				},
			}
			return m
		},
	}
}

func (m *CaddyModule) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

func (m *CaddyModule) Provision(ctx caddy.Context) error {
	var err error
	m.server, err = server.NewIronbirdServer(m.config)
	return err
}

// Cleanup implements caddy.CleanerUpper and closes the IronbirdServer
func (m *CaddyModule) Cleanup() error {
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}

func (m *CaddyModule) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	switch {
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ironbird/workflow"):
		m.server.HandleCreateWorkflow(w, r)
	//case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/ironbird/workflow/"):
	//	m.server.HandleUpdateWorkflow(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/ironbird/workflow/"):
		m.server.HandleGetWorkflow(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ironbird/loadtest/"):
		m.server.HandleRunLoadTest(w, r)
	default:
		return next.ServeHTTP(w, r)
	}
	return nil
}

var (
	_ caddy.Module                = (*CaddyModule)(nil)
	_ caddy.Provisioner           = (*CaddyModule)(nil)
	_ caddy.CleanerUpper          = (*CaddyModule)(nil)
	_ caddyhttp.MiddlewareHandler = (*CaddyModule)(nil)
	_ caddyfile.Unmarshaler       = (*CaddyModule)(nil)
)

func init() {
	caddy.RegisterModule(CaddyModule{})
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr))
}

// startAPIServer starts a standard HTTP server that uses the existing IronbirdServer
func startAPIServer() {
	// Create a default temporal config
	temporalConfig := types.TemporalConfig{
		Host:      "127.0.0.1:7233", // Use IPv4 address instead of localhost
		Namespace: "default",        // Default Temporal namespace
	}

	// Create shared server instance to be reused by requests
	ironbirdServer, err := server.NewIronbirdServer(temporalConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Ironbird server: %v\n", err)
		os.Exit(1)
	}

	// Create a handler that uses the shared server instance
	ironbirdHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ironbird/workflow"):
			ironbirdServer.HandleCreateWorkflow(w, r)
		//case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/ironbird/workflow/"):
		//	ironbirdServer.HandleUpdateWorkflow(w, r)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/ironbird/workflow/"):
			ironbirdServer.HandleGetWorkflow(w, r)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/ironbird/loadtest/"):
			ironbirdServer.HandleRunLoadTest(w, r)
		default:
			http.NotFound(w, r)
			return
		}
	})

	// Start the server in a goroutine
	go func() {
		fmt.Println("Starting API server on :8090")
		if err := http.ListenAndServe(":8090", ironbirdHandler); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting API server: %v\n", err)
			os.Exit(1)
		}
	}()
}

func main() {
	caddyfileFlag := flag.String("caddyfile", "./server/Caddyfile", "Path to Caddyfile")
	flag.Parse()

	// Start the API server that will be proxied by Caddy
	startAPIServer()

	caddyfileBytes, err := os.ReadFile(*caddyfileFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading Caddyfile: %v\n", err)
		os.Exit(1)
	}

	cfgAdapter := caddyconfig.GetAdapter("caddyfile")
	config, warn, err := cfgAdapter.Adapt(caddyfileBytes, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adapting Caddyfile: %v\n", err)
		os.Exit(1)
	}
	if warn != nil {
		fmt.Fprintf(os.Stderr, "Warnings: %v\n", warn)
	}

	err = caddy.Load(config, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	select {}
}

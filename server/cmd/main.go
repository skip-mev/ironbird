package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/skip-mev/ironbird/db"
	"github.com/skip-mev/ironbird/server"
	"github.com/skip-mev/ironbird/types"
	"go.uber.org/zap"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
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
					Host:      "127.0.0.1:7233", // TODO(nadim-az): update when deploying to prod to point to temporalconfig
					Namespace: "default",
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

	dbPath := getEnvOrDefault("DATABASE_PATH", "./ironbird.db")

	database, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	logger, _ := zap.NewDevelopment()

	m.server, err = server.NewIronbirdServer(m.config, database, logger)
	return err
}

func (m *CaddyModule) Cleanup() error {
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}

// handleIronbirdRoutes routes HTTP requests to the appropriate handler based on method and path.
// Returns true if the request was handled, false otherwise.
func handleIronbirdRoutes(w http.ResponseWriter, r *http.Request, server *server.IronbirdServer) bool {
	const (
		basePath      = "/ironbird"
		workflowPath  = basePath + "/workflow"
		workflowsPath = basePath + "/workflows"
		loadTestPath  = basePath + "/loadtest/"
	)

	handleRequest := func(handlerFunc func(http.ResponseWriter, *http.Request) error) bool {
		if err := handlerFunc(w, r); err != nil {
			errMsg := fmt.Sprintf("Error: %v", err)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return false
		}
		return true
	}

	switch {
	// POST /ironbird/workflow/{id}/cancel - Cancel a workflow
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, workflowPath) && strings.HasSuffix(r.URL.Path, "/cancel"):
		return handleRequest(server.HandleCancelWorkflow)

	// POST /ironbird/workflow/{id}/signal/{signal} - Signal a workflow
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, workflowPath) && strings.Contains(r.URL.Path, "/signal/"):
		return handleRequest(server.HandleSignalWorkflow)

	// POST /ironbird/workflow - Create a new workflow
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, workflowPath):
		return handleRequest(server.HandleCreateWorkflow)

	// GET /ironbird/workflows - List all workflows
	case r.Method == http.MethodGet && r.URL.Path == workflowsPath:
		return handleRequest(server.HandleListWorkflows)

	// GET /ironbird/workflow/{id} - Get a specific workflow
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, workflowPath+"/"):
		return handleRequest(server.HandleGetWorkflow)

	// POST /ironbird/loadtest/{id} - Run a load test
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, loadTestPath):
		return handleRequest(server.HandleRunLoadTest)

	// No matching route found
	default:
		return false
	}
}

func (m *CaddyModule) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if strings.HasPrefix(r.URL.Path, "/ironbird/") {
		if !handleIronbirdRoutes(w, r, m.server) {
			http.NotFound(w, r)
		}
		return nil
	}
	return next.ServeHTTP(w, r)
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

func main() {
	logger, _ := zap.NewDevelopment()

	caddyfileFlag := flag.String("caddyfile", "./server/Caddyfile", "Path to Caddyfile")
	flag.Parse()

	caddyfileBytes, err := os.ReadFile(*caddyfileFlag)
	if err != nil {
		logger.Error("reading Caddyfile", zap.Error(err))
		os.Exit(1)
	}

	cfgAdapter := caddyconfig.GetAdapter("caddyfile")
	config, warn, err := cfgAdapter.Adapt(caddyfileBytes, nil)
	if err != nil {
		logger.Error("adapting Caddyfile", zap.Error(err))
		os.Exit(1)
	}
	if warn != nil {
		logger.Warn("warnings during Caddyfile adaptation", zap.Any("warnings", warn))
	}

	err = caddy.Load(config, true)
	if err != nil {
		logger.Error("loading configuration", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("server started successfully - only reading from database")

	select {}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

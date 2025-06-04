package server

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/util"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/skip-mev/ironbird/server/db"
	pb "github.com/skip-mev/ironbird/server/proto"
	"github.com/skip-mev/ironbird/server/services/workflow"
	"github.com/uber-go/tally/v4/prometheus"
	temporalclient "go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRpcServer struct {
	temporalClient  temporalclient.Client
	config          types.TemporalConfig
	db              db.DB
	grpcServer      *grpc.Server
	logger          *zap.Logger
	stopCh          chan struct{}
	workflowService *workflow.Service
}

func NewGRpcServer(config types.TemporalConfig, database db.DB, logger *zap.Logger) (*GRpcServer, error) {
	temporalClient, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  config.Host,
		Namespace: config.Namespace,
		MetricsHandler: sdktally.NewMetricsHandler(util.NewPrometheusScope(prometheus.Configuration{
			ListenAddress: "0.0.0.0:9090",
			TimerType:     "histogram",
		})),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	grpcServer := grpc.NewServer()

	workflowService := workflow.NewService(database, logger, temporalClient, config)

	server := &GRpcServer{
		temporalClient:  temporalClient,
		config:          config,
		db:              database,
		grpcServer:      grpcServer,
		logger:          logger,
		stopCh:          make(chan struct{}),
		workflowService: workflowService,
	}

	pb.RegisterIronbirdServiceServer(grpcServer, workflowService)
	reflection.Register(grpcServer)

	go server.startWorkflowStatusUpdater()

	return server, nil
}

func (s *GRpcServer) Start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	wrappedGrpc := grpcweb.WrapServer(s.grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.logger.Info("HTTP request received",
			zap.String("method", r.Method),
			zap.String("url", r.URL.String()),
			zap.String("path", r.URL.Path),
		)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-Agent, X-Grpc-Web")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		wrappedGrpc.ServeHTTP(w, r)
	})

	s.logger.Info("gRpc server with gRpc-Web support listening", zap.String("address", address))
	return http.Serve(lis, handler)
}

func (s *GRpcServer) Stop() {
	close(s.stopCh)

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	if s.temporalClient != nil {
		s.temporalClient.Close()
	}
}

func (s *GRpcServer) startWorkflowStatusUpdater() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	s.logger.Info("starting workflow status updater background process")

	for {
		select {
		case <-ticker.C:
			s.workflowService.UpdateWorkflowStatuses()
		case <-s.stopCh:
			s.logger.Info("stopping workflow status updater background process")
			return
		}
	}
}

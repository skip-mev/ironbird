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

type GRPCServer struct {
	temporalClient  temporalclient.Client
	db              db.DB
	grpcServer      *grpc.Server
	logger          *zap.Logger
	stopCh          chan struct{}
	workflowService *workflow.Service
}

func NewGRPCServer(config types.TemporalConfig, database db.DB, logger *zap.Logger) (*GRPCServer, error) {
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
	logger.Info("Creating new workflow service", zap.Any("temporal_config", config))
	workflowService := workflow.NewService(database, logger, temporalClient)

	server := &GRPCServer{
		temporalClient:  temporalClient,
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

func (s *GRPCServer) Start(address string, webAddress string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		s.logger.Info("gRPC server listening", zap.String("address", address))
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Error("gRPC server error", zap.Error(err))
		}
	}()

	webLis, err := net.Listen("tcp", webAddress)
	if err != nil {
		s.logger.Warn("Failed to start gRPC-Web server", zap.Error(err))
		return nil
	}

	wrappedGPPC := grpcweb.WrapServer(s.grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
	)

	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-Agent, X-Grpc-Web")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		wrappedGPPC.ServeHTTP(w, r)
	})

	s.logger.Info("gRPC-Web server listening", zap.String("address", webAddress))
	return http.Serve(webLis, httpHandler)
}

func (s *GRPCServer) Stop() {
	close(s.stopCh)

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	if s.temporalClient != nil {
		s.temporalClient.Close()
	}
}

func (s *GRPCServer) startWorkflowStatusUpdater() {
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

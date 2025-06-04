package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/core/db"
	"github.com/skip-mev/ironbird/core/messages"
	"github.com/skip-mev/ironbird/core/types"
	"github.com/skip-mev/ironbird/core/util"
	"github.com/skip-mev/ironbird/core/workflows/testnet"
	pb "github.com/skip-mev/ironbird/server/proto"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"github.com/uber-go/tally/v4/prometheus"
	"go.temporal.io/api/enums/v1"
	temporalclient "go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type WorkflowResponse struct {
	WorkflowID string `json:"WorkflowID"`
	Status     string `json:"Status"`
}

type Workflow struct {
	WorkflowID    string                          `json:"WorkflowID"`
	Status        string                          `json:"Status"`
	Nodes         []messages.Node                 `json:"Nodes"`
	Validators    []messages.Node                 `json:"Validators"`
	LoadBalancers []messages.Node                 `json:"LoadBalancers"`
	Monitoring    map[string]string               `json:"Monitoring"`
	Config        messages.TestnetWorkflowRequest `json:"Config,omitempty"`
	LoadTestSpec  json.RawMessage                 `json:"loadTestSpec,omitempty"`
}

type WorkflowSummary struct {
	WorkflowID string `json:"WorkflowID"`
	Status     string `json:"Status"`
	StartTime  string `json:"StartTime"`
	Repo       string `json:"Repo,omitempty"`
	SHA        string `json:"SHA,omitempty"`
}

type WorkflowListResponse struct {
	Workflows []WorkflowSummary `json:"Workflows"`
	Count     int               `json:"Count"`
}

type GRPCServer struct {
	pb.UnimplementedIronbirdServiceServer
	temporalClient temporalclient.Client
	config         types.TemporalConfig
	db             db.DB
	grpcServer     *grpc.Server
	logger         *zap.Logger
	stopCh         chan struct{}
}

func NewGRPCServer(config types.TemporalConfig, database db.DB, logger *zap.Logger) (*GRPCServer, error) {
	temporalClient, err := temporalclient.Dial(temporalclient.Options{
		HostPort:  config.Host,
		Namespace: config.Namespace,
		MetricsHandler: sdktally.NewMetricsHandler(util.NewPrometheusScope(prometheus.Configuration{
			ListenAddress: "0.0.0.0:9091",
			TimerType:     "histogram",
		})),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create temporal client: %w", err)
	}

	grpcServer := grpc.NewServer()
	server := &GRPCServer{
		temporalClient: temporalClient,
		config:         config,
		db:             database,
		grpcServer:     grpcServer,
		logger:         logger,
		stopCh:         make(chan struct{}),
	}

	pb.RegisterIronbirdServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	go server.startWorkflowStatusUpdater()

	return server, nil
}

func (s *GRPCServer) Start(address string) error {
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

	s.logger.Info("gRPC server with gRPC-Web support listening", zap.String("address", address))
	return http.Serve(lis, handler)
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
			s.updateWorkflowStatuses()
		case <-s.stopCh:
			s.logger.Info("stopping workflow status updater background process")
			return
		}
	}
}

func (s *GRPCServer) updateWorkflowStatuses() {
	workflows, err := s.db.ListWorkflows(1000, 0)
	if err != nil {
		s.logger.Error("Error listing workflows from database", zap.Error(err))
		return
	}

	for _, workflow := range workflows {
		if workflow.Status == enums.WORKFLOW_EXECUTION_STATUS_COMPLETED ||
			workflow.Status == enums.WORKFLOW_EXECUTION_STATUS_FAILED ||
			workflow.Status == enums.WORKFLOW_EXECUTION_STATUS_CANCELED ||
			workflow.Status == enums.WORKFLOW_EXECUTION_STATUS_TERMINATED {
			continue
		}

		workflowID := workflow.WorkflowID
		desc, err := s.temporalClient.DescribeWorkflowExecution(
			context.Background(),
			workflowID,
			"", // Empty run ID to get the latest run
		)

		if err != nil {
			s.logger.Error("Error describing workflow",
				zap.String("workflowID", workflowID),
				zap.Error(err))
			continue
		}

		var newStatus db.WorkflowStatus
		newStatus = desc.WorkflowExecutionInfo.Status

		if newStatus != workflow.Status {
			s.logger.Info("updating workflow status",
				zap.String("workflowID", workflowID),
				zap.String("oldStatus", db.WorkflowStatusToString(workflow.Status)),
				zap.String("newStatus", db.WorkflowStatusToString(newStatus)))

			update := db.WorkflowUpdate{
				Status: &newStatus,
			}

			if err := s.db.UpdateWorkflow(workflowID, update); err != nil {
				s.logger.Error("updating workflow status",
					zap.String("workflowID", workflowID),
					zap.Error(err))
			}
		}
	}
}

func (s *GRPCServer) CreateWorkflow(ctx context.Context, req *pb.CreateWorkflowRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("CreateWorkflow request received", zap.Any("request", req))

	workflowReq := messages.TestnetWorkflowRequest{
		Repo:               req.Repo,
		SHA:                req.Sha,
		Evm:                req.Evm,
		RunnerType:         messages.RunnerType(req.RunnerType),
		LongRunningTestnet: req.LongRunningTestnet,
		TestnetDuration:    time.Duration(req.TestnetDuration),
		NumWallets:         int(req.NumWallets),
	}

	if req.ChainConfig != nil {
		chainConfig := types.ChainsConfig{
			Name:            req.ChainConfig.Name,
			Image:           req.ChainConfig.Image,
			NumOfNodes:      req.ChainConfig.NumOfNodes,
			NumOfValidators: req.ChainConfig.NumOfValidators,
		}

		if req.ChainConfig.GenesisModifications != nil {
			s.logger.Info("Processing genesis modifications", zap.Int("count", len(req.ChainConfig.GenesisModifications)))

			for _, gm := range req.ChainConfig.GenesisModifications {
				var jsonValue interface{}
				if err := json.Unmarshal([]byte(gm.Value), &jsonValue); err == nil {
					chainConfig.GenesisModifications = append(
						chainConfig.GenesisModifications,
						chain.GenesisKV{
							Key:   gm.Key,
							Value: jsonValue,
						},
					)
				} else {
					chainConfig.GenesisModifications = append(
						chainConfig.GenesisModifications,
						chain.GenesisKV{
							Key:   gm.Key,
							Value: gm.Value,
						},
					)
				}
			}

			s.logger.Info("Processed genesis modifications",
				zap.Int("count", len(chainConfig.GenesisModifications)))
		} else {
			s.logger.Info("No genesis modifications provided")
		}

		workflowReq.ChainConfig = chainConfig
	}

	if req.LoadTestSpec != nil {
		var loadTestSpec catalysttypes.LoadTestSpec
		loadTestSpec = convertProtoLoadTestSpec(req.LoadTestSpec)
		workflowReq.LoadTestSpec = &loadTestSpec
	}

	options := temporalclient.StartWorkflowOptions{
		TaskQueue: messages.TaskQueue,
	}

	workflowRun, err := s.temporalClient.ExecuteWorkflow(ctx, options, testnet.Workflow, workflowReq)
	if err != nil {
		s.logger.Error("executing workflow", zap.Error(err))
		return nil, fmt.Errorf("failed to execute workflow: %w", err)
	}

	workflowID := workflowRun.GetID()
	s.logger.Info("workflow execution started", zap.String("workflowID", workflowID))

	workflow := &db.Workflow{
		WorkflowID:      workflowID,
		Nodes:           []messages.Node{},
		Validators:      []messages.Node{},
		LoadBalancers:   []messages.Node{},
		MonitoringLinks: make(map[string]string),
		Status:          enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		Config:          workflowReq,
	}

	if err := s.db.CreateWorkflow(workflow); err != nil {
		s.logger.Error("creating workflow record", zap.Error(err))
	}

	return &pb.WorkflowResponse{
		WorkflowId: workflowID,
		Status:     "running",
	}, nil
}

func (s *GRPCServer) GetWorkflow(ctx context.Context, req *pb.GetWorkflowRequest) (*pb.Workflow, error) {
	s.logger.Info("GetWorkflow request received", zap.String("workflowID", req.WorkflowId))

	workflow, err := s.db.GetWorkflow(req.WorkflowId)
	if err != nil {
		s.logger.Error("failed to get workflow", zap.Error(err), zap.String("workflowID", req.WorkflowId))
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	desc, err := s.temporalClient.DescribeWorkflowExecution(
		ctx,
		req.WorkflowId,
		"", // Empty run ID to get the latest run
	)
	if err != nil {
		s.logger.Error("failed to describe workflow", zap.Error(err), zap.String("workflowID", req.WorkflowId))
		return nil, fmt.Errorf("failed to describe workflow: %w", err)
	}

	status := db.WorkflowStatusToString(desc.WorkflowExecutionInfo.Status)

	response := &pb.Workflow{
		WorkflowId: req.WorkflowId,
		Status:     status,
	}

	if workflow.Nodes != nil {
		for _, node := range workflow.Nodes {
			response.Nodes = append(response.Nodes, &pb.Node{
				Name:    node.Name,
				Address: node.Address,
				Rpc:     node.RPC,
				Lcd:     node.LCD,
			})
		}
	}

	if workflow.Validators != nil {
		for _, validator := range workflow.Validators {
			response.Validators = append(response.Validators, &pb.Node{
				Name:    validator.Name,
				Address: validator.Address,
				Rpc:     validator.RPC,
				Lcd:     validator.LCD,
			})
		}
	}

	if workflow.LoadBalancers != nil {
		for _, lb := range workflow.LoadBalancers {
			response.LoadBalancers = append(response.LoadBalancers, &pb.Node{
				Name:    lb.Name,
				Address: lb.Address,
				Rpc:     lb.RPC,
				Lcd:     lb.LCD,
			})
		}
	}

	if workflow.MonitoringLinks != nil {
		response.Monitoring = workflow.MonitoringLinks
	}

	if workflow.LoadTestSpec != nil {
		var loadTestSpec catalysttypes.LoadTestSpec
		if err := json.Unmarshal(workflow.LoadTestSpec, &loadTestSpec); err == nil {
			response.LoadTestSpec = convertCatalystLoadTestSpecToProto(&loadTestSpec)
		}
	}

	chainConfig := &pb.ChainConfig{
		Name:            workflow.Config.ChainConfig.Name,
		NumOfNodes:      workflow.Config.ChainConfig.NumOfNodes,
		NumOfValidators: workflow.Config.ChainConfig.NumOfValidators,
		Image:           workflow.Config.ChainConfig.Image,
	}

	if workflow.Config.ChainConfig.GenesisModifications != nil {
		s.logger.Info("Found genesis modifications in workflow config",
			zap.Int("count", len(workflow.Config.ChainConfig.GenesisModifications)))

		for _, gm := range workflow.Config.ChainConfig.GenesisModifications {
			var valueStr string
			switch v := gm.Value.(type) {
			case string:
				valueStr = v
			default:
				valueBytes, err := json.Marshal(gm.Value)
				if err != nil {
					s.logger.Warn("Failed to marshal genesis value", zap.String("key", gm.Key), zap.Error(err))
					continue
				}
				valueStr = string(valueBytes)
			}

			chainConfig.GenesisModifications = append(
				chainConfig.GenesisModifications,
				&pb.GenesisKV{
					Key:   gm.Key,
					Value: valueStr,
				},
			)
		}
	}

	response.Config = &pb.CreateWorkflowRequest{
		Repo:               workflow.Config.Repo,
		Sha:                workflow.Config.SHA,
		Evm:                workflow.Config.Evm,
		RunnerType:         string(workflow.Config.RunnerType),
		LongRunningTestnet: workflow.Config.LongRunningTestnet,
		TestnetDuration:    int64(workflow.Config.TestnetDuration.Seconds()),
		NumWallets:         int32(workflow.Config.NumWallets),
		ChainConfig:        chainConfig,
	}

	return response, nil
}

func (s *GRPCServer) ListWorkflows(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.WorkflowListResponse, error) {
	s.logger.Info("ListWorkflows request received", zap.Int32("limit", req.Limit), zap.Int32("offset", req.Offset))

	workflows, err := s.db.ListWorkflows(int(req.Limit), int(req.Offset))
	if err != nil {
		s.logger.Error("failed to list workflows", zap.Error(err))
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	s.logger.Info("Retrieved workflows from database",
		zap.Int("count", len(workflows)),
		zap.Int("requested_limit", int(req.Limit)),
		zap.Int("requested_offset", int(req.Offset)),
	)

	response := &pb.WorkflowListResponse{
		Count: int32(len(workflows)),
	}

	for _, workflow := range workflows {
		status := db.WorkflowStatusToString(workflow.Status)
		startTime := workflow.CreatedAt.Format("2006-01-02 15:04:05")

		s.logger.Debug("Adding workflow to response",
			zap.String("workflowId", workflow.WorkflowID),
			zap.String("status", status),
			zap.String("startTime", startTime),
		)

		response.Workflows = append(response.Workflows, &pb.WorkflowSummary{
			WorkflowId: workflow.WorkflowID,
			Status:     status,
			StartTime:  startTime,
			Repo:       workflow.Config.Repo,
			Sha:        workflow.Config.SHA,
		})
	}

	s.logger.Info("Returning ListWorkflows response", zap.Int32("totalCount", response.Count))

	return response, nil
}

func (s *GRPCServer) CancelWorkflow(ctx context.Context, req *pb.CancelWorkflowRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("CancelWorkflow request received", zap.String("workflowID", req.WorkflowId))

	err := s.temporalClient.CancelWorkflow(ctx, req.WorkflowId, "")
	if err != nil {
		s.logger.Error("failed to cancel workflow", zap.Error(err), zap.String("workflowID", req.WorkflowId))
		return nil, fmt.Errorf("failed to cancel workflow: %w", err)
	}

	return &pb.WorkflowResponse{
		WorkflowId: req.WorkflowId,
		Status:     "canceled",
	}, nil
}

func (s *GRPCServer) SignalWorkflow(ctx context.Context, req *pb.SignalWorkflowRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("SignalWorkflow request received", zap.String("workflowID", req.WorkflowId), zap.String("signalName", req.SignalName))

	err := s.temporalClient.SignalWorkflow(ctx, req.WorkflowId, "", req.SignalName, nil)
	if err != nil {
		s.logger.Error("failed to signal workflow", zap.Error(err), zap.String("workflowID", req.WorkflowId), zap.String("signalName", req.SignalName))
		return nil, fmt.Errorf("failed to signal workflow: %w", err)
	}

	return &pb.WorkflowResponse{
		WorkflowId: req.WorkflowId,
		Status:     "signaled",
	}, nil
}

func (s *GRPCServer) RunLoadTest(ctx context.Context, req *pb.RunLoadTestRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("RunLoadTest request received", zap.String("workflowID", req.WorkflowId))

	return &pb.WorkflowResponse{
		WorkflowId: req.WorkflowId,
		Status:     "not_implemented",
	}, nil
}

func convertProtoLoadTestSpec(spec *pb.LoadTestSpec) catalysttypes.LoadTestSpec {
	if spec == nil {
		return catalysttypes.LoadTestSpec{}
	}

	result := catalysttypes.LoadTestSpec{
		Name:         spec.Name,
		Description:  spec.Description,
		ChainID:      spec.ChainId,
		NumOfTxs:     int(spec.NumOfTxs),
		NumOfBlocks:  int(spec.NumOfBlocks),
		GasDenom:     spec.GasDenom,
		Bech32Prefix: spec.Bech32Prefix,
		UnorderedTxs: spec.UnorderedTxs,
		TxTimeout:    time.Duration(spec.TxTimeout),
	}

	if spec.NodesAddresses != nil {
		for _, addr := range spec.NodesAddresses {
			result.NodesAddresses = append(result.NodesAddresses, catalysttypes.NodeAddress{
				GRPC: addr.Grpc,
				RPC:  addr.Rpc,
			})
		}
	}

	if spec.Mnemonics != nil {
		result.Mnemonics = spec.Mnemonics
	}

	if spec.Msgs != nil {
		for _, msg := range spec.Msgs {
			result.Msgs = append(result.Msgs, catalysttypes.LoadTestMsg{
				Weight:          float64(msg.Weight),
				Type:            catalysttypes.MsgType(msg.Type),
				NumMsgs:         int(msg.NumMsgs),
				ContainedType:   catalysttypes.MsgType(msg.ContainedType),
				NumOfRecipients: int(msg.NumOfRecipients),
			})
		}
	}

	return result
}

func convertCatalystLoadTestSpecToProto(spec *catalysttypes.LoadTestSpec) *pb.LoadTestSpec {
	if spec == nil {
		return nil
	}

	result := &pb.LoadTestSpec{
		Name:         spec.Name,
		Description:  spec.Description,
		ChainId:      spec.ChainID,
		NumOfTxs:     int32(spec.NumOfTxs),
		NumOfBlocks:  int32(spec.NumOfBlocks),
		GasDenom:     spec.GasDenom,
		Bech32Prefix: spec.Bech32Prefix,
		UnorderedTxs: spec.UnorderedTxs,
		TxTimeout:    int64(spec.TxTimeout.Seconds()),
	}

	if spec.NodesAddresses != nil {
		for _, addr := range spec.NodesAddresses {
			result.NodesAddresses = append(result.NodesAddresses, &pb.NodeAddress{
				Grpc: addr.GRPC,
				Rpc:  addr.RPC,
			})
		}
	}

	if spec.Mnemonics != nil {
		result.Mnemonics = spec.Mnemonics
	}

	if spec.Msgs != nil {
		for _, msg := range spec.Msgs {
			result.Msgs = append(result.Msgs, &pb.LoadTestMsg{
				Weight:          float32(msg.Weight),
				Type:            msg.Type.String(),
				NumMsgs:         int32(msg.NumMsgs),
				ContainedType:   msg.ContainedType.String(),
				NumOfRecipients: int32(msg.NumOfRecipients),
			})
		}
	}

	return result
}

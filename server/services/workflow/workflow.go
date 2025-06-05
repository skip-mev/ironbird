package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/workflows/testnet"

	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/server/db"
	pb "github.com/skip-mev/ironbird/server/proto"
	"github.com/skip-mev/petri/cosmos/v3/chain"
	"go.temporal.io/api/enums/v1"
	temporalclient "go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type Service struct {
	pb.UnimplementedIronbirdServiceServer
	db             db.DB
	logger         *zap.Logger
	temporalClient temporalclient.Client
	config         types.TemporalConfig
}

func NewService(database db.DB, logger *zap.Logger, temporalClient temporalclient.Client, config types.TemporalConfig) *Service {
	return &Service{
		db:             database,
		logger:         logger,
		temporalClient: temporalClient,
		config:         config,
	}
}

func (s *Service) CreateWorkflow(ctx context.Context, req *pb.CreateWorkflowRequest) (*pb.WorkflowResponse, error) {
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
			for _, gm := range req.ChainConfig.GenesisModifications {
				if isNumericString(gm.Value) {
					// Keep numeric strings as strings to avoid precision issues in genesis
					chainConfig.GenesisModifications = append(
						chainConfig.GenesisModifications,
						chain.GenesisKV{
							Key:   gm.Key,
							Value: gm.Value,
						},
					)
				} else {
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
			}

			s.logger.Info("Processed genesis modifications",
				zap.Int("count", len(chainConfig.GenesisModifications)))
		} else {
			s.logger.Info("No genesis modifications provided")
		}

		workflowReq.ChainConfig = chainConfig
	}

	if req.LoadTestSpec != nil {
		loadTestSpec := s.convertProtoLoadTestSpec(req.LoadTestSpec)
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
		Nodes:           []*pb.Node{},
		Validators:      []*pb.Node{},
		LoadBalancers:   []*pb.Node{},
		MonitoringLinks: make(map[string]string),
		Status:          enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		Config:          workflowReq,
	}

	if err := s.db.CreateWorkflow(workflow); err != nil {
		s.logger.Error("creating workflow record", zap.Error(err))
	}

	return &pb.WorkflowResponse{
		WorkflowId: workflowID,
	}, nil
}

func (s *Service) GetWorkflow(ctx context.Context, req *pb.GetWorkflowRequest) (*pb.Workflow, error) {
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

	response.Nodes = workflow.Nodes
	response.Validators = workflow.Validators
	response.LoadBalancers = workflow.LoadBalancers

	if workflow.MonitoringLinks != nil {
		response.Monitoring = workflow.MonitoringLinks
	}

	if workflow.LoadTestSpec != nil {
		var loadTestSpec catalysttypes.LoadTestSpec
		if err := json.Unmarshal(workflow.LoadTestSpec, &loadTestSpec); err == nil {
			response.LoadTestSpec = s.convertCatalystLoadTestSpecToProto(&loadTestSpec)
		}
	}

	chainConfig := &pb.ChainConfig{
		Name:            workflow.Config.ChainConfig.Name,
		NumOfNodes:      workflow.Config.ChainConfig.NumOfNodes,
		NumOfValidators: workflow.Config.ChainConfig.NumOfValidators,
		Image:           workflow.Config.ChainConfig.Image,
	}

	if workflow.Config.ChainConfig.GenesisModifications != nil {
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

func (s *Service) ListWorkflows(ctx context.Context, req *pb.ListWorkflowsRequest) (*pb.WorkflowListResponse, error) {
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

func (s *Service) CancelWorkflow(ctx context.Context, req *pb.CancelWorkflowRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("CancelWorkflow request received", zap.String("workflowID", req.WorkflowId))

	err := s.temporalClient.CancelWorkflow(ctx, req.WorkflowId, "")
	if err != nil {
		s.logger.Error("failed to cancel workflow", zap.Error(err), zap.String("workflowID", req.WorkflowId))
		return nil, fmt.Errorf("failed to cancel workflow: %w", err)
	}

	return &pb.WorkflowResponse{
		WorkflowId: req.WorkflowId,
	}, nil
}

func (s *Service) SignalWorkflow(ctx context.Context, req *pb.SignalWorkflowRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("SignalWorkflow request received", zap.String("workflowID", req.WorkflowId), zap.String("signalName", req.SignalName))

	err := s.temporalClient.SignalWorkflow(ctx, req.WorkflowId, "", req.SignalName, nil)
	if err != nil {
		s.logger.Error("failed to signal workflow", zap.Error(err), zap.String("workflowID", req.WorkflowId), zap.String("signalName", req.SignalName))
		return nil, fmt.Errorf("failed to signal workflow: %w", err)
	}

	return &pb.WorkflowResponse{
		WorkflowId: req.WorkflowId,
	}, nil
}

func (s *Service) RunLoadTest(ctx context.Context, req *pb.RunLoadTestRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("RunLoadTest request received", zap.String("workflowID", req.WorkflowId))

	return &pb.WorkflowResponse{
		WorkflowId: req.WorkflowId,
	}, nil
}

func (s *Service) UpdateWorkflowData(ctx context.Context, req *pb.UpdateWorkflowDataRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("UpdateWorkflowData request received",
		zap.String("workflowID", req.WorkflowId),
		zap.Int("loadBalancers", len(req.LoadBalancers)),
		zap.Int("monitoringLinks", len(req.Monitoring)),
		zap.Int("nodes", len(req.Nodes)),
		zap.Int("validators", len(req.Validators)))

	loadBalancers := convertProtoNodes(req.LoadBalancers)
	nodes := convertProtoNodes(req.Nodes)
	validators := convertProtoNodes(req.Validators)

	update := db.WorkflowUpdate{}

	if len(loadBalancers) > 0 {
		update.LoadBalancers = &loadBalancers
	}

	if len(req.Monitoring) > 0 {
		update.MonitoringLinks = &req.Monitoring
	}

	if len(nodes) > 0 {
		update.Nodes = &nodes
	}

	if len(validators) > 0 {
		update.Validators = &validators
	}

	if err := s.db.UpdateWorkflow(req.WorkflowId, update); err != nil {
		s.logger.Error("Failed to update workflow data", zap.Error(err))
		return nil, fmt.Errorf("failed to update workflow data: %w", err)
	}

	s.logger.Info("Successfully updated workflow data", zap.String("workflowID", req.WorkflowId))

	return &pb.WorkflowResponse{
		WorkflowId: req.WorkflowId,
	}, nil
}

func (s *Service) UpdateWorkflowStatuses() {
	workflows, err := s.db.ListWorkflows(1000, 0)
	if err != nil {
		s.logger.Error("Error listing workflows from database", zap.Error(err))
		return
	}

	for _, workflow := range workflows {
		if isWorkflowTerminal(workflow.Status) {
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

		newStatus := desc.WorkflowExecutionInfo.Status

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

func (s *Service) convertProtoLoadTestSpec(spec *pb.LoadTestSpec) catalysttypes.LoadTestSpec {
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

	result.NodesAddresses = convertProtoNodeAddresses(spec.NodesAddresses)
	result.Mnemonics = spec.Mnemonics
	result.Msgs = convertProtoLoadTestMsgs(spec.Msgs)

	return result
}

func isNumericString(s string) bool {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	isNumeric := true
	for i, c := range s {
		if (c < '0' || c > '9') && c != '.' && (i > 0 || c != '-') {
			isNumeric = false
			break
		}
	}

	return isNumeric
}

func (s *Service) convertCatalystLoadTestSpecToProto(spec *catalysttypes.LoadTestSpec) *pb.LoadTestSpec {
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

	result.NodesAddresses = convertCatalystNodeAddresses(spec.NodesAddresses)
	result.Mnemonics = spec.Mnemonics
	result.Msgs = convertCatalystLoadTestMsgs(spec.Msgs)

	return result
}

func convertProtoNodes(protoNodes []*pb.Node) []pb.Node {
	if protoNodes == nil {
		return nil
	}

	var result []pb.Node
	for i := range protoNodes {
		result = append(result, pb.Node{
			Name:    protoNodes[i].Name,
			Address: protoNodes[i].Address,
			Rpc:     protoNodes[i].Rpc,
			Lcd:     protoNodes[i].Lcd,
		})
	}
	return result
}

func convertProtoNodeAddresses(protoAddrs []*pb.NodeAddress) []catalysttypes.NodeAddress {
	if protoAddrs == nil {
		return nil
	}

	var result []catalysttypes.NodeAddress
	for _, addr := range protoAddrs {
		result = append(result, catalysttypes.NodeAddress{
			GRPC: addr.Grpc,
			RPC:  addr.Rpc,
		})
	}
	return result
}

func convertCatalystNodeAddresses(addrs []catalysttypes.NodeAddress) []*pb.NodeAddress {
	if addrs == nil {
		return nil
	}

	var result []*pb.NodeAddress
	for _, addr := range addrs {
		result = append(result, &pb.NodeAddress{
			Grpc: addr.GRPC,
			Rpc:  addr.RPC,
		})
	}
	return result
}

func convertProtoLoadTestMsgs(protoMsgs []*pb.LoadTestMsg) []catalysttypes.LoadTestMsg {
	if protoMsgs == nil {
		return nil
	}

	var result []catalysttypes.LoadTestMsg
	for _, msg := range protoMsgs {
		result = append(result, catalysttypes.LoadTestMsg{
			Weight:          float64(msg.Weight),
			Type:            catalysttypes.MsgType(msg.Type),
			NumMsgs:         int(msg.NumMsgs),
			ContainedType:   catalysttypes.MsgType(msg.ContainedType),
			NumOfRecipients: int(msg.NumOfRecipients),
		})
	}
	return result
}

func convertCatalystLoadTestMsgs(msgs []catalysttypes.LoadTestMsg) []*pb.LoadTestMsg {
	if msgs == nil {
		return nil
	}

	var result []*pb.LoadTestMsg
	for _, msg := range msgs {
		result = append(result, &pb.LoadTestMsg{
			Weight:          float32(msg.Weight),
			Type:            msg.Type.String(),
			NumMsgs:         int32(msg.NumMsgs),
			ContainedType:   msg.ContainedType.String(),
			NumOfRecipients: int32(msg.NumOfRecipients),
		})
	}
	return result
}

func isWorkflowTerminal(status enums.WorkflowExecutionStatus) bool {
	return status == enums.WORKFLOW_EXECUTION_STATUS_COMPLETED ||
		status == enums.WORKFLOW_EXECUTION_STATUS_FAILED ||
		status == enums.WORKFLOW_EXECUTION_STATUS_CANCELED ||
		status == enums.WORKFLOW_EXECUTION_STATUS_TERMINATED
}

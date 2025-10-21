package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/workflows/testnet"
	"gopkg.in/yaml.v3"

	cosmostypes "github.com/skip-mev/catalyst/chains/cosmos/types"
	ethtypes "github.com/skip-mev/catalyst/chains/ethereum/types"
	catalysttypes "github.com/skip-mev/catalyst/chains/types"
	petritypes "github.com/skip-mev/ironbird/petri/core/types"
	"github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/skip-mev/ironbird/server/db"
	pb "github.com/skip-mev/ironbird/server/proto"
	"go.temporal.io/api/enums/v1"
	temporalclient "go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type Service struct {
	pb.UnimplementedIronbirdServiceServer
	db             db.DB
	logger         *zap.Logger
	temporalClient temporalclient.Client
}

func NewService(database db.DB, logger *zap.Logger, temporalClient temporalclient.Client) *Service {
	cosmostypes.Register()
	ethtypes.Register()
	return &Service{
		db:             database,
		logger:         logger,
		temporalClient: temporalClient,
	}
}

const DefaultBaseMnemonic = "copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"

func (s *Service) CreateWorkflow(ctx context.Context, req *pb.CreateWorkflowRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("CreateWorkflow request received", zap.Any("request", req))

	if req.TestnetDuration != "" {
		_, err := time.ParseDuration(req.TestnetDuration)
		if err != nil {
			return nil, fmt.Errorf("invalid testnet duration format '%s': %w", req.TestnetDuration, err)
		}
	}
	if req.GetBaseMnemonic() == "" {
		req.BaseMnemonic = DefaultBaseMnemonic
	}

	workflowReq := messages.TestnetWorkflowRequest{
		Repo:                   req.Repo,
		SHA:                    req.Sha,
		CosmosSdkSha:           req.CosmosSdkSha,
		CometBFTSha:            req.CometbftSha,
		IsEvmChain:             req.IsEvmChain,
		RunnerType:             messages.RunnerType(req.RunnerType),
		LongRunningTestnet:     req.LongRunningTestnet,
		LaunchLoadBalancer:     req.LaunchLoadBalancer,
		TestnetDuration:        req.TestnetDuration,
		NumWallets:             int(req.NumWallets),
		BaseMnemonic:           req.BaseMnemonic,
		CatalystVersion:        req.CatalystVersion,
		ProviderSpecificConfig: req.ProviderConfig,
	}

	if req.ChainConfig != nil {
		chainConfig := types.ChainsConfig{
			Name:                  req.ChainConfig.Name,
			Image:                 req.ChainConfig.Image,
			Version:               req.ChainConfig.Version,
			NumOfNodes:            req.ChainConfig.NumOfNodes,
			NumOfValidators:       req.ChainConfig.NumOfValidators,
			SetSeedNode:           req.ChainConfig.SetSeedNode,
			SetPersistentPeers:    req.ChainConfig.SetPersistentPeers,
			CustomAppConfig:       s.parseJSONConfig(req.ChainConfig.CustomAppConfig, "custom_app_config"),
			CustomConsensusConfig: s.parseJSONConfig(req.ChainConfig.CustomConsensusConfig, "custom_consensus_config"),
			CustomClientConfig:    s.parseJSONConfig(req.ChainConfig.CustomClientConfig, "custom_client_config"),
		}

		if req.ChainConfig.RegionConfigs != nil {
			for _, rc := range req.ChainConfig.RegionConfigs {
				chainConfig.RegionConfigs = append(chainConfig.RegionConfigs, petritypes.RegionConfig{
					Name:          rc.Name,
					NumNodes:      int(rc.NumOfNodes),
					NumValidators: int(rc.NumOfValidators),
				})
			}
		}

		if !chainConfig.SetSeedNode && !chainConfig.SetPersistentPeers {
			return nil, fmt.Errorf("at least one of SetSeedNode or SetPersistentPeers must be set to true")
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

	if len(req.EncodedLoadTestSpec) != 0 {
		loadTestSpec, err := decodeLoadTestSpec(req.EncodedLoadTestSpec)
		if err != nil {
			return nil, err
		}

		switch loadTestSpec.Kind {
		case "eth":
			workflowReq.EthereumLoadTestSpec = &loadTestSpec
		case "cosmos":
			workflowReq.CosmosLoadTestSpec = &loadTestSpec
		}
	}

	if err := workflowReq.Validate(); err != nil {
		s.logger.Error("workflow request validation failed", zap.Error(err))
		return nil, fmt.Errorf("workflow request validation failed: %w", err)
	}

	options := temporalclient.StartWorkflowOptions{
		TaskQueue:           messages.TaskQueue,
		WorkflowTaskTimeout: 30 * time.Second,
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 90*time.Minute)
	defer cancel()
	workflowRun, err := s.temporalClient.ExecuteWorkflow(ctxWithTimeout, options, testnet.Workflow, workflowReq)
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

	var startTimeStr, endTimeStr string
	if desc.WorkflowExecutionInfo.StartTime != nil {
		startTimeStr = desc.WorkflowExecutionInfo.StartTime.AsTime().Format(time.RFC3339)
	}
	if desc.WorkflowExecutionInfo.CloseTime != nil {
		endTimeStr = desc.WorkflowExecutionInfo.CloseTime.AsTime().Format(time.RFC3339)
	}

	response := &pb.Workflow{
		WorkflowId: req.WorkflowId,
		Status:     status,
		Provider:   workflow.Provider,
		StartTime:  startTimeStr,
		EndTime:    endTimeStr,
	}

	response.Nodes = workflow.Nodes
	response.Validators = workflow.Validators
	response.LoadBalancers = workflow.LoadBalancers

	if workflow.Wallets != nil {
		response.Wallets = workflow.Wallets
	}

	if workflow.MonitoringLinks != nil {
		response.Monitoring = workflow.MonitoringLinks
	}

	if workflow.LoadTestSpec != nil {
		var loadTestSpec catalysttypes.LoadTestSpec
		if err := json.Unmarshal(workflow.LoadTestSpec, &loadTestSpec); err == nil {
			encodedSpec, err := encodeLoadTestSpec(loadTestSpec)
			if err != nil {
				return nil, err
			}
			response.LoadTestSpec = encodedSpec
		}
	}

	chainConfig := &pb.ChainConfig{
		Name:                  workflow.Config.ChainConfig.Name,
		Image:                 workflow.Config.ChainConfig.Image,
		Version:               workflow.Config.ChainConfig.Version,
		NumOfNodes:            workflow.Config.ChainConfig.NumOfNodes,
		NumOfValidators:       workflow.Config.ChainConfig.NumOfValidators,
		SetSeedNode:           workflow.Config.ChainConfig.SetSeedNode,
		SetPersistentPeers:    workflow.Config.ChainConfig.SetPersistentPeers,
		CustomAppConfig:       marshalJSONConfig(workflow.Config.ChainConfig.CustomAppConfig),
		CustomConsensusConfig: marshalJSONConfig(workflow.Config.ChainConfig.CustomConsensusConfig),
		CustomClientConfig:    marshalJSONConfig(workflow.Config.ChainConfig.CustomClientConfig),
	}

	for _, rc := range workflow.Config.ChainConfig.RegionConfigs {
		chainConfig.RegionConfigs = append(chainConfig.RegionConfigs, &pb.RegionConfig{
			Name:            rc.Name,
			NumOfNodes:      uint64(rc.NumNodes),
			NumOfValidators: uint64(rc.NumValidators),
		})
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
		CosmosSdkSha:       workflow.Config.CosmosSdkSha,
		CometbftSha:        workflow.Config.CometBFTSha,
		IsEvmChain:         workflow.Config.IsEvmChain,
		RunnerType:         string(workflow.Config.RunnerType),
		LongRunningTestnet: workflow.Config.LongRunningTestnet,
		LaunchLoadBalancer: workflow.Config.LaunchLoadBalancer,
		TestnetDuration:    workflow.Config.TestnetDuration,
		NumWallets:         int32(workflow.Config.NumWallets),
		BaseMnemonic:       workflow.Config.BaseMnemonic,
		CatalystVersion:    workflow.Config.CatalystVersion,
		ChainConfig:        chainConfig,
		ProviderConfig:     workflow.Config.ProviderSpecificConfig,
	}

	if workflow.Config.EthereumLoadTestSpec != nil {
		encodedSpec, err := encodeLoadTestSpec(*workflow.Config.EthereumLoadTestSpec)
		if err == nil {
			response.Config.EncodedLoadTestSpec = encodedSpec
		}
	} else if workflow.Config.CosmosLoadTestSpec != nil {
		encodedSpec, err := encodeLoadTestSpec(*workflow.Config.CosmosLoadTestSpec)
		if err == nil {
			response.Config.EncodedLoadTestSpec = encodedSpec
		}
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

		response.Workflows = append(response.Workflows, &pb.WorkflowSummary{
			WorkflowId: workflow.WorkflowID,
			Status:     status,
			StartTime:  startTime,
			Repo:       workflow.Config.Repo,
			Sha:        workflow.Config.SHA,
			Provider:   workflow.Provider,
			TemplateId: workflow.TemplateID,
			RunName:    workflow.RunName,
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

	if req.Wallets != nil {
		update.Wallets = req.Wallets
	}

	if req.Provider != "" {
		update.Provider = &req.Provider
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
			Grpc:    protoNodes[i].Grpc,
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

func (s *Service) parseJSONConfig(jsonStr, configType string) map[string]interface{} {
	if jsonStr == "" {
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		s.logger.Warn("Failed to parse "+configType+" JSON", zap.Error(err))
		return nil
	}

	return config
}

func marshalJSONConfig(config map[string]interface{}) string {
	if config == nil {
		return ""
	}

	if configBytes, err := json.Marshal(config); err == nil {
		return string(configBytes)
	}

	return ""
}

func decodeLoadTestSpec(s string) (catalysttypes.LoadTestSpec, error) {
	spec := catalysttypes.LoadTestSpec{}
	err := yaml.Unmarshal([]byte(s), &spec)
	return spec, err
}

func encodeLoadTestSpec(s catalysttypes.LoadTestSpec) (string, error) {
	bz, err := yaml.Marshal(s)
	return string(bz), err
}

func (s *Service) CreateWorkflowTemplate(ctx context.Context, req *pb.CreateWorkflowTemplateRequest) (*pb.WorkflowTemplateResponse, error) {
	s.logger.Info("CreateWorkflowTemplate request received", zap.Any("request", req))

	config := s.convertProtoToWorkflowRequest(req.TemplateConfig)

	err := s.db.CreateWorkflowTemplate(&db.WorkflowTemplate{
		ID:          req.Id,
		Description: req.Description,
		Config:      config,
	})
	if err != nil {
		s.logger.Error("Failed to create workflow template", zap.Error(err))
		return nil, fmt.Errorf("failed to create workflow template: %w", err)
	}

	return &pb.WorkflowTemplateResponse{
		Id: req.Id,
	}, nil
}

func (s *Service) GetWorkflowTemplate(ctx context.Context, req *pb.GetWorkflowTemplateRequest) (*pb.WorkflowTemplate, error) {
	s.logger.Info("GetWorkflowTemplate request received", zap.String("template_id", req.Id))

	template, err := s.db.GetWorkflowTemplate(req.Id)
	if err != nil {
		s.logger.Error("Failed to get workflow template", zap.Error(err))
		return nil, fmt.Errorf("failed to get workflow template: %w", err)
	}

	protoTemplate := s.convertTemplateToProto(template)
	return protoTemplate, nil
}

func (s *Service) ListWorkflowTemplates(ctx context.Context, req *pb.ListWorkflowTemplatesRequest) (*pb.WorkflowTemplateListResponse, error) {
	s.logger.Info("ListWorkflowTemplates request received", zap.Any("request", req))

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 100
	}

	templates, err := s.db.ListWorkflowTemplates(limit, int(req.Offset))
	if err != nil {
		s.logger.Error("Failed to list workflow templates", zap.Error(err))
		return nil, fmt.Errorf("failed to list workflow templates: %w", err)
	}

	protoTemplates := make([]*pb.WorkflowTemplateSummary, len(templates))
	for i, template := range templates {
		runCount, err := s.getTemplateRunCount(template.ID)
		if err != nil {
			s.logger.Error("Failed to get template run count", zap.Error(err))
			return nil, fmt.Errorf("failed to get template run count: %w", err)
		}
		protoTemplates[i] = &pb.WorkflowTemplateSummary{
			Id:          template.ID,
			Description: template.Description,
			CreatedAt:   template.CreatedAt.Format(time.RFC3339),
			RunCount:    int32(runCount),
		}
	}

	return &pb.WorkflowTemplateListResponse{
		Templates: protoTemplates,
		Count:     int32(len(protoTemplates)),
	}, nil
}

func (s *Service) UpdateWorkflowTemplate(ctx context.Context, req *pb.UpdateWorkflowTemplateRequest) (*pb.WorkflowTemplateResponse, error) {
	s.logger.Info("UpdateWorkflowTemplate request received", zap.String("template_id", req.Id))

	config := s.convertProtoToWorkflowRequest(req.TemplateConfig)

	template := &db.WorkflowTemplate{
		Description: req.Description,
		Config:      config,
	}

	err := s.db.UpdateWorkflowTemplate(req.Id, template)
	if err != nil {
		s.logger.Error("Failed to update workflow template", zap.Error(err))
		return nil, fmt.Errorf("failed to update workflow template: %w", err)
	}

	return &pb.WorkflowTemplateResponse{
		Id: req.Id,
	}, nil
}

func (s *Service) DeleteWorkflowTemplate(ctx context.Context, req *pb.DeleteWorkflowTemplateRequest) (*pb.WorkflowTemplateResponse, error) {
	s.logger.Info("DeleteWorkflowTemplate request received", zap.String("template_id", req.Id))

	err := s.db.DeleteWorkflowTemplate(req.Id)
	if err != nil {
		s.logger.Error("Failed to delete workflow template", zap.Error(err))
		return nil, fmt.Errorf("failed to delete workflow template: %w", err)
	}

	return &pb.WorkflowTemplateResponse{
		Id: req.Id,
	}, nil
}

func (s *Service) ExecuteWorkflowTemplate(ctx context.Context, req *pb.ExecuteWorkflowTemplateRequest) (*pb.WorkflowResponse, error) {
	s.logger.Info("ExecuteWorkflowTemplate request received", zap.Any("request", req))

	template, err := s.db.GetWorkflowTemplate(req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow template: %w", err)
	}

	workflowReq := template.Config
	workflowReq.SHA = req.Sha

	protoReq := s.convertWorkflowRequestToProto(workflowReq)

	workflowResp, err := s.CreateWorkflow(ctx, protoReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow from template: %w", err)
	}

	update := db.WorkflowUpdate{
		TemplateID: &req.Id,
		RunName:    &req.RunName,
	}
	err = s.db.UpdateWorkflow(workflowResp.WorkflowId, update)
	if err != nil {
		s.logger.Warn("Failed to update workflow with template information", zap.Error(err))
	}

	return workflowResp, nil
}

func (s *Service) GetTemplateRunHistory(ctx context.Context, req *pb.GetTemplateRunHistoryRequest) (*pb.TemplateRunHistoryResponse, error) {
	s.logger.Info("GetTemplateRunHistory request received", zap.String("template_id", req.Id))

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50
	}

	workflows, err := s.db.ListTemplateWorkflows(req.Id, limit, int(req.Offset))
	if err != nil {
		s.logger.Error("Failed to list template workflows", zap.Error(err))
		return nil, fmt.Errorf("failed to list template workflows: %w", err)
	}

	protoRuns := make([]*pb.TemplateRun, len(workflows))
	for i, workflow := range workflows {
		protoRuns[i] = &pb.TemplateRun{
			RunId:           workflow.RunName,
			WorkflowId:      workflow.WorkflowID,
			TemplateId:      workflow.TemplateID,
			Sha:             workflow.Config.SHA,
			RunName:         workflow.RunName,
			Status:          db.WorkflowStatusToString(workflow.Status),
			StartedAt:       workflow.CreatedAt.Format(time.RFC3339),
			MonitoringLinks: workflow.MonitoringLinks,
			Provider:        workflow.Provider,
		}
		if isWorkflowTerminal(workflow.Status) {
			protoRuns[i].CompletedAt = workflow.UpdatedAt.Format(time.RFC3339)
		}
	}

	return &pb.TemplateRunHistoryResponse{
		Runs:  protoRuns,
		Count: int32(len(protoRuns)),
	}, nil
}

func (s *Service) convertProtoToWorkflowRequest(req *pb.CreateWorkflowRequest) messages.TestnetWorkflowRequest {
	workflowReq := messages.TestnetWorkflowRequest{
		Repo:                   req.Repo,
		SHA:                    req.Sha,
		CosmosSdkSha:           req.CosmosSdkSha,
		CometBFTSha:            req.CometbftSha,
		IsEvmChain:             req.IsEvmChain,
		RunnerType:             messages.RunnerType(req.RunnerType),
		LongRunningTestnet:     req.LongRunningTestnet,
		LaunchLoadBalancer:     req.LaunchLoadBalancer,
		TestnetDuration:        req.TestnetDuration,
		NumWallets:             int(req.NumWallets),
		ProviderSpecificConfig: req.ProviderConfig,
	}

	if req.ChainConfig != nil {
		chainConfig := types.ChainsConfig{
			Name:                  req.ChainConfig.Name,
			Image:                 req.ChainConfig.Image,
			NumOfNodes:            req.ChainConfig.NumOfNodes,
			NumOfValidators:       req.ChainConfig.NumOfValidators,
			SetSeedNode:           req.ChainConfig.SetSeedNode,
			SetPersistentPeers:    req.ChainConfig.SetPersistentPeers,
			CustomAppConfig:       s.parseJSONConfig(req.ChainConfig.CustomAppConfig, "custom_app_config"),
			CustomConsensusConfig: s.parseJSONConfig(req.ChainConfig.CustomConsensusConfig, "custom_consensus_config"),
			CustomClientConfig:    s.parseJSONConfig(req.ChainConfig.CustomClientConfig, "custom_client_config"),
		}

		if req.ChainConfig.RegionConfigs != nil {
			for _, rc := range req.ChainConfig.RegionConfigs {
				chainConfig.RegionConfigs = append(chainConfig.RegionConfigs, petritypes.RegionConfig{
					Name:          rc.Name,
					NumNodes:      int(rc.NumOfNodes),
					NumValidators: int(rc.NumOfValidators),
				})
			}
		}

		if req.ChainConfig.GenesisModifications != nil {
			for _, gm := range req.ChainConfig.GenesisModifications {
				chainConfig.GenesisModifications = append(
					chainConfig.GenesisModifications,
					chain.GenesisKV{
						Key:   gm.Key,
						Value: gm.Value,
					},
				)
			}
		}

		workflowReq.ChainConfig = chainConfig
	}

	if len(req.EncodedLoadTestSpec) != 0 {
		loadTestSpec, err := decodeLoadTestSpec(req.EncodedLoadTestSpec)
		if err != nil {
			s.logger.Warn("Failed to decode load test spec in template", zap.Error(err))
		} else {
			switch loadTestSpec.Kind {
			case "eth":
				workflowReq.EthereumLoadTestSpec = &loadTestSpec
			case "cosmos":
				workflowReq.CosmosLoadTestSpec = &loadTestSpec
			}
		}
	}

	return workflowReq
}

func (s *Service) convertWorkflowRequestToProto(req messages.TestnetWorkflowRequest) *pb.CreateWorkflowRequest {
	protoReq := &pb.CreateWorkflowRequest{
		Repo:               req.Repo,
		Sha:                req.SHA,
		CosmosSdkSha:       req.CosmosSdkSha,
		CometbftSha:        req.CometBFTSha,
		IsEvmChain:         req.IsEvmChain,
		RunnerType:         string(req.RunnerType),
		LongRunningTestnet: req.LongRunningTestnet,
		LaunchLoadBalancer: req.LaunchLoadBalancer,
		TestnetDuration:    req.TestnetDuration,
		NumWallets:         int32(req.NumWallets),
		ProviderConfig:     req.ProviderSpecificConfig,
	}

	chainConfig := &pb.ChainConfig{
		Name:                  req.ChainConfig.Name,
		Image:                 req.ChainConfig.Image,
		NumOfNodes:            req.ChainConfig.NumOfNodes,
		NumOfValidators:       req.ChainConfig.NumOfValidators,
		SetSeedNode:           req.ChainConfig.SetSeedNode,
		SetPersistentPeers:    req.ChainConfig.SetPersistentPeers,
		CustomAppConfig:       marshalJSONConfig(req.ChainConfig.CustomAppConfig),
		CustomConsensusConfig: marshalJSONConfig(req.ChainConfig.CustomConsensusConfig),
		CustomClientConfig:    marshalJSONConfig(req.ChainConfig.CustomClientConfig),
	}

	for _, rc := range req.ChainConfig.RegionConfigs {
		chainConfig.RegionConfigs = append(chainConfig.RegionConfigs, &pb.RegionConfig{
			Name:            rc.Name,
			NumOfNodes:      uint64(rc.NumNodes),
			NumOfValidators: uint64(rc.NumValidators),
		})
	}

	for _, gm := range req.ChainConfig.GenesisModifications {
		value := ""
		if gm.Value != nil {
			if str, ok := gm.Value.(string); ok {
				value = str
			} else {
				if bytes, err := json.Marshal(gm.Value); err == nil {
					value = string(bytes)
				}
			}
		}
		chainConfig.GenesisModifications = append(chainConfig.GenesisModifications, &pb.GenesisKV{
			Key:   gm.Key,
			Value: value,
		})
	}

	protoReq.ChainConfig = chainConfig

	if req.EthereumLoadTestSpec != nil {
		encodedSpec, err := encodeLoadTestSpec(*req.EthereumLoadTestSpec)
		if err == nil {
			protoReq.EncodedLoadTestSpec = encodedSpec
		}
	} else if req.CosmosLoadTestSpec != nil {
		encodedSpec, err := encodeLoadTestSpec(*req.CosmosLoadTestSpec)
		if err == nil {
			protoReq.EncodedLoadTestSpec = encodedSpec
		}
	}

	return protoReq
}

func (s *Service) convertTemplateToProto(template *db.WorkflowTemplate) *pb.WorkflowTemplate {
	protoTemplate := &pb.WorkflowTemplate{
		Id:             template.ID,
		Description:    template.Description,
		TemplateConfig: s.convertWorkflowRequestToProto(template.Config),
		CreatedAt:      template.CreatedAt.Format(time.RFC3339),
		CreatedBy:      template.CreatedBy,
	}
	return protoTemplate
}

func (s *Service) getTemplateRunCount(templateID string) (int, error) {
	workflows, err := s.db.ListTemplateWorkflows(templateID, 1000, 0)
	if err != nil {
		return 0, err
	}
	return len(workflows), nil
}

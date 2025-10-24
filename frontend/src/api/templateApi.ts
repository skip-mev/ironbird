import { grpcWorkflowApi } from './grpcClient';
import type { 
  WorkflowTemplate,
  WorkflowTemplateSummary,
  CreateWorkflowTemplateRequest,
  WorkflowTemplateResponse,
  ExecuteWorkflowTemplateRequest,
  TemplateRunHistoryResponse,
  TestnetWorkflowRequest
} from '../types/workflow';
import {
  CreateWorkflowTemplateRequest as ProtoCreateRequest,
  ExecuteWorkflowTemplateRequest as ProtoExecuteRequest,
} from '../gen/proto/ironbird_pb.js';
import { convertToGrpcCreateWorkflowRequest } from './workflowApi';

// Helper function to safely parse JSON strings
const safeJSONParse = (jsonString: string): any => {
  try {
    return JSON.parse(jsonString);
  } catch (error) {
    console.warn('Failed to parse JSON config:', jsonString, error);
    return undefined;
  }
};

// Helper function to convert frontend template request to protobuf
const convertToProtoTemplateRequest = (request: CreateWorkflowTemplateRequest): ProtoCreateRequest => {
  const protoRequest = new ProtoCreateRequest({
    id: request.templateId,
    description: request.description,
    templateConfig: convertToGrpcCreateWorkflowRequest(request.templateConfig),
  });

  return protoRequest;
};

// Helper function to convert protobuf template to frontend type
const convertFromProtoTemplate = (protoTemplate: any): WorkflowTemplate => {
  return {
    templateId: protoTemplate.id,
    description: protoTemplate.description,
    templateConfig: convertFromProtoWorkflowRequest(protoTemplate.templateConfig),
    createdAt: protoTemplate.createdAt,
    createdBy: protoTemplate.createdBy,
  };
};

// Helper function to convert protobuf workflow request to frontend type  
const convertFromProtoWorkflowRequest = (protoConfig: any): TestnetWorkflowRequest => {
  // Convert genesis modifications from protobuf format
  const genesisModifications = (protoConfig.chainConfig?.genesisModifications || []).map((gm: any) => ({
    key: gm.key || '',
    value: gm.value || '',
  }));

  // Convert region configs from protobuf format
  const regionConfigs = (protoConfig.chainConfig?.regionConfigs || []).map((rc: any) => ({
    name: rc.name || '',
    numOfNodes: Number(rc.numOfNodes) || 0,
    numOfValidators: Number(rc.numOfValidators) || 0,
  }));

  return {
    Repo: protoConfig.repo || '',
    SHA: protoConfig.sha || '',
    IsEvmChain: protoConfig.isEvmChain || false,
    ChainConfig: {
      Name: protoConfig.chainConfig?.name || '',
      Image: protoConfig.chainConfig?.image || '',
      Version: protoConfig.chainConfig?.version || '',
      NumOfNodes: Number(protoConfig.chainConfig?.numOfNodes) || 0,
      NumOfValidators: Number(protoConfig.chainConfig?.numOfValidators) || 0,
      GenesisModifications: genesisModifications,
      RegionConfigs: regionConfigs,
      AppConfig: protoConfig.chainConfig?.customAppConfig ? 
        safeJSONParse(protoConfig.chainConfig.customAppConfig) : undefined,
      ConsensusConfig: protoConfig.chainConfig?.customConsensusConfig ? 
        safeJSONParse(protoConfig.chainConfig.customConsensusConfig) : undefined,
      ClientConfig: protoConfig.chainConfig?.customClientConfig ? 
        safeJSONParse(protoConfig.chainConfig.customClientConfig) : undefined,
      SetSeedNode: protoConfig.chainConfig?.setSeedNode || false,
      SetPersistentPeers: protoConfig.chainConfig?.setPersistentPeers || false,
    },
    RunnerType: protoConfig.runnerType || '',
    EncodedLoadTestSpec: protoConfig.encodedLoadTestSpec || '',
    LongRunningTestnet: protoConfig.longRunningTestnet || false,
    LaunchLoadBalancer: protoConfig.launchLoadBalancer || false,
    TestnetDuration: protoConfig.testnetDuration || '',
    NumWallets: Number(protoConfig.numWallets) || 2500,
    CatalystVersion: protoConfig.catalystVersion || '',
  };
};

export const templateApi = {
  createTemplate: async (request: CreateWorkflowTemplateRequest): Promise<WorkflowTemplateResponse> => {
    const protoRequest = convertToProtoTemplateRequest(request);
    const response = await grpcWorkflowApi.createWorkflowTemplate(protoRequest);
    return {
      templateId: response.id,
    };
  },

  getTemplate: async (templateId: string): Promise<WorkflowTemplate> => {
    const response = await grpcWorkflowApi.getWorkflowTemplate(templateId);
    return convertFromProtoTemplate(response);
  },

  listTemplates: async (limit?: number, offset?: number): Promise<{templates: WorkflowTemplateSummary[], returnedCount: number}> => {
    const response = await grpcWorkflowApi.listWorkflowTemplates(limit, offset);
    
    const templates: WorkflowTemplateSummary[] = (response.templates || []).map((template: any) => ({
      templateId: template.id,
      description: template.description,
      createdAt: template.createdAt,
      runCount: template.runCount || 0,
    }));

    return {
      templates,
      returnedCount: response.returnedCount || 0,
    };
  },

  deleteTemplate: async (templateId: string): Promise<WorkflowTemplateResponse> => {
    const response = await grpcWorkflowApi.deleteWorkflowTemplate(templateId);
    return {
      templateId: response.id,
    };
  },

  executeTemplate: async (request: ExecuteWorkflowTemplateRequest): Promise<{workflowId: string}> => {
    const protoRequest = new ProtoExecuteRequest({
      id: request.templateId,
      sha: request.sha,
      runName: request.runName || '',
    });
    
    const response = await grpcWorkflowApi.executeWorkflowTemplate(protoRequest);
    return {
      workflowId: response.workflowId,
    };
  },

  getTemplateRunHistory: async (templateId: string, limit?: number, offset?: number): Promise<TemplateRunHistoryResponse> => {
    const response = await grpcWorkflowApi.getTemplateRunHistory(templateId, limit, offset);
    
    const runs = (response.runs || []).map((run: any) => ({
      runId: run.runId,
      workflowId: run.workflowId,
      templateId: run.templateId,
      sha: run.sha,
      runName: run.runName,
      status: run.status,
      startedAt: run.startedAt,
      completedAt: run.completedAt,
      monitoringLinks: run.monitoringLinks || {},
      provider: run.provider || '',
    }));

    return {
      runs,
      returnedCount: response.returnedCount || 0,
    };
  },
};
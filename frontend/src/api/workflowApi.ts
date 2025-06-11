import { grpcWorkflowApi } from './grpcClient';
import type { TestnetWorkflowRequest, WorkflowResponse, WorkflowStatus, LoadTestSpec } from '../types/workflow';
import { 
  CreateWorkflowRequest, 
  ChainConfig, 
  LoadTestSpec as GrpcLoadTestSpec,
  LoadTestMsg,
  GenesisKV
} from '../gen/proto/ironbird_pb.js';
import { protoInt64 } from "@bufbuild/protobuf";

// Helper function to convert frontend LoadTestSpec to gRPC LoadTestSpec
const convertToGrpcLoadTestSpec = (spec: LoadTestSpec): GrpcLoadTestSpec => {
  const grpcSpec = new GrpcLoadTestSpec({
    name: spec.name,
    description: spec.description,
    evm: spec.evm || false,
    chainId: spec.chain_id,
    numOfBlocks: spec.NumOfBlocks,
    numOfTxs: spec.NumOfTxs || 0,
    nodesAddresses: [],
    mnemonics: [],
    gasDenom: "",
    bech32Prefix: "",
    msgs: [],
    unorderedTxs: spec.unordered_txs,
    txTimeout: protoInt64.parse(spec.tx_timeout || "30")
  });
  
  // Add messages
  if (spec.msgs && spec.msgs.length > 0) {
    grpcSpec.msgs = spec.msgs.map(msg => new LoadTestMsg({
      weight: msg.weight,
      type: msg.type,
      numMsgs: msg.NumMsgs || 1,
      containedType: msg.ContainedType || "",
      numOfRecipients: msg.NumOfRecipients || 1
    }));
  }
  
  return grpcSpec;
};

// Helper function to convert frontend TestnetWorkflowRequest to gRPC CreateWorkflowRequest
const convertToGrpcCreateWorkflowRequest = (request: TestnetWorkflowRequest): CreateWorkflowRequest => {
  // Create the chain config with proper constructor
  const chainConfig = new ChainConfig({
    name: request.ChainConfig.Name,
    numOfNodes: protoInt64.zero,
    numOfValidators: protoInt64.zero,
    image: request.ChainConfig.Image,
    genesisModifications: []
  });
  
  // Add genesis modifications if available
  if (request.ChainConfig.GenesisModifications && request.ChainConfig.GenesisModifications.length > 0) {
    chainConfig.genesisModifications = request.ChainConfig.GenesisModifications.map(gm => {
      // Convert value to string if it's not already a string
      let valueStr = typeof gm.value === 'string' ? gm.value : JSON.stringify(gm.value);
      return new GenesisKV({
        key: gm.key,
        value: valueStr
      });
    });
  }
  
  // Create the request with proper constructor
  const grpcRequest = new CreateWorkflowRequest({
    repo: request.Repo,
    sha: request.SHA,
    evm: request.evm,
    chainConfig: chainConfig,
    runnerType: request.RunnerType,
    longRunningTestnet: request.LongRunningTestnet,
    testnetDuration: protoInt64.zero,
    numWallets: request.NumWallets
  });
  
  // Convert number values to bigint
  if (request.ChainConfig.NumOfNodes) {
    grpcRequest.chainConfig!.numOfNodes = protoInt64.parse(request.ChainConfig.NumOfNodes.toString());
  }
  
  if (request.ChainConfig.NumOfValidators) {
    grpcRequest.chainConfig!.numOfValidators = protoInt64.parse(request.ChainConfig.NumOfValidators.toString());
  }
  
  if (request.TestnetDuration) {
    grpcRequest.testnetDuration = protoInt64.parse(request.TestnetDuration.toString());
  }

  if (request.LoadTestSpec) {
    const grpcLoadTestSpec = convertToGrpcLoadTestSpec(request.LoadTestSpec);
    grpcLoadTestSpec.evm = request.evm;
    grpcRequest.loadTestSpec = grpcLoadTestSpec;
  }

  return grpcRequest;
};

// Helper function to convert gRPC response to frontend WorkflowResponse
const convertFromGrpcWorkflowResponse = (response: any): WorkflowResponse => {
  return {
    WorkflowID: response.workflowId,
    Status: response.status
  };
};

// Helper function to convert gRPC Workflow to frontend WorkflowStatus
const convertFromGrpcWorkflow = (workflow: any): WorkflowStatus => {
  // Create a config object from the workflow.config field if it exists
  let config: TestnetWorkflowRequest | undefined = undefined;
  
  if (workflow.config) {
    config = {
      Repo: workflow.config.repo,
      SHA: workflow.config.sha,
      evm: workflow.config.evm,
      RunnerType: workflow.config.runnerType,
      LongRunningTestnet: workflow.config.longRunningTestnet,
      TestnetDuration: workflow.config.testnetDuration,
      NumWallets: workflow.config.numWallets,
      ChainConfig: {
        Name: workflow.config.chainConfig?.name || '',
        Image: workflow.config.chainConfig?.image || '',
        NumOfNodes: Number(workflow.config.chainConfig?.numOfNodes) || 0,
        NumOfValidators: Number(workflow.config.chainConfig?.numOfValidators) || 0,
        GenesisModifications: (workflow.config.chainConfig?.genesisModifications || []).map((gm: any) => {
          // Try to parse the value as JSON if it's a string
          let value = gm.value;
          try {
            if (typeof gm.value === 'string') {
              value = JSON.parse(gm.value);
            }
          } catch (e) {
            // If parsing fails, use the original string value
            console.warn('Failed to parse genesis modification value as JSON', e);
          }
          
          return {
            key: gm.key,
            value: value
          };
        })
      },
      LoadTestSpec: workflow.loadTestSpec ? {
        name: workflow.loadTestSpec.name,
        description: workflow.loadTestSpec.description,
        chain_id: workflow.loadTestSpec.chainId,
        NumOfBlocks: workflow.loadTestSpec.numOfBlocks,
        NumOfTxs: workflow.loadTestSpec.numOfTxs,
        msgs: (workflow.loadTestSpec.msgs || []).map((msg: any) => ({
          type: msg.type,
          weight: msg.weight,
          NumMsgs: msg.numMsgs,
          ContainedType: msg.containedType,
          NumOfRecipients: msg.numOfRecipients
        })),
        unordered_txs: workflow.loadTestSpec.unorderedTxs,
        tx_timeout: workflow.loadTestSpec.txTimeout?.toString() || '',
        evm: workflow.loadTestSpec.evm
      } : undefined
    };
  }
  
  return {
    WorkflowID: workflow.workflowId,
    Status: workflow.status,
    Nodes: (workflow.nodes || []).map((node: any) => ({
      Name: node.name,
      RPC: node.rpc,
      LCD: node.lcd,
      GRPC: node.grpc,
      Metrics: ""
    })),
    Validators: (workflow.validators || []).map((validator: any) => ({
      Name: validator.name,
      RPC: validator.rpc,
      LCD: validator.lcd,
      GRPC: validator.grpc,
      Metrics: ""
    })),
    LoadBalancers: (workflow.loadBalancers || []).map((lb: any) => ({
      Name: lb.name,
      RPC: lb.rpc,
      LCD: lb.lcd,
      GRPC: lb.grpc,
      Metrics: ""
    })),
    Monitoring: workflow.monitoring || {},
    wallets: workflow.wallets ? {
      faucetAddress: workflow.wallets.faucetAddress || '',
      faucetMnemonic: workflow.wallets.faucetMnemonic || '',
      userAddresses: workflow.wallets.userAddresses || [],
      userMnemonics: workflow.wallets.userMnemonics || []
    } : undefined,
    config: config,
    loadTestSpec: workflow.loadTestSpec
  };
};

export const workflowApi = {
  createWorkflow: async (request: TestnetWorkflowRequest): Promise<WorkflowResponse> => {
    const grpcRequest = convertToGrpcCreateWorkflowRequest(request);
    const response = await grpcWorkflowApi.createWorkflow(grpcRequest);
    return convertFromGrpcWorkflowResponse(response);
  },

  listWorkflows: async (): Promise<{Workflows: Array<{WorkflowID: string; Status: string; StartTime: string; Repo?: string; SHA?: string}>; Count: number}> => {
    const response = await grpcWorkflowApi.listWorkflows();
    return {
      Workflows: (response.workflows || []).map((workflow: any) => ({
        WorkflowID: workflow.workflowId,
        Status: workflow.status,
        StartTime: workflow.startTime,
        Repo: workflow.repo,
        SHA: workflow.sha
      })),
      Count: response.count || 0
    };
  },

  // updateWorkflow: async (workflowId: string, request: TestnetWorkflowRequest): Promise<WorkflowResponse> => {
  //   // gRPC doesn't support workflow updates yet
  //   throw new Error("Workflow updates are not supported in the gRPC API");
  // },

  getWorkflow: async (workflowId: string): Promise<WorkflowStatus> => {
    const response = await grpcWorkflowApi.getWorkflow(workflowId);
    return convertFromGrpcWorkflow(response);
  },

  runLoadTest: async (workflowId: string, spec: LoadTestSpec): Promise<WorkflowResponse> => {
    // Make sure we have a copy of the spec to avoid modifying the original
    const specCopy = { ...spec };
    
    // If evm is not set in the spec, default to false
    if (specCopy.evm === undefined) {
      specCopy.evm = false;
    }
    
    const grpcSpec = convertToGrpcLoadTestSpec(specCopy);
    const response = await grpcWorkflowApi.runLoadTest(workflowId, grpcSpec);
    return convertFromGrpcWorkflowResponse(response);
  },

  cancelWorkflow: async (workflowId: string): Promise<WorkflowResponse> => {
    const response = await grpcWorkflowApi.cancelWorkflow(workflowId);
    return convertFromGrpcWorkflowResponse(response);
  },

  sendShutdownSignal: async (workflowId: string): Promise<WorkflowResponse> => {
    const response = await grpcWorkflowApi.sendShutdownSignal(workflowId);
    return convertFromGrpcWorkflowResponse(response);
  },
};

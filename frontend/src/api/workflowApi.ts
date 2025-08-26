import { grpcWorkflowApi } from './grpcClient';
import type { TestnetWorkflowRequest, WorkflowResponse, WorkflowStatus, LoadTestSpec } from '../types/workflow';
import { 
  CreateWorkflowRequest, 
  ChainConfig, 
  GenesisKV,
  RegionConfig
} from '../gen/proto/ironbird_pb.js';
import { protoInt64 } from "@bufbuild/protobuf";

// Helper function to convert frontend LoadTestSpec to YAML string
const convertLoadTestSpecToYaml = (spec: LoadTestSpec): string => {
  // Convert the frontend LoadTestSpec to YAML format for the server
  const yamlSpec = {
    name: spec.name,
    description: spec.description,
    kind: spec.kind,
    chain_id: spec.chain_id,
    ...(spec.NumOfBlocks && { num_of_blocks: spec.NumOfBlocks }),
    ...(spec.NumOfTxs && { num_of_txs: spec.NumOfTxs }),
    unordered_txs: spec.unordered_txs,
    ...(spec.tx_timeout && { tx_timeout: spec.tx_timeout }),
    msgs: spec.msgs.map(msg => ({
      type: msg.type,
      weight: msg.weight || 0, // Ethereum uses weight: 0
      num_msgs: msg.num_msgs || msg.NumMsgs || 1,
      contained_type: msg.ContainedType,
      num_of_recipients: msg.NumOfRecipients,
      // Ethereum-specific fields
      num_of_iterations: msg.NumOfIterations,
      calldata_size: msg.CalldataSize
    })),
    // Add conditional fields based on kind
    ...(spec.kind === 'eth' && {
      ...(spec.send_interval && { send_interval: spec.send_interval }),
      ...(spec.num_batches && { num_batches: spec.num_batches })
    }),
    ...(spec.kind === 'cosmos' && {
      ...(spec.gas_denom && { gas_denom: spec.gas_denom }),
      ...(spec.bech32_prefix && { bech32_prefix: spec.bech32_prefix })
    }),
    chain_config: {} // This will be populated by the server
  };
  
  // Convert to YAML string (simplified - in practice you'd use a YAML library)
  return JSON.stringify(yamlSpec, null, 2);
};

// Helper function to convert frontend TestnetWorkflowRequest to gRPC CreateWorkflowRequest
const convertToGrpcCreateWorkflowRequest = (request: TestnetWorkflowRequest): CreateWorkflowRequest => {
  // Create the chain config with proper constructor
  const chainConfig = new ChainConfig({
    name: request.ChainConfig.Name,
    numOfNodes: request.ChainConfig.NumOfNodes !== undefined ? protoInt64.parse(request.ChainConfig.NumOfNodes.toString()) : protoInt64.zero,
    numOfValidators: request.ChainConfig.NumOfValidators !== undefined ? protoInt64.parse(request.ChainConfig.NumOfValidators.toString()) : protoInt64.zero,
    image: request.ChainConfig.Image,
    genesisModifications: [],
    setSeedNode: request.ChainConfig.SetSeedNode || false,
    setPersistentPeers: request.ChainConfig.SetPersistentPeers || false,
    customAppConfig: request.ChainConfig.AppConfig ? JSON.stringify(request.ChainConfig.AppConfig) : "",
    customConsensusConfig: request.ChainConfig.ConsensusConfig ? JSON.stringify(request.ChainConfig.ConsensusConfig) : "",
    customClientConfig: request.ChainConfig.ClientConfig ? JSON.stringify(request.ChainConfig.ClientConfig) : ""
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

  // Add RegionConfigs if available
  if (request.ChainConfig.RegionConfigs && request.ChainConfig.RegionConfigs.length > 0) {
    chainConfig.regionConfigs = request.ChainConfig.RegionConfigs.map(rc => new RegionConfig({
      name: rc.name,
      numOfNodes: protoInt64.parse(rc.numOfNodes.toString()),
      numOfValidators: protoInt64.parse(rc.numOfValidators.toString())
    }));
  }
  
  // Create the request with proper constructor
  const grpcRequest = new CreateWorkflowRequest({
    repo: request.Repo,
    sha: request.SHA,
    isEvmChain: request.IsEvmChain,
    chainConfig: chainConfig,
    runnerType: request.RunnerType,
    longRunningTestnet: request.LongRunningTestnet,
    launchLoadBalancer: request.LaunchLoadBalancer,
    testnetDuration: request.TestnetDuration || '',
    numWallets: request.NumWallets
  });
  
  // Convert number values to bigint
  if (request.ChainConfig.NumOfNodes) {
    grpcRequest.chainConfig!.numOfNodes = protoInt64.parse(request.ChainConfig.NumOfNodes.toString());
  }
  
  if (request.ChainConfig.NumOfValidators) {
    grpcRequest.chainConfig!.numOfValidators = protoInt64.parse(request.ChainConfig.NumOfValidators.toString());
  }
  
  if (request.LoadTestSpec) {
    grpcRequest.encodedLoadTestSpec = convertLoadTestSpecToYaml(request.LoadTestSpec);
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
      IsEvmChain: workflow.config.IsEvmChain,
      RunnerType: workflow.config.runnerType,
      LongRunningTestnet: workflow.config.longRunningTestnet,
      LaunchLoadBalancer: workflow.config.launchLoadBalancer,
      TestnetDuration: workflow.config.testnetDuration,
      NumWallets: workflow.config.numWallets,
      ChainConfig: {
        Name: workflow.config.chainConfig?.name || '',
        Image: workflow.config.chainConfig?.image || '',
        NumOfNodes: workflow.config.chainConfig?.numOfNodes ? Number(workflow.config.chainConfig.numOfNodes) : 0,
        NumOfValidators: workflow.config.chainConfig?.numOfValidators ? Number(workflow.config.chainConfig.numOfValidators) : 0,
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
        }),
        RegionConfigs: (workflow.config.chainConfig?.regionConfigs || []).map((rc: any) => ({
          name: rc.name,
          numOfNodes: Number(rc.numOfNodes) || 0,
          numOfValidators: Number(rc.numOfValidators) || 0
        })),
        SetSeedNode: workflow.config.chainConfig?.setSeedNode,
        SetPersistentPeers: workflow.config.chainConfig?.setPersistentPeers,
        // Parse config JSON strings back to objects
        AppConfig: (() => {
          try {
            return workflow.config.chainConfig?.customAppConfig ? JSON.parse(workflow.config.chainConfig.customAppConfig) : undefined;
          } catch (e) {
            console.warn('Failed to parse app config JSON', e);
            return undefined;
          }
        })(),
        ConsensusConfig: (() => {
          try {
            return workflow.config.chainConfig?.customConsensusConfig ? JSON.parse(workflow.config.chainConfig.customConsensusConfig) : undefined;
          } catch (e) {
            console.warn('Failed to parse consensus config JSON', e);
            return undefined;
          }
        })(),
        ClientConfig: (() => {
          try {
            return workflow.config.chainConfig?.customClientConfig ? JSON.parse(workflow.config.chainConfig.customClientConfig) : undefined;
          } catch (e) {
            console.warn('Failed to parse client config JSON', e);
            return undefined;
          }
        })()
      },
      LoadTestSpec: workflow.loadTestSpec ? JSON.parse(workflow.loadTestSpec) : undefined
    };
  }
  
  return {
    WorkflowID: workflow.workflowId,
    Status: workflow.status,
    Provider: workflow.provider || '',
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

  listWorkflows: async (): Promise<{Workflows: Array<{WorkflowID: string; Status: string; StartTime: string; Repo?: string; SHA?: string; Provider?: string}>; Count: number}> => {
    const response = await grpcWorkflowApi.listWorkflows();
    return {
      Workflows: (response.workflows || []).map((workflow: any) => ({
        WorkflowID: workflow.workflowId,
        Status: workflow.status,
        StartTime: workflow.startTime,
        Repo: workflow.repo,
        SHA: workflow.sha,
        Provider: workflow.provider || ''
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
    const yamlSpec = convertLoadTestSpecToYaml(spec);
    const response = await grpcWorkflowApi.runLoadTest(workflowId, yamlSpec);
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

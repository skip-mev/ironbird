export interface GenesisModification {
  key: string;
  value: any;
}

export interface RegionConfig {
  name: string;
  numOfNodes: number;
  numOfValidators: number;
}

export interface ChainConfig {
  Name: string;
  Image: string;
  Version?: string;
  NumOfNodes?: number;
  NumOfValidators?: number;
  GenesisModifications: GenesisModification[];
  RegionConfigs?: RegionConfig[];
  AppConfig?: Record<string, any>;
  ConsensusConfig?: Record<string, any>;
  ClientConfig?: Record<string, any>;
  ProviderConfig?: Record<string, any>;
  SetSeedNode?: boolean;
  SetPersistentPeers?: boolean;
}

// Define message types as string literals
export const MsgType = {
  // Cosmos message types
  MsgSend: "MsgSend",
  MsgMultiSend: "MsgMultiSend",
  MsgArr: "MsgArr",
  // Ethereum message types
  MsgCreateContract: "MsgCreateContract",
  MsgWriteTo: "MsgWriteTo",
  MsgCrossContractCall: "MsgCrossContractCall",
  MsgCallDataBlast: "MsgCallDataBlast",
  MsgNativeTransferERC20: "MsgNativeTransferERC20"
} as const;

// Type for message types
export type MsgType = typeof MsgType[keyof typeof MsgType];

export interface Message {
  type: MsgType;
  weight?: number; // Used by Cosmos (for weighted distribution)
  NumMsgs?: number; // Number of messages in MsgArr (Cosmos) or transactions (Ethereum)
  ContainedType?: MsgType; // Type of contained messages for MsgArr (Cosmos)
  NumOfRecipients?: number; // Number of recipients for MsgMultiSend (Cosmos)
  // Ethereum-specific fields
  num_msgs?: number; // Number of transactions to create (Ethereum)
  NumOfIterations?: number; // Number of iterations for contract operations (Ethereum)
  CalldataSize?: number; // Size of calldata for large payload tests (Ethereum)
}

export interface LoadTestSpec {
  name: string;
  description: string;
  chain_id: string;
  kind: 'cosmos' | 'eth'; // Load test type
  NumOfBlocks: number;
  NumOfTxs: number;
  msgs: Message[];
  unordered_txs: boolean;
  tx_timeout: string;
  // Ethereum-specific fields
  send_interval?: string;
  num_batches?: number;
  // Cosmos-specific fields
  gas_denom?: string;
  bech32_prefix?: string;
}

export interface TestnetWorkflowRequest {
  Repo: string;
  SHA: string;
  IsEvmChain: boolean;
  ChainConfig: ChainConfig;
  RunnerType: string;
  CosmosSdkSha?: string; // Optional: cosmos-sdk version/SHA for EVM builds
  CometBFTSha?: string; // Optional: cometbft version/SHA for EVM builds
  LoadTestSpec?: LoadTestSpec;
  EthereumLoadTestSpec?: LoadTestSpec;
  CosmosLoadTestSpec?: LoadTestSpec;
  EncodedLoadTestSpec?: string; // YAML-encoded load test spec
  LongRunningTestnet: boolean;
  LaunchLoadBalancer: boolean;
  TestnetDuration: string;
  NumWallets: number;
  CatalystVersion?: string;
}

export interface Node {
  Name: string;
  RPC: string;
  LCD: string;
  Metrics: string;
  GRPC: string;
  EVMRPC?: string;
  EVMWS?: string;
}

export interface WalletInfo {
  faucetAddress: string;
  faucetMnemonic: string;
  userAddresses: string[];
  userMnemonics: string[];
}

export interface WorkflowStatus {
  WorkflowID: string;
  Status: string;
  StartTime?: string;
  EndTime?: string;
  Nodes: Node[];
  Validators: Node[];
  LoadBalancers: Node[];
  Monitoring: Record<string, string>;
  wallets?: WalletInfo;
  config?: TestnetWorkflowRequest;
  loadTestSpec?: any;
  Provider?: string;
}

export interface WorkflowResponse {
  WorkflowID: string;
  Status: string;
  Data?: Record<string, any>;
}

// Template-related types
export interface WorkflowTemplate {
  templateId: string;
  description: string;
  templateConfig: TestnetWorkflowRequest;
  createdAt: string;
  createdBy: string;
}

export interface WorkflowTemplateSummary {
  templateId: string;
  description: string;
  createdAt: string;
  runCount: number;
}

export interface CreateWorkflowTemplateRequest {
  templateId: string;
  description: string;
  templateConfig: TestnetWorkflowRequest;
}

export interface WorkflowTemplateResponse {
  templateId: string;
}

export interface ExecuteWorkflowTemplateRequest {
  templateId: string;
  sha: string;
  runName?: string;
}

export interface TemplateRun {
  runId: string;
  workflowId: string;
  templateId: string;
  sha: string;
  runName?: string;
  status: string;
  startedAt: string;
  completedAt?: string;
  monitoringLinks: Record<string, string>;
  provider: string;
}

export interface TemplateRunHistoryResponse {
  runs: TemplateRun[];
  count: number;
}

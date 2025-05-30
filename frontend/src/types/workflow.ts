export interface GenesisModification {
  key: string;
  value: any;
}

export interface ChainConfig {
  Name: string;
  Image: string;
  GenesisModifications: GenesisModification[];
  NumOfNodes: number;
  NumOfValidators: number;
}

export enum MsgType {
  MsgSend = "MsgSend",
  MsgMultiSend = "MsgMultiSend",
  MsgArr = "MsgArr"
}

export interface Message {
  type: MsgType;
  weight: number;
  NumMsgs?: number;
  ContainedType?: MsgType;
  NumOfRecipients?: number;
}

export interface LoadTestSpec {
  name: string;
  description: string;
  chain_id: string;
  num_of_blocks: number;
  num_of_txs?: number;
  msgs: Message[];
  unordered_txs: boolean;
  tx_timeout: string;
}

export interface TestnetWorkflowRequest {
  Repo: string;
  SHA: string;
  GaiaEVM: boolean;
  ChainConfig: ChainConfig;
  RunnerType: string;
  LoadTestSpec?: LoadTestSpec;
  LongRunningTestnet: boolean;
  TestnetDuration: number;
}

export interface Node {
  Name: string;
  RPC: string;
  LCD: string;
  Metrics: string;
}

export interface WorkflowStatus {
  WorkflowID: string;
  Status: string;
  Nodes: Node[];
  Monitoring: Record<string, string>;
  Config?: TestnetWorkflowRequest;
  
  // Individual fields from the database
  repo?: string;
  sha?: string;
  chainName?: string;
  runnerType?: string;
  numOfNodes?: number;
  numOfValidators?: number;
  longRunningTestnet?: boolean;
  testnetDuration?: number;
  loadTestSpec?: any;
}

export interface WorkflowResponse {
  WorkflowID: string;
  Status: string;
  Data?: Record<string, any>;
}

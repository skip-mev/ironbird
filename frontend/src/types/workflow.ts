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
  AppConfig?: Record<string, any>;       
  ConsensusConfig?: Record<string, any>; 
  ClientConfig?: Record<string, any>;
  SetSeedNode?: boolean;
  SetPersistentPeers?: boolean;
}

// Define message types as string literals
export const MsgType = {
  MsgSend: "MsgSend",
  MsgMultiSend: "MsgMultiSend",
  MsgArr: "MsgArr"
} as const;

// Type for message types
export type MsgType = typeof MsgType[keyof typeof MsgType];

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
  NumOfBlocks: number;
  NumOfTxs: number;
  msgs: Message[];
  unordered_txs: boolean;
  tx_timeout: string;
  isEvmChain?: boolean;
}

export interface TestnetWorkflowRequest {
  Repo: string;
  SHA: string;
  IsEvmChain: boolean;
  ChainConfig: ChainConfig;
  RunnerType: string;
  LoadTestSpec?: LoadTestSpec;
  LongRunningTestnet: boolean;
  TestnetDuration: string;
  NumWallets: number;
}

export interface Node {
  Name: string;
  RPC: string;
  LCD: string;
  Metrics: string;
  GRPC: string;
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
  Nodes: Node[];
  Validators: Node[];
  LoadBalancers: Node[];
  Monitoring: Record<string, string>;
  wallets?: WalletInfo;
  config?: TestnetWorkflowRequest;
  loadTestSpec?: any;
}

export interface WorkflowResponse {
  WorkflowID: string;
  Status: string;
  Data?: Record<string, any>;
}

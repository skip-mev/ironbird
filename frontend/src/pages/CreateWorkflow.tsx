import { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  Box,
  Button,
  FormControl,
  FormLabel,
  Heading,
  Input,
  NumberInput,
  NumberInputField,
  Select,
  Stack,
  useToast,
  HStack,
  IconButton,
  Switch,
  Text,
  VStack,
  Flex,
  FormHelperText,
  Textarea,
} from '@chakra-ui/react';
import { useMutation } from '@tanstack/react-query';
import { workflowApi } from '../api/workflowApi';
import type { TestnetWorkflowRequest, LoadTestSpec } from '../types/workflow';
import { AddIcon, DeleteIcon } from '@chakra-ui/icons';
import LoadTestForm from '../components/LoadTestForm';
import ChainConfigsModal from '../components/ChainConfigsModal';
import GenesisModificationsModal from '../components/GenesisModificationsModal';

// Helper functions to generate default configs (based on backend GenerateDefault* functions)
const generateDefaultAppConfig = (gasPrices: string, isEvm: boolean = false, evmChainId: string = "4231") => {
  const config: any = {
    "minimum-gas-prices": gasPrices,
    "grpc": {
      "address": "0.0.0.0:9090"
    },
    "api": {
      "enable": true,
      "swagger": true,
      "address": "tcp://0.0.0.0:1317"
    },
    "telemetry": {
      "enabled": true,
      "prometheus-retention-time": 3600
    }
  };

  if (isEvm) {
    config["evm"] = {
      "tracer": "",
      "max-tx-gas-wanted": 0,
      "cache-preimage": false,
      "evm-chain-id": evmChainId
    };

    config["json-rpc"] = {
      "enable": true,
      "address": "127.0.0.1:8545",
      "ws-address": "127.0.0.1:8546",
      "api": "eth,net,web3",
      "gas-cap": 25000000,
      "allow-insecure-unlock": true,
      "evm-timeout": "5s",
      "txfee-cap": 1,
      "filter-cap": 200,
      "feehistory-cap": 100,
      "logs-cap": 10000,
      "block-range-cap": 10000,
      "http-timeout": "30s",
      "http-idle-timeout": "2m0s",
      "allow-unprotected-txs": false,
      "max-open-connections": 0,
      "enable-indexer": false,
      "metrics-address": "127.0.0.1:6065",
      "fix-revert-gas-refund-height": 0
    };
  }

  return config;
};

const generateDefaultConsensusConfig = () => ({
  "log_level": "info",
  "p2p": {
    "allow_duplicate_ip": true,
    "addr_book_strict": false
  },
  "consensus": {
    "timeout_commit": "2s",
    "timeout_propose": "2s"
  },
  "instrumentation": {
    "prometheus": true
  },
  "rpc": {
    "laddr": "tcp://0.0.0.0:26657",
    "allowed_origins": ["*"]
  }
});

const generateDefaultClientConfig = (chainId: string) => ({
  "chain-id": chainId,
  "keyring-backend": "test",
  "output": "text",
  "node": "http://localhost:26657",
  "broadcast-mode": "sync"
});

// Helper function to get gas prices based on repository
const getGasPricesForRepo = (repo: string): string => {
  switch (repo) {
    case 'gaia':
      return '0.00025uatom';
    default:
      return '0.0005stake';
  }
};

const CreateWorkflow = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const toast = useToast();
  // Default values to use as placeholders
  const defaultValues: TestnetWorkflowRequest = {
    Repo: 'cosmos-sdk',
    SHA: '',
    ChainConfig: {
      Name: 'test-chain',
      Image: '',
      GenesisModifications: [],
      NumOfNodes: 4,
      NumOfValidators: 3,
    },
    RunnerType: 'Docker',
    IsEvmChain: false,
    LoadTestSpec: undefined,
    LongRunningTestnet: false,
    TestnetDuration: 2, // 2 hours
    NumWallets: 2500, // Default number of wallets
  };

  // Initialize form data with empty values
  const [formData, setFormData] = useState<TestnetWorkflowRequest>({
    Repo: '',
    SHA: '',
    ChainConfig: {
      Name: '',
      Image: '',
      GenesisModifications: [],
      NumOfNodes: 0,
      NumOfValidators: 0,
    },
    RunnerType: '',
    IsEvmChain: false,
    LoadTestSpec: undefined,
    LongRunningTestnet: false,
    TestnetDuration: 0,
    NumWallets: 2500,
  });
  
  // Check for URL parameters (for cloning workflows)
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    
    console.log("URL parameters:", Object.fromEntries(params.entries()));
    
    // Only proceed if we have parameters
    if (params.toString()) {
      // Start with a fresh form data object based on default values
      // This ensures we have proper defaults for any fields not specified in the URL
      const newFormData: TestnetWorkflowRequest = {
        Repo: '',
        SHA: '',
        ChainConfig: {
          Name: '',
          Image: '',
          GenesisModifications: [],
          NumOfNodes: 0,
          NumOfValidators: 0,
        },
        RunnerType: '',
        IsEvmChain: false,
        LoadTestSpec: undefined,
        LongRunningTestnet: false,
        TestnetDuration: 0,
        NumWallets: 2500,
      };
      
      let hasChanges = false;
      
      if (params.get('repo')) {
        newFormData.Repo = params.get('repo')!;
        hasChanges = true;
        console.log("Setting repo:", newFormData.Repo);
      }
      
      if (params.get('sha')) {
        newFormData.SHA = params.get('sha')!;
        hasChanges = true;
        console.log("Setting SHA:", newFormData.SHA);
      }
      
      if (params.get('runnerType')) {
        newFormData.RunnerType = params.get('runnerType')!;
        hasChanges = true;
        console.log("Setting runnerType:", newFormData.RunnerType);
      }
      
      if (params.get('chainName')) {
        newFormData.ChainConfig.Name = params.get('chainName')!;
        hasChanges = true;
        console.log("Setting chainName:", newFormData.ChainConfig.Name);
      }
      
      if (params.get('image')) {
        newFormData.ChainConfig.Image = params.get('image')!;
        hasChanges = true;
        console.log("Setting image:", newFormData.ChainConfig.Image);
      }
      
      if (params.get('numOfNodes')) {
        const numNodes = parseInt(params.get('numOfNodes')!, 10);
        if (!isNaN(numNodes)) {
          newFormData.ChainConfig.NumOfNodes = numNodes;
          hasChanges = true;
          console.log("Setting numOfNodes:", newFormData.ChainConfig.NumOfNodes);
        }
      }
      
      if (params.get('numOfValidators')) {
        const numValidators = parseInt(params.get('numOfValidators')!, 10);
        if (!isNaN(numValidators)) {
          newFormData.ChainConfig.NumOfValidators = numValidators;
          hasChanges = true;
          console.log("Setting numOfValidators:", newFormData.ChainConfig.NumOfValidators);
        }
      }
      
      // Genesis modifications
      const genesisModsParam = params.get('genesisModifications');
      if (genesisModsParam) {
        try {
          const genesisMods = JSON.parse(genesisModsParam);
          if (Array.isArray(genesisMods)) {
            newFormData.ChainConfig.GenesisModifications = genesisMods;
            hasChanges = true;
            console.log("Setting genesisModifications:", newFormData.ChainConfig.GenesisModifications);
          }
        } catch (e) {
          console.error('Failed to parse genesis modifications', e);
        }
      }
      
      if (params.get('longRunningTestnet')) {
        newFormData.LongRunningTestnet = params.get('longRunningTestnet') === 'true';
        hasChanges = true;
        console.log("Setting longRunningTestnet:", newFormData.LongRunningTestnet);
      }
      
      if (params.get('isEvmChain')) {
        newFormData.IsEvmChain = params.get('isEvmChain') === 'true';
        hasChanges = true;
        console.log("Setting evm flag:", newFormData.IsEvmChain);
      }
      
      if (params.get('testnetDuration')) {
        const duration = parseFloat(params.get('testnetDuration')!);
        if (!isNaN(duration)) {
          newFormData.TestnetDuration = duration;
          hasChanges = true;
          console.log("Setting testnetDuration:", newFormData.TestnetDuration);
        }
      }
      
      if (params.get('numWallets')) {
        const numWallets = parseInt(params.get('numWallets')!, 10);
        if (!isNaN(numWallets)) {
          newFormData.NumWallets = numWallets;
          hasChanges = true;
          console.log("Setting numWallets:", newFormData.NumWallets);
        }
      }
      
      const loadTestSpecParam = params.get('loadTestSpec');
      if (loadTestSpecParam) {
        try {
          const parsedLoadTestSpec = JSON.parse(loadTestSpecParam);
          console.log("Parsed loadTestSpec from URL:", parsedLoadTestSpec);
          
          // Normalize the LoadTestSpec structure to match the expected interface
          const normalizedLoadTestSpec: LoadTestSpec = {
            name: parsedLoadTestSpec.Name || parsedLoadTestSpec.name || "",
            description: parsedLoadTestSpec.Description || parsedLoadTestSpec.description || "",
            chain_id: parsedLoadTestSpec.ChainID || parsedLoadTestSpec.chain_id || "",
            NumOfBlocks: parsedLoadTestSpec.NumOfBlocks || 0,
            NumOfTxs: parsedLoadTestSpec.NumOfTxs || 0,
            msgs: Array.isArray(parsedLoadTestSpec.Msgs) 
              ? parsedLoadTestSpec.Msgs.map((msg: any) => ({
                  type: msg.Type || msg.type,
                  weight: msg.Weight || msg.weight || 0,
                  NumMsgs: msg.NumMsgs || msg.numMsgs,
                  ContainedType: msg.ContainedType || msg.containedType,
                  NumOfRecipients: msg.NumOfRecipients || msg.numOfRecipients
                }))
              : (Array.isArray(parsedLoadTestSpec.msgs) 
                ? parsedLoadTestSpec.msgs 
                : []),
            unordered_txs: parsedLoadTestSpec.unordered_txs || false,
            tx_timeout: parsedLoadTestSpec.tx_timeout || "",
          };
          
          console.log("Normalized loadTestSpec:", normalizedLoadTestSpec);
          
          newFormData.LoadTestSpec = normalizedLoadTestSpec;
          setHasLoadTest(true);
          hasChanges = true;
        } catch (e) {
          console.error('Failed to parse load test spec', e);
        }
      }
      
      // Update form data only if there were changes
      if (hasChanges) {
        console.log("Updating form data with:", newFormData);
        setFormData(newFormData);
      } else {
        console.log("No changes to form data");
      }
    }
  }, [location.search]); // Only run when URL parameters change

  const [isLoadTestModalOpen, setIsLoadTestModalOpen] = useState(false);
  const [hasLoadTest, setHasLoadTest] = useState(false);

  // Chain configurations modal state
  const [isChainConfigsModalOpen, setIsChainConfigsModalOpen] = useState(false);

  // Genesis modifications modal state
  const [isGenesisModsModalOpen, setIsGenesisModsModalOpen] = useState(false);

  const createWorkflowMutation = useMutation({
    mutationFn: workflowApi.createWorkflow,
    onSuccess: (data) => {
      toast({
        title: 'Workflow created',
        description: `Workflow ID: ${data.WorkflowID}`,
        status: 'success',
        duration: 5000,
      });
      navigate(`/workflow/${data.WorkflowID}`);
    },
    onError: (error) => {
      toast({
        title: 'Error creating workflow',
        description: error instanceof Error ? error.message : 'Unknown error occurred',
        status: 'error',
        duration: 5000,
      });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    // Validate all required fields are filled
    const requiredFields = [
      { name: 'Repository', value: formData.Repo },
      { name: 'Commit SHA', value: formData.SHA },
      { name: 'Chain Name', value: formData.ChainConfig.Name },
      { name: 'Runner Type', value: formData.RunnerType },
      { name: 'Number of Nodes', value: formData.ChainConfig.NumOfNodes },
      { name: 'Number of Validators', value: formData.ChainConfig.NumOfValidators }
    ];

    // Add Chain Image to required fields if repo is cometbft or ironbird-cometbft
    if (formData.Repo === 'cometbft' || formData.Repo === 'ironbird-cometbft') {
      requiredFields.push({ name: 'Chain Image', value: formData.ChainConfig.Image });
    }

    // Add Testnet Duration if not long running
    if (!formData.LongRunningTestnet) {
      requiredFields.push({ name: 'Testnet Duration', value: formData.TestnetDuration });
    }

    // Check for empty required fields
    const emptyFields = requiredFields.filter(field => 
      field.value === '' || field.value === 0 || field.value === undefined
    );

    if (emptyFields.length > 0) {
      toast({
        title: 'Validation Error',
        description: `Please fill in all required fields: ${emptyFields.map(f => f.name).join(', ')}`,
        status: 'error',
        duration: 5000,
      });
      return;
    }

    // Convert hours to nanoseconds
    let durationInNanos = 0;
    if (!formData.LongRunningTestnet && formData.TestnetDuration) {
      // Convert hours to nanoseconds (hours * 60 * 60 * 10^9)
      durationInNanos = formData.TestnetDuration * 60 * 60 * 1000000000;
    }

    const submissionData: TestnetWorkflowRequest = {
      Repo: formData.Repo,
      SHA: formData.SHA,
      ChainConfig: {
        Name: formData.ChainConfig.Name,
        Image: formData.ChainConfig.Image,
        GenesisModifications: formData.ChainConfig.GenesisModifications || [],
        NumOfNodes: formData.ChainConfig.NumOfNodes,
        NumOfValidators: formData.ChainConfig.NumOfValidators,
        AppConfig: formData.ChainConfig.AppConfig,
        ConsensusConfig: formData.ChainConfig.ConsensusConfig,
        ClientConfig: formData.ChainConfig.ClientConfig,
      },
      IsEvmChain: formData.IsEvmChain || false,
      RunnerType: formData.RunnerType,
      LoadTestSpec: formData.LoadTestSpec,
      LongRunningTestnet: formData.LongRunningTestnet,
      TestnetDuration: durationInNanos,
      NumWallets: formData.NumWallets,
    };

    createWorkflowMutation.mutate(submissionData);
  };
  
  const handleLoadTestSave = (loadTestSpec: LoadTestSpec) => {
    setFormData({
      ...formData,
      LoadTestSpec: loadTestSpec,
    });
    setHasLoadTest(true);
  };

  const handleDeleteLoadTest = () => {
    setFormData({
      ...formData,
      LoadTestSpec: undefined
    });
    setHasLoadTest(false);
  };

  const handleChainConfigsSave = (configs: {
    appConfig?: any;
    consensusConfig?: any;
    clientConfig?: any;
  }) => {
    setFormData({
      ...formData,
      ChainConfig: {
        ...formData.ChainConfig,
        AppConfig: configs.appConfig,
        ConsensusConfig: configs.consensusConfig,
        ClientConfig: configs.clientConfig,
      },
    });
  };

  return (
    <>
      <Box as="form" onSubmit={handleSubmit}>
        <Heading mb={6}>Create New Testnet</Heading>
        <Stack direction="column" gap={4}>
          <FormControl isRequired>
            <FormLabel color="text">Repository</FormLabel>
              <Select
                value={formData.Repo}
                onChange={(e) => setFormData({ ...formData, Repo: e.target.value })}
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder="Select repository"
              >
              <option value="cosmos-sdk">Cosmos SDK</option>
              <option value="cometbft">CometBFT</option>
              <option value="gaia">Gaia</option>
              <option value="ironbird-cosmos-sdk">Ironbird Cosmos SDK</option>
              <option value="ironbird-cometbft">Ironbird CometBFT</option>
            </Select>
          </FormControl>

          <FormControl isRequired>
            <FormLabel color="text">Commit SHA</FormLabel>
              <Input
                value={formData.SHA}
                onChange={(e) => setFormData({ ...formData, SHA: e.target.value })}
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder="Enter commit SHA"
              />
          </FormControl>

          <FormControl isRequired>
            <FormLabel color="text">Chain Name</FormLabel>
              <Input
                value={formData.ChainConfig.Name}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    ChainConfig: {
                      ...formData.ChainConfig,
                      Name: e.target.value,
                    },
                  })
                }
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder={defaultValues.ChainConfig.Name}
              />
          </FormControl>

          <FormControl isRequired>
            <FormLabel color="text">Runner Type</FormLabel>
              <Select
                value={formData.RunnerType}
                onChange={(e) => setFormData({ ...formData, RunnerType: e.target.value })}
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder="Select runner type"
              >
              <option value="Docker">Docker</option>
              <option value="DigitalOcean">DigitalOcean</option>
            </Select>
          </FormControl>

          {(formData.Repo === 'cometbft' || formData.Repo === 'ironbird-cometbft') && (
            <FormControl isRequired>
              <FormLabel color="text">Chain Image</FormLabel>
                <Select
                  value={formData.ChainConfig.Image}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      ChainConfig: {
                        ...formData.ChainConfig,
                        Image: e.target.value,
                      },
                    })
                  }
                  bg="surface"
                  color="text"
                  borderColor="divider"
                  placeholder="Select chain image"
                >
                <option value="simapp-v47">Simapp v0.47</option>
                <option value="simapp-v50">Simapp v0.50</option>
                <option value="simapp-v53">Simapp v0.53</option>
              </Select>
            </FormControl>
          )}

          <FormControl isRequired>
            <FormLabel color="text">Number of Nodes</FormLabel>
              <NumberInput
                value={formData.ChainConfig.NumOfNodes || ''}
                min={1}
                onChange={(_, value) =>
                  setFormData({
                    ...formData,
                    ChainConfig: {
                      ...formData.ChainConfig,
                      NumOfNodes: value || 0,
                    },
                  })
                }
              >
                <NumberInputField 
                  bg="surface" 
                  color="text" 
                  borderColor="divider" 
                  placeholder={defaultValues.ChainConfig.NumOfNodes.toString()}
                />
            </NumberInput>
          </FormControl>

          <FormControl isRequired>
            <FormLabel color="text">Number of Validators</FormLabel>
              <NumberInput
                value={formData.ChainConfig.NumOfValidators || ''}
                min={1}
                onChange={(_, value) =>
                  setFormData({
                    ...formData,
                    ChainConfig: {
                      ...formData.ChainConfig,
                      NumOfValidators: value || 0,
                    },
                  })
                }
              >
                <NumberInputField 
                  bg="surface" 
                  color="text" 
                  borderColor="divider" 
                  placeholder={defaultValues.ChainConfig.NumOfValidators.toString()}
                />
            </NumberInput>
          </FormControl>
          
          <FormControl>
            <FormLabel>Chain Configurations</FormLabel>
            <VStack spacing={3} align="start">
              <HStack width="100%">
                <Button
                  colorScheme="blue"
                  variant="outline"
                  onClick={() => setIsChainConfigsModalOpen(true)}
                >
                  Set Chain Configs
                </Button>
              </HStack>

              {/* Display applied configurations */}
              {(formData.ChainConfig.AppConfig || formData.ChainConfig.ConsensusConfig || formData.ChainConfig.ClientConfig) && (
                <Box width="100%" p={3} bg="green.50" borderRadius="md" border="1px solid" borderColor="green.200">
                  <Text fontWeight="bold" color="green.800" mb={2}>Applied Configurations:</Text>
                  {formData.ChainConfig.AppConfig && (
                    <Text fontSize="sm" color="green.700">• App Config: Applied</Text>
                  )}
                  {formData.ChainConfig.ConsensusConfig && (
                    <Text fontSize="sm" color="green.700">• Consensus Config: Applied</Text>
                  )}
                  {formData.ChainConfig.ClientConfig && (
                    <Text fontSize="sm" color="green.700">• Client Config: Applied</Text>
                  )}
                </Box>
              )}
            </VStack>
          </FormControl>

          <FormControl>
            <FormLabel>Genesis Modifications</FormLabel>
            <VStack spacing={3} align="start">
              <HStack width="100%">
                <Button
                  colorScheme="blue"
                  variant="outline"
                  onClick={() => setIsGenesisModsModalOpen(true)}
                >
                  Set Genesis Modifications
                </Button>
              </HStack>
              
              {/* Display applied modifications */}
              {formData.ChainConfig?.GenesisModifications && formData.ChainConfig.GenesisModifications.length > 0 && (
                <Box width="100%" p={3} bg="green.50" borderRadius="md" border="1px solid" borderColor="green.200">
                  <Text fontWeight="bold" color="green.800" mb={2}>
                    Applied Modifications ({formData.ChainConfig.GenesisModifications.length}):
                  </Text>
                  <VStack align="start" spacing={1} maxH="150px" overflowY="auto">
                    {formData.ChainConfig.GenesisModifications.slice(0, 5).map((mod, index) => (
                      <Text key={index} fontSize="sm" color="green.700">
                        • {mod.key}: {typeof mod.value === 'string' ? mod.value : 'Complex value'}
                      </Text>
                    ))}
                    {formData.ChainConfig.GenesisModifications.length > 5 && (
                      <Text fontSize="sm" color="green.600" fontStyle="italic">
                        ... and {formData.ChainConfig.GenesisModifications.length - 5} more
                      </Text>
                    )}
                  </VStack>
                </Box>
              )}
            </VStack>
          </FormControl>
          
          <FormControl display="flex" alignItems="center">
            <FormLabel mb="0">Run Load Test</FormLabel>
            <Switch
              isChecked={hasLoadTest}
              onChange={(e) => {
                if (!e.target.checked) {
                  setFormData({
                    ...formData,
                    LoadTestSpec: undefined
                  });
                  setHasLoadTest(false);
                } else {
                  setFormData({ 
                    ...formData, 
                    LoadTestSpec: {
                      name: 'basic-load-test',
                      description: 'Basic load test configuration',
                      chain_id: 'test-chain',
                      NumOfBlocks: 100,
                      NumOfTxs: 1000,
                      msgs: [],
                      unordered_txs: false,
                      tx_timeout: '',
                    }
                  });
                  setHasLoadTest(true);
                }
              }}
            />
          </FormControl>
          
          {hasLoadTest && (
            <Button 
              colorScheme="blue" 
              variant="outline" 
              onClick={() => setIsLoadTestModalOpen(true)}
              leftIcon={<AddIcon />}
            >
              Configure Load Test
            </Button>
          )}
          
          {hasLoadTest && formData.LoadTestSpec && formData.LoadTestSpec.msgs.length > 0 && (
            <Box p={3} bg="surface" borderRadius="md" boxShadow="sm">
              <Flex justify="space-between" align="center" mb={2}>
                <Text fontWeight="bold" color="text">Load Test Configuration</Text>
                <IconButton
                  aria-label="Delete load test configuration"
                  icon={<DeleteIcon />}
                  size="sm"
                  colorScheme="red"
                  variant="ghost"
                  onClick={handleDeleteLoadTest}
                />
              </Flex>
              <Text color="text">Name: {formData.LoadTestSpec.name}</Text>
              <Text color="text">Transactions: {formData.LoadTestSpec.NumOfTxs}</Text>
              <Text color="text">Blocks: {formData.LoadTestSpec.NumOfBlocks}</Text>
              {formData.LoadTestSpec.unordered_txs && (
                <>
                  <Text color="text">Unordered Transactions: Yes</Text>
                  <Text color="text">Transaction Timeout: {formData.LoadTestSpec.tx_timeout}</Text>
                </>
              )}
              <Text color="text">Message Types:</Text>
              {formData.LoadTestSpec.msgs.length > 0 ? (
                <VStack align="start" pl={4} spacing={1}>
                  {formData.LoadTestSpec.msgs.map((msg, idx) => {
                    // Get all set properties for this message
                    const pairs = Object.entries(msg)
                      .filter(([_, value]) => value !== undefined && value !== '')
                      .map(([key, value]) => `${key}: ${value}`);
                    
                    return (
                      <Text key={idx} fontSize="sm" color="textSecondary">
                        • {pairs.join(' | ')}
                      </Text>
                    );
                  })}
                </VStack>
              ) : (
                <Text pl={4} color="gray.500" fontSize="sm">No message types configured</Text>
              )}
            </Box>
          )}
          
          <FormControl display="flex" alignItems="center">
            <FormLabel mb="0">Long Running Testnet</FormLabel>
            <Switch
              isChecked={formData.LongRunningTestnet}
              onChange={(e) => setFormData({ ...formData, LongRunningTestnet: e.target.checked })}
            />
          </FormControl>
          
          {!formData.LongRunningTestnet && (
            <FormControl>
              <FormLabel>Testnet Duration (Hours)</FormLabel>
              <NumberInput
                min={1}
                value={formData.TestnetDuration || ''}
                onChange={(_, value) => {
                  setFormData({ 
                    ...formData, 
                    TestnetDuration: value || 0
                  });
                }}
              >
                <NumberInputField placeholder={defaultValues.TestnetDuration.toString()} />
              </NumberInput>
              <FormHelperText>Duration in hours</FormHelperText>
            </FormControl>
          )}

          <FormControl>
            <FormLabel>Number of Wallets</FormLabel>
            <NumberInput
              min={1}
              value={formData.NumWallets || ''}
              onChange={(_, value) => {
                setFormData({ 
                  ...formData, 
                  NumWallets: value || 2500
                });
              }}
            >
              <NumberInputField placeholder="2500" />
            </NumberInput>
            <FormHelperText>Number of wallets to create</FormHelperText>
          </FormControl>

          <Button
            mt={4}
            colorScheme="blue"
            disabled={createWorkflowMutation.isPending}
            type="submit"
          >
            Create Testnet
          </Button>
        </Stack>
      </Box>
      
      <LoadTestForm
        isOpen={isLoadTestModalOpen}
        onClose={() => setIsLoadTestModalOpen(false)}
        initialData={formData.LoadTestSpec || {
          name: 'basic-load-test',
          description: 'Basic load test configuration',
          chain_id: formData.ChainConfig.Name || 'test-chain',
          NumOfBlocks: 100,
          NumOfTxs: 1000,
          msgs: [],
          unordered_txs: false,
          tx_timeout: '',
        }}
        onSave={handleLoadTestSave}
      />
      
      <ChainConfigsModal
        isOpen={isChainConfigsModalOpen}
        onClose={() => setIsChainConfigsModalOpen(false)}
        initialAppConfig={formData.ChainConfig.AppConfig}
        initialConsensusConfig={formData.ChainConfig.ConsensusConfig}
        initialClientConfig={formData.ChainConfig.ClientConfig}
        onSave={handleChainConfigsSave}
      />

      <GenesisModificationsModal
        isOpen={isGenesisModsModalOpen}
        onClose={() => setIsGenesisModsModalOpen(false)}
        initialModifications={formData.ChainConfig.GenesisModifications}
        onSave={(newModifications) => setFormData({
          ...formData,
          ChainConfig: {
            ...formData.ChainConfig,
            GenesisModifications: newModifications,
          },
        })}
      />
    </>
  );
};

export default CreateWorkflow;

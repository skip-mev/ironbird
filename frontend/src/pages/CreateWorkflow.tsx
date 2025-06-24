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
import type { TestnetWorkflowRequest, GenesisModification, LoadTestSpec } from '../types/workflow';
import { AddIcon, DeleteIcon } from '@chakra-ui/icons';
import LoadTestForm from '../components/LoadTestForm';

const EVM_GENESIS_MODIFICATIONS: GenesisModification[] = [
  {
    key: "app_state.staking.params.bond_denom",
    value: "uatom",
  },
  {
    key: "app_state.gov.params.expedited_voting_period",
    value: "120s",
  },
  {
    key: "app_state.gov.params.voting_period",
    value: "300s",
  },
  {
    key: "app_state.gov.params.expedited_min_deposit.0.amount",
    value: "1",
  },
  {
    key: "app_state.gov.params.expedited_min_deposit.0.denom",
    value: "uatom",
  },
  {
    key: "app_state.gov.params.min_deposit.0.amount",
    value: "1",
  },
  {
    key: "app_state.gov.params.min_deposit.0.denom",
    value: "uatom",
  },
  {
    key: "app_state.evm.params.evm_denom",
    value: "uatom",
  },
  {
    key: "app_state.mint.params.mint_denom",
    value: "uatom",
  },
  {
    key: "app_state.bank.denom_metadata",
    value: [
      {
        "description": "The native staking token for evmd.",
        "denom_units": [
          {
            "denom": "uatom",
            "exponent": 0,
            "aliases": ["attotest"],
          },
          {
            "denom": "test",
            "exponent": 18,
            "aliases": [],
          },
        ],
        "base": "uatom",
        "display": "test",
        "name": "Test Token",
        "symbol": "TEST",
        "uri": "",
        "uri_hash": "",
      },
    ],
  },
  {
    key: "app_state.evm.params.active_static_precompiles",
    value: [
      "0x0000000000000000000000000000000000000100",
      "0x0000000000000000000000000000000000000400",
      "0x0000000000000000000000000000000000000800",
      "0x0000000000000000000000000000000000000801",
      "0x0000000000000000000000000000000000000802",
      "0x0000000000000000000000000000000000000803",
      "0x0000000000000000000000000000000000000804",
      "0x0000000000000000000000000000000000000805",
    ],
  },
  {
    key: "app_state.erc20.params.native_precompiles",
    value: ["0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"],
  },
  {
    key: "app_state.erc20.token_pairs",
    value: [
      {
        "contract_owner": 1,
        "erc20_address": "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE",
        "denom": "uatom",
        "enabled": true,
      },
    ],
  },
  {
    key: "consensus.params.block.max_gas",
    value: "75000000",
  },
];

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
    evm: false,
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
    evm: false,
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
        evm: false,
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
      
      if (params.get('evm')) {
        newFormData.evm = params.get('evm') === 'true';
        hasChanges = true;
        console.log("Setting evm flag:", newFormData.evm);
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

  const [newKeyValue, setNewKeyValue] = useState<GenesisModification>({
    key: '',
    value: '',
  });
  
  // Track if we're in JSON mode for the value input
  const [valueInputMode, setValueInputMode] = useState<'simple' | 'json'>('simple');
  const [valueJsonInput, setValueJsonInput] = useState<string>('');
  
  const [isLoadTestModalOpen, setIsLoadTestModalOpen] = useState(false);
  const [hasLoadTest, setHasLoadTest] = useState(false);

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
      },
      evm: formData.evm || false,
      RunnerType: formData.RunnerType,
      LoadTestSpec: formData.LoadTestSpec,
      LongRunningTestnet: formData.LongRunningTestnet,
      TestnetDuration: durationInNanos,
      NumWallets: formData.NumWallets,
    };

    createWorkflowMutation.mutate(submissionData);
  };

  const addGenesisModification = () => {
    if (newKeyValue.key.trim() === '') {
      toast({
        title: 'Validation Error',
        description: 'Key must be provided',
        status: 'error',
        duration: 3000,
      });
      return;
    }

    let value = newKeyValue.value;
    
    // If in JSON mode, try to parse the JSON
    if (valueInputMode === 'json') {
      if (valueJsonInput.trim() === '') {
        toast({
          title: 'Validation Error',
          description: 'Value must be provided',
          status: 'error',
          duration: 3000,
        });
        return;
      }
      
      try {
        value = JSON.parse(valueJsonInput);
      } catch (error) {
        toast({
          title: 'JSON Parse Error',
          description: 'Please provide valid JSON for the value',
          status: 'error',
          duration: 3000,
        });
        return;
      }
    } else {
      // Simple mode - check for string value
      if (typeof value === 'string' && value.trim() === '') {
        toast({
          title: 'Validation Error',
          description: 'Value must be provided',
          status: 'error',
          duration: 3000,
        });
        return;
      }
    }

    const updatedModifications = [
      ...(formData.ChainConfig?.GenesisModifications || []),
      { key: newKeyValue.key, value },
    ];

    setFormData({
      ...formData,
      ChainConfig: {
        ...formData.ChainConfig!,
        GenesisModifications: updatedModifications,
      },
    });

    // Reset form
    setNewKeyValue({ key: '', value: '' });
    setValueJsonInput('');
    setValueInputMode('simple');
  };

  const removeGenesisModification = (index: number) => {
    const updatedModifications = [...(formData.ChainConfig?.GenesisModifications || [])];
    updatedModifications.splice(index, 1);

    setFormData({
      ...formData,
      ChainConfig: {
        ...formData.ChainConfig!,
        GenesisModifications: updatedModifications,
      },
    });
  };

  const setEVMGenesisModifications = () => {
    setFormData({
      ...formData,
      ChainConfig: {
        ...formData.ChainConfig!,
        GenesisModifications: [...EVM_GENESIS_MODIFICATIONS],
      },
      evm: true,
    });

    toast({
      title: 'EVM Genesis Modifications Set',
      description: 'Applied EVM genesis modifications',
      status: 'success',
      duration: 3000,
    });
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
            <FormLabel>Genesis Modifications</FormLabel>
            <VStack spacing={3} align="start">
              <HStack width="100%">
                <Button
                  colorScheme="purple"
                  variant="outline"
                  size="sm"
                  onClick={setEVMGenesisModifications}
                >
                  Set EVM Genesis Modifications
                </Button>
              </HStack>
              
              {formData.ChainConfig?.GenesisModifications?.map((mod, index) => (
                <HStack key={index} width="100%">
                  <Text flex="1" fontWeight="medium">{mod.key}:</Text>
                  <Text flex="2" fontSize="sm" fontFamily="mono" 
                        bg="surface" p={2} borderRadius="md"
                        color="text" maxH="100px" overflowY="auto">
                    {typeof mod.value === 'string' 
                      ? mod.value 
                      : JSON.stringify(mod.value, null, 2)}
                  </Text>
                  <IconButton
                    aria-label="Remove modification"
                    size="sm"
                    icon={<DeleteIcon />}
                    onClick={() => removeGenesisModification(index)}
                  />
                </HStack>
              ))}
              
              <VStack spacing={2} align="start" width="100%">
                <HStack width="100%">
                  <Switch
                    isChecked={valueInputMode === 'json'}
                    onChange={(e) => {
                      setValueInputMode(e.target.checked ? 'json' : 'simple');
                      if (e.target.checked) {
                        // Convert simple value to JSON string if switching to JSON mode
                        setValueJsonInput(
                          typeof newKeyValue.value === 'string' && newKeyValue.value.trim() !== ''
                            ? JSON.stringify(newKeyValue.value, null, 2)
                            : ''
                        );
                      } else {
                        // Convert JSON back to simple string if possible
                        if (valueJsonInput.trim() !== '') {
                          try {
                            const parsed = JSON.parse(valueJsonInput);
                            if (typeof parsed === 'string') {
                              setNewKeyValue({ ...newKeyValue, value: parsed });
                            } else {
                              setNewKeyValue({ ...newKeyValue, value: '' });
                            }
                          } catch {
                            setNewKeyValue({ ...newKeyValue, value: '' });
                          }
                        }
                      }
                    }}
                  />
                  <Text fontSize="sm">JSON Mode (for complex values like arrays/objects)</Text>
                </HStack>
                
                <HStack width="100%">
                  <Input
                    placeholder="Key"
                    value={newKeyValue.key}
                    onChange={(e) => setNewKeyValue({ ...newKeyValue, key: e.target.value })}
                    mr={2}
                    bg="surface"
                    color="text"
                    borderColor="divider"
                  />
                  {valueInputMode === 'json' ? (
                    <Textarea
                    placeholder='Value (JSON) - e.g. ["item1", "item2"] or {"key": "value"}'
                    value={valueJsonInput}
                    onChange={(e) => setValueJsonInput(e.target.value)}
                    mr={2}
                    resize="vertical"
                    minH="60px"
                    bg="surface"
                    color="text"
                    borderColor="divider"
                    />
                  ) : (
                    <Input
                      placeholder="Value"
                      value={newKeyValue.value}
                      onChange={(e) => setNewKeyValue({ ...newKeyValue, value: e.target.value })}
                      mr={2}
                      bg="surface"
                      color="text"
                      borderColor="divider"
                    />
                  )}
                  <IconButton
                    aria-label="Add modification"
                    icon={<AddIcon />}
                    onClick={addGenesisModification}
                  />
                </HStack>
              </VStack>
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
    </>
  );
};

export default CreateWorkflow;

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
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

// Gaia genesis modifications preset
const GAIA_GENESIS_MODIFICATIONS: GenesisModification[] = [
  {
    key: "app_state.staking.params.bond_denom",
    value: "atest",
  },
  {
    key: "app_state.gov.deposit_params.min_deposit.0.denom",
    value: "atest",
  },
  {
    key: "app_state.gov.params.min_deposit.0.denom",
    value: "atest",
  },
  {
    key: "app_state.evm.params.evm_denom",
    value: "atest",
  },
  {
    key: "app_state.mint.params.mint_denom",
    value: "atest",
  },
  {
    key: "app_state.bank.denom_metadata",
    value: [
      {
        "description": "The native staking token for evmd.",
        "denom_units": [
          {
            "denom": "atest",
            "exponent": 0,
            "aliases": ["attotest"],
          },
          {
            "denom": "test",
            "exponent": 18,
            "aliases": [],
          },
        ],
        "base": "atest",
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
        "denom": "atest",
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
  const toast = useToast();
  const [formData, setFormData] = useState<TestnetWorkflowRequest>({
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
    LoadTestSpec: undefined,
    LongRunningTestnet: false,
    TestnetDuration: 2, // 2 hours
  });

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
        Name: formData.ChainConfig.Name || 'test-chain',
        Image: formData.ChainConfig.Image,
        GenesisModifications: formData.ChainConfig.GenesisModifications || [],
        NumOfNodes: formData.ChainConfig.NumOfNodes || 4,
        NumOfValidators: formData.ChainConfig.NumOfValidators || 3,
      },
      RunnerType: formData.RunnerType,
      LoadTestSpec: formData.LoadTestSpec,
      LongRunningTestnet: formData.LongRunningTestnet,
      TestnetDuration: durationInNanos,
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

  const setGaiaGenesisModifications = () => {
    setFormData({
      ...formData,
      ChainConfig: {
        ...formData.ChainConfig!,
        GenesisModifications: [...GAIA_GENESIS_MODIFICATIONS],
      },
    });

    toast({
      title: 'Gaia Genesis Modifications Set',
      description: 'Applied Gaia EVM genesis modifications',
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
            <FormLabel>Repository</FormLabel>
            <Select
              value={formData.Repo}
              onChange={(e) => setFormData({ ...formData, Repo: e.target.value })}
            >
              <option value="cosmos-sdk">Cosmos SDK</option>
              <option value="cometbft">CometBFT</option>
              <option value="gaia">Gaia</option>
              <option value="ironbird-cosmos-sdk">Ironbird Cosmos SDK</option>
              <option value="ironbird-cometbft">Ironbird CometBFT</option>
            </Select>
          </FormControl>

          <FormControl isRequired>
            <FormLabel>Commit SHA</FormLabel>
            <Input
              value={formData.SHA}
              onChange={(e) => setFormData({ ...formData, SHA: e.target.value })}
            />
          </FormControl>

          <FormControl isRequired>
            <FormLabel>Chain Name</FormLabel>
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
            />
          </FormControl>

          {(formData.Repo === 'cometbft' || formData.Repo === 'ironbird-cometbft') && (
            <FormControl isRequired>
              <FormLabel>Chain Image</FormLabel>
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
              >
                <option value="simapp-v47">Simapp v0.47</option>
                <option value="simapp-v50">Simapp v0.50</option>
                <option value="simapp-v53">Simapp v0.53</option>
              </Select>
            </FormControl>
          )}

          <FormControl isRequired>
            <FormLabel>Number of Nodes</FormLabel>
            <NumberInput
              value={formData.ChainConfig.NumOfNodes}
              min={1}
              onChange={(_, value) =>
                setFormData({
                  ...formData,
                  ChainConfig: {
                    ...formData.ChainConfig,
                    NumOfNodes: value || 4,
                  },
                })
              }
            >
              <NumberInputField />
            </NumberInput>
          </FormControl>

          <FormControl isRequired>
            <FormLabel>Number of Validators</FormLabel>
            <NumberInput
              value={formData.ChainConfig.NumOfValidators}
              min={1}
              onChange={(_, value) =>
                setFormData({
                  ...formData,
                  ChainConfig: {
                    ...formData.ChainConfig,
                    NumOfValidators: value || 3,
                  },
                })
              }
            >
              <NumberInputField />
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
                  onClick={setGaiaGenesisModifications}
                >
                  Set Gaia Genesis Modifications
                </Button>
              </HStack>
              
              {formData.ChainConfig?.GenesisModifications?.map((mod, index) => (
                <HStack key={index} width="100%">
                  <Text flex="1" fontWeight="medium">{mod.key}:</Text>
                  <Text flex="2" fontSize="sm" fontFamily="mono" 
                        bg="gray.50" p={2} borderRadius="md"
                        maxH="100px" overflowY="auto">
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
                  />
                  {valueInputMode === 'json' ? (
                    <Textarea
                      placeholder='Value (JSON) - e.g. ["item1", "item2"] or {"key": "value"}'
                      value={valueJsonInput}
                      onChange={(e) => setValueJsonInput(e.target.value)}
                      mr={2}
                      resize="vertical"
                      minH="60px"
                    />
                  ) : (
                    <Input
                      placeholder="Value"
                      value={newKeyValue.value}
                      onChange={(e) => setNewKeyValue({ ...newKeyValue, value: e.target.value })}
                      mr={2}
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
                      num_of_blocks: 100,
                      num_of_txs: 1000,
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
            <Box p={3} bg="blue.50" borderRadius="md">
              <Flex justify="space-between" align="center" mb={2}>
                <Text fontWeight="bold">Load Test Configuration</Text>
                <IconButton
                  aria-label="Delete load test configuration"
                  icon={<DeleteIcon />}
                  size="sm"
                  colorScheme="red"
                  variant="ghost"
                  onClick={handleDeleteLoadTest}
                />
              </Flex>
              <Text>Name: {formData.LoadTestSpec.name}</Text>
              <Text>Transactions: {formData.LoadTestSpec.num_of_txs}</Text>
              <Text>Blocks: {formData.LoadTestSpec.num_of_blocks}</Text>
              {formData.LoadTestSpec.unordered_txs && (
                <>
                  <Text>Unordered Transactions: Yes</Text>
                  <Text>Transaction Timeout: {formData.LoadTestSpec.tx_timeout}</Text>
                </>
              )}
              <Text>Message Types:</Text>
              {formData.LoadTestSpec.msgs.length > 0 ? (
                <VStack align="start" pl={4} spacing={1}>
                  {formData.LoadTestSpec.msgs.map((msg, idx) => {
                    // Get all set properties for this message
                    const pairs = Object.entries(msg)
                      .filter(([_, value]) => value !== undefined && value !== '')
                      .map(([key, value]) => `${key}: ${value}`);
                    
                    return (
                      <Text key={idx} fontSize="sm" color="gray.700">
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
                value={formData.TestnetDuration}
                onChange={(_, value) => {
                  setFormData({ 
                    ...formData, 
                    TestnetDuration: value || 2
                  });
                }}
              >
                <NumberInputField />
              </NumberInput>
              <FormHelperText>Duration in hours</FormHelperText>
            </FormControl>
          )}

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
          num_of_blocks: 100,
          num_of_txs: 1000,
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

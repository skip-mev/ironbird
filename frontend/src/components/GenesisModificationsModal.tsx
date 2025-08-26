import { useState, useEffect } from 'react';
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
  Button,
  VStack,
  HStack,
  Text,
  Box,
  IconButton,
  useToast,
  FormLabel,
  Code,
  Flex,
  Input,
  Textarea,
  Switch,
  Divider,
} from '@chakra-ui/react';
import { CopyIcon, AddIcon, DeleteIcon } from '@chakra-ui/icons';
import type { GenesisModification } from '../types/workflow';

interface GenesisModificationsModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialModifications: GenesisModification[];
  onSave: (modifications: GenesisModification[]) => void;
}

// Sample genesis modifications that users can copy
const generateSampleEvmModifications = (): GenesisModification[] => [
  {
    key: "app_state.staking.params.bond_denom",
    value: "atest"
  },
  {
    key: "app_state.gov.params.expedited_voting_period",
    value: "120s"
  },
  {
    key: "app_state.gov.params.voting_period",
    value: "300s"
  },
  {
    key: "app_state.gov.params.expedited_min_deposit.0.amount",
    value: "1"
  },
  {
    key: "app_state.gov.params.expedited_min_deposit.0.denom",
    value: "atest"
  },
  {
    key: "app_state.gov.params.min_deposit.0.amount",
    value: "1"
  },
  {
    key: "app_state.gov.params.min_deposit.0.denom",
    value: "atest"
  },
  {
    key: "app_state.evm.params.evm_denom",
    value: "atest"
  },
  {
    key: "app_state.mint.params.mint_denom",
    value: "atest"
  },
  {
    key: "app_state.bank.denom_metadata",
    value: "[{\"description\":\"The native staking token for evmd.\",\"denom_units\":[{\"denom\":\"atest\",\"exponent\":0,\"aliases\":[\"attotest\"]},{\"denom\":\"test\",\"exponent\":18,\"aliases\":[]}],\"base\":\"atest\",\"display\":\"test\",\"name\":\"Test Token\",\"symbol\":\"TEST\",\"uri\":\"\",\"uri_hash\":\"\"}]"
  },
  {
    key: "app_state.evm.params.active_static_precompiles",
    value: "[\"0x0000000000000000000000000000000000000100\",\"0x0000000000000000000000000000000000000400\",\"0x0000000000000000000000000000000000000800\",\"0x0000000000000000000000000000000000000801\",\"0x0000000000000000000000000000000000000802\",\"0x0000000000000000000000000000000000000803\",\"0x0000000000000000000000000000000000000804\",\"0x0000000000000000000000000000000000000805\"]"
  },
  {
    key: "app_state.erc20.token_pairs",
    value: "[{\"contract_owner\": 1,\"erc20_address\": \"0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE\",\"denom\": \"atest\",\"enabled\": true}]"
  },
  {
    key: "consensus.params.block.max_gas",
    value: "75000000"
  }
];

const GenesisModificationsModal = ({ 
  isOpen, 
  onClose, 
  initialModifications,
  onSave 
}: GenesisModificationsModalProps) => {
  const [modifications, setModifications] = useState<GenesisModification[]>([]);
  const [newKeyValue, setNewKeyValue] = useState<GenesisModification>({
    key: '',
    value: '',
  });
  const [valueInputMode, setValueInputMode] = useState<'simple' | 'json'>('simple');
  const [valueJsonInput, setValueJsonInput] = useState<string>('');
  const toast = useToast();

  // Initialize form data when modal opens
  useEffect(() => {
    if (isOpen) {
      setModifications([...initialModifications]);
      setNewKeyValue({ key: '', value: '' });
      setValueJsonInput('');
      setValueInputMode('simple');
    }
  }, [isOpen, initialModifications]);

  const copyToClipboard = (modifications: GenesisModification[], modType: string) => {
    const jsonString = JSON.stringify(modifications, null, 2);
    navigator.clipboard.writeText(jsonString).then(() => {
      toast({
        title: 'Copied to clipboard',
        description: `Sample ${modType} modifications copied`,
        status: 'success',
        duration: 2000,
      });
    });
  };

  const loadSampleModifications = (sampleMods: GenesisModification[], modType: string) => {
    setModifications([...sampleMods]);
    toast({
      title: 'Sample modifications loaded',
      description: `${modType} modifications have been loaded`,
      status: 'success',
      duration: 2000,
    });
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
      ...modifications,
      { key: newKeyValue.key, value },
    ];

    setModifications(updatedModifications);

    // Reset form
    setNewKeyValue({ key: '', value: '' });
    setValueJsonInput('');
    setValueInputMode('simple');
  };

  const removeGenesisModification = (index: number) => {
    const updatedModifications = [...modifications];
    updatedModifications.splice(index, 1);
    setModifications(updatedModifications);
  };

  const handleSave = () => {
    onSave(modifications);
    toast({
      title: 'Genesis modifications saved',
      description: 'Genesis modifications have been applied',
      status: 'success',
      duration: 3000,
    });
    onClose();
  };

  const clearAll = () => {
    setModifications([]);
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="6xl">
      <ModalOverlay />
      <ModalContent bg="surface" maxH="90vh">
        <ModalHeader color="text">Set Genesis Modifications</ModalHeader>
        <ModalCloseButton />
        <ModalBody overflowY="auto">
          <VStack spacing={6} align="stretch">
            <Text color="text" fontSize="sm">
              Configure genesis modifications for your chain. Use the sample configurations on the right as a starting point.
            </Text>

            <Flex gap={6} h="500px">
              {/* Left side - Current modifications and add new */}
              <VStack flex="1" align="stretch" spacing={4}>
                <FormLabel fontSize="sm" color="text">
                  Current Modifications ({modifications.length})
                </FormLabel>
                
                {/* Current modifications list */}
                <Box
                  h="200px"
                  overflowY="auto"
                  border="1px solid"
                  borderColor="divider"
                  borderRadius="md"
                  p={3}
                >
                  {modifications.length === 0 ? (
                    <Text fontSize="sm" color="gray.500" textAlign="center" mt={4}>
                      No modifications added yet
                    </Text>
                  ) : (
                    <VStack spacing={2} align="stretch">
                      {modifications.map((mod, index) => (
                        <HStack key={index} width="100%" p={2} bg="gray.50" borderRadius="md">
                          <VStack flex="1" align="start" spacing={1}>
                            <Text fontSize="xs" fontWeight="medium" color="gray.700">
                              {mod.key}
                            </Text>
                            <Text fontSize="xs" fontFamily="mono" color="gray.600" noOfLines={2}>
                              {typeof mod.value === 'string' 
                                ? mod.value 
                                : JSON.stringify(mod.value)}
                            </Text>
                          </VStack>
                          <IconButton
                            aria-label="Remove modification"
                            size="xs"
                            icon={<DeleteIcon />}
                            onClick={() => removeGenesisModification(index)}
                            colorScheme="red"
                            variant="ghost"
                          />
                        </HStack>
                      ))}
                    </VStack>
                  )}
                </Box>

                <Divider />

                {/* Add new modification */}
                <VStack spacing={3} align="start">
                  <FormLabel fontSize="sm" color="text">
                    Add New Modification
                  </FormLabel>
                  
                  <HStack width="100%">
                    <Switch
                      isChecked={valueInputMode === 'json'}
                      onChange={(e) => {
                        setValueInputMode(e.target.checked ? 'json' : 'simple');
                        if (e.target.checked) {
                          setValueJsonInput(
                            typeof newKeyValue.value === 'string' && newKeyValue.value.trim() !== ''
                              ? JSON.stringify(newKeyValue.value, null, 2)
                              : ''
                          );
                        } else {
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
                  
                  <Input
                    placeholder="Key (e.g., app_state.staking.params.bond_denom)"
                    value={newKeyValue.key}
                    onChange={(e) => setNewKeyValue({ ...newKeyValue, key: e.target.value })}
                    bg="surface"
                    color="text"
                    borderColor="divider"
                    fontSize="sm"
                  />
                  
                  {valueInputMode === 'json' ? (
                    <Textarea
                      placeholder='Value (JSON) - e.g. ["item1", "item2"] or {"key": "value"}'
                      value={valueJsonInput}
                      onChange={(e) => setValueJsonInput(e.target.value)}
                      resize="vertical"
                      minH="80px"
                      bg="surface"
                      color="text"
                      borderColor="divider"
                      fontFamily="mono"
                      fontSize="sm"
                    />
                  ) : (
                    <Input
                      placeholder="Value (e.g., stake)"
                      value={newKeyValue.value}
                      onChange={(e) => setNewKeyValue({ ...newKeyValue, value: e.target.value })}
                      bg="surface"
                      color="text"
                      borderColor="divider"
                      fontSize="sm"
                    />
                  )}
                  
                  <Button leftIcon={<AddIcon />} onClick={addGenesisModification} size="sm" alignSelf="flex-end">
                    Add Modification
                  </Button>
                </VStack>
              </VStack>
              
              {/* Right side - Sample configurations */}
              <VStack flex="1" align="stretch" spacing={4}>
                <FormLabel fontSize="sm" color="text">
                  Sample Configuration
                </FormLabel>
                
                {/* EVM Configuration */}
                <Box border="1px solid" borderColor="divider" borderRadius="md" p={3} h="100%">
                  <HStack justify="space-between" mb={2}>
                    <Text fontSize="sm" fontWeight="medium" color="text">
                      EVM Configuration
                    </Text>
                    <HStack spacing={1}>
                      <IconButton
                        aria-label="Copy EVMD config"
                        icon={<CopyIcon />}
                        size="xs"
                        onClick={() => copyToClipboard(generateSampleEvmModifications(), 'EVM')}
                      />
                      <Button
                        size="xs"
                        colorScheme="purple"
                        variant="outline"
                        onClick={() => loadSampleModifications(generateSampleEvmModifications(), 'EVM')}
                      >
                        Load
                      </Button>
                    </HStack>
                  </HStack>
                  <Box
                    h="400px"
                    overflowY="auto"
                    bg="gray.50"
                    borderRadius="md"
                    p={2}
                  >
                    <Code
                      fontSize="xs"
                      whiteSpace="pre-wrap"
                      display="block"
                      bg="transparent"
                      color="gray.800"
                      overflowWrap="break-word"
                      wordBreak="break-all"
                    >
                      {JSON.stringify(generateSampleEvmModifications(), null, 2)}
                    </Code>
                  </Box>
                </Box>
              </VStack>
            </Flex>
          </VStack>
        </ModalBody>
        
        <ModalFooter>
          <HStack spacing={3}>
            <Button variant="ghost" onClick={clearAll}>
              Clear All
            </Button>
            <Button variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button colorScheme="blue" onClick={handleSave}>
              Save Modifications
            </Button>
          </HStack>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};

export default GenesisModificationsModal; 
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
  FormControl,
  FormLabel,
  Input,
  NumberInput,
  NumberInputField,
  Switch,
  VStack,
  HStack,
  Select,
  Text,
  IconButton,
  Divider,
  useToast,
  FormHelperText,
} from '@chakra-ui/react';
import { AddIcon, DeleteIcon } from '@chakra-ui/icons';
import type { LoadTestSpec, Message } from '../types/workflow';
import { MsgType } from '../types/workflow';

interface LoadTestFormProps {
  isOpen: boolean;
  onClose: () => void;
  initialData: LoadTestSpec;
  onSave: (loadTestSpec: LoadTestSpec) => void;
  selectedRepo?: string;
}

const LoadTestForm = ({ isOpen, onClose, initialData, onSave, selectedRepo }: LoadTestFormProps) => {
  const [defaultValues, setDefaultValues] = useState<LoadTestSpec>(initialData);
  
  const [formData, setFormData] = useState<LoadTestSpec>({
    name: '',
    description: '',
    chain_id: '',
    kind: 'cosmos',
    NumOfBlocks: 0,
    NumOfTxs: 0,
    msgs: [],
    unordered_txs: false,
    tx_timeout: '',
    send_interval: '',
    num_batches: 0,
    gas_denom: '',
    bech32_prefix: '',
  });
  
  const [newMessage, setNewMessage] = useState<Message>({
    type: MsgType.MsgSend,
    weight: 0.5,
    num_msgs: 1,
  });
  const toast = useToast();

  // Reset form data when modal is opened
  useEffect(() => {
    // Update default values for placeholders
    setDefaultValues(initialData);
    
    // Determine the correct kind based on selected repo
    let kind: 'cosmos' | 'eth' = 'cosmos';
    if (selectedRepo === 'evm') {
      kind = 'eth';
    } else if (selectedRepo && selectedRepo !== 'evm') {
      kind = 'cosmos';
    } else {
      kind = initialData.kind || 'cosmos';
    }
    
    // Initialize with empty form
    setFormData({
      name: '',
      description: '',
      chain_id: initialData.chain_id || '', // Keep chain_id from initialData
      kind: kind,
      NumOfBlocks: 0,
      NumOfTxs: 0,
      msgs: [],
      unordered_txs: false,
      tx_timeout: '',
      send_interval: '',
      num_batches: 0,
      gas_denom: '',
      bech32_prefix: '',
    });
  }, [initialData, isOpen, selectedRepo]);

  const calculateTotalWeight = (): number => {
    if (!Array.isArray(formData.msgs)) {
      return 0; // Or handle as an error, though 0 is safe for toFixed()
    }
    return formData.msgs.reduce((total, msg) => {
      const weight = msg.weight ? Number(msg.weight) : 0; // Handle undefined weight
      return total + (isNaN(weight) ? 0 : weight); // Add 0 if weight is NaN
    }, 0);
  };

  const handleSubmit = () => {
    // Validate required fields
    const requiredFields = [
      { name: 'Test Name', value: formData.name },
      { name: 'Description', value: formData.description },
      { name: 'Number of Transactions', value: formData.NumOfTxs },
      { name: 'Number of Blocks', value: formData.NumOfBlocks }
    ];

    // Add Transaction Timeout if unordered_txs is true
    if (formData.unordered_txs) {
      requiredFields.push({ name: 'Transaction Timeout', value: formData.tx_timeout });
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

    // Validate total weight should be exactly 1.0 if there are any Cosmos messages
    if (formData.kind === 'cosmos' && formData.msgs.length > 0) {
      const totalWeight = calculateTotalWeight();
      if (Math.abs(totalWeight - 1.0) > 0.001) { // Allow small rounding errors
        toast({
          title: 'Validation Error',
          description: `Total message weight must be exactly 1.0. Current: ${totalWeight.toFixed(2)}`,
          status: 'error',
          duration: 3000,
        });
        return;
      }
    }

    // Check for duplicate message types
    const messageTypes = new Set<string>();
    for (const msg of formData.msgs) {
      const typeKey = msg.type === MsgType.MsgArr 
        ? `${msg.type}-${msg.ContainedType}` 
        : msg.type;
      
      if (messageTypes.has(typeKey)) {
        toast({
          title: 'Validation Error',
          description: `Duplicate message type: ${msg.type}${msg.type === MsgType.MsgArr ? ` with contained type ${msg.ContainedType}` : ''}`,
          status: 'error',
          duration: 3000,
        });
        return;
      }
      
      messageTypes.add(typeKey);
    }

    // If UnorderedTxs is false, make sure TxTimeout is not set
    const dataToSave = { ...formData };
    if (!dataToSave.unordered_txs) {
      dataToSave.tx_timeout = '';
    }

    onSave(dataToSave);
    onClose();
  };

  const addMessage = () => {
    // Basic validation for Cosmos messages (weight required)
    if (formData.kind === 'cosmos' && (newMessage.weight <= 0 || newMessage.weight > 1)) {
      toast({
        title: 'Validation Error',
        description: 'Weight must be between 0 and 1',
        status: 'error',
        duration: 3000,
      });
      return;
    }
    
    // Basic validation for Ethereum messages (num_msgs required)
    if (formData.kind === 'eth' && (!newMessage.num_msgs || newMessage.num_msgs <= 0)) {
      toast({
        title: 'Validation Error',
        description: 'Number of messages must be greater than 0',
        status: 'error',
        duration: 3000,
      });
      return;
    }

    // Additional validation for MsgArr type
    if (newMessage.type === MsgType.MsgArr && (!newMessage.ContainedType || !newMessage.NumMsgs)) {
      toast({
        title: 'Validation Error',
        description: 'MsgArr type requires Contained Type and Number of Messages',
        status: 'error',
        duration: 3000,
      });
      return;
    }

    // Check for duplicate message type (only for Cosmos messages with MsgArr special case)
    if (formData.kind === 'cosmos') {
      const duplicateType = formData.msgs.find(msg => 
        msg.type === newMessage.type && 
        // For MsgArr, we also need to check containedType
        (msg.type !== MsgType.MsgArr || 
          (msg.type === MsgType.MsgArr && msg.ContainedType === newMessage.ContainedType))
      );

      if (duplicateType) {
        toast({
          title: 'Validation Error',
          description: newMessage.type === MsgType.MsgArr 
            ? `A message with type ${newMessage.type} and contained type ${newMessage.ContainedType} already exists` 
            : `A message with type ${newMessage.type} already exists`,
          status: 'error',
          duration: 3000,
        });
        return;
      }
    }

    const updatedMessages = [...formData.msgs, { ...newMessage }];
    setFormData({
      ...formData,
      msgs: updatedMessages,
    });

    // Reset new message form based on load test type
    if (formData.kind === 'cosmos') {
      setNewMessage({
        type: MsgType.MsgSend,
        weight: 0.5,
      });
    } else {
      setNewMessage({
        type: MsgType.MsgCreateContract,
        num_msgs: 1,
      });
    }
  };

  const removeMessage = (index: number) => {
    const updatedMessages = [...formData.msgs];
    updatedMessages.splice(index, 1);
    setFormData({
      ...formData,
      msgs: updatedMessages,
    });
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="xl">
      <ModalOverlay />
      <ModalContent bg="surface">
        <ModalHeader color="text">Load Test Configuration</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <VStack spacing={4} align="stretch">
            
            <FormControl>
              <FormLabel color="text">Test Name</FormLabel>
              <Input
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder={defaultValues.name || "e.g. basic-load-test"}
                bg="surface"
                color="text"
                borderColor="divider"
              />
            </FormControl>

            <FormControl>
              <FormLabel color="text">Description</FormLabel>
              <Input
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder={defaultValues.description || "Brief description of this load test"}
                bg="surface"
                color="text"
                borderColor="divider"
              />
            </FormControl>

            {!selectedRepo && (
              <FormControl isRequired>
                <FormLabel color="text">Load Test Type</FormLabel>
                <Select
                  value={formData.kind}
                  onChange={(e) => setFormData({ ...formData, kind: e.target.value as 'cosmos' | 'eth' })}
                  bg="surface"
                  color="text"
                  borderColor="divider"
                >
                  <option value="cosmos">Cosmos</option>
                  <option value="eth">Ethereum</option>
                </Select>
              </FormControl>
            )}

            <FormControl>
              <FormLabel color="text">Number of Transactions</FormLabel>
              <NumberInput
              value={formData.NumOfTxs || ''}
              min={1}
              onChange={(_, value) => setFormData({ ...formData, NumOfTxs: value })}
              >
                <NumberInputField 
                  bg="surface"
                  color="text"
                  borderColor="divider"
                  placeholder={defaultValues.NumOfTxs?.toString() || "1000"}
                />
              </NumberInput>
            </FormControl>

            {formData.kind === 'cosmos' && (
              <FormControl>
                <FormLabel color="text">Number of Blocks</FormLabel>
                <NumberInput
                  value={formData.NumOfBlocks || ''}
                  min={1}
                  onChange={(_, value) => setFormData({ ...formData, NumOfBlocks: value })}
                >
                  <NumberInputField 
                    bg="surface"
                    color="text"
                    borderColor="divider"
                    placeholder={defaultValues.NumOfBlocks?.toString() || "100"}
                  />
                </NumberInput>
              </FormControl>
            )}

            {formData.kind === 'eth' && (
              <>
                <FormControl>
                  <FormLabel color="text">Send Interval</FormLabel>
                  <Input
                    value={formData.send_interval}
                    onChange={(e) => setFormData({ ...formData, send_interval: e.target.value })}
                    placeholder="1.5s"
                    bg="surface"
                    color="text"
                    borderColor="divider"
                  />
                </FormControl>

                <FormControl>
                  <FormLabel color="text">Number of Batches</FormLabel>
                  <NumberInput
                    value={formData.num_batches || ''}
                    min={1}
                    onChange={(_, value) => setFormData({ ...formData, num_batches: value })}
                  >
                    <NumberInputField 
                      bg="surface"
                      color="text"
                      borderColor="divider"
                      placeholder="15"
                    />
                  </NumberInput>
                </FormControl>
              </>
            )}

            {formData.kind === 'cosmos' && (
              <>
                <FormControl>
                  <FormLabel color="text">Gas Denom</FormLabel>
                  <Input
                    value={formData.gas_denom}
                    onChange={(e) => setFormData({ ...formData, gas_denom: e.target.value })}
                    placeholder="stake"
                    bg="surface"
                    color="text"
                    borderColor="divider"
                  />
                </FormControl>

                <FormControl>
                  <FormLabel color="text">Bech32 Prefix</FormLabel>
                  <Input
                    value={formData.bech32_prefix}
                    onChange={(e) => setFormData({ ...formData, bech32_prefix: e.target.value })}
                    placeholder="cosmos"
                    bg="surface"
                    color="text"
                    borderColor="divider"
                  />
                </FormControl>
              </>
            )}

            <FormControl display="flex" alignItems="center">
              <FormLabel mb="0" color="text">Unordered Transactions</FormLabel>
              <Switch
                isChecked={formData.unordered_txs}
                onChange={(e) => setFormData({ ...formData, unordered_txs: e.target.checked })}
              />
            </FormControl>

            {formData.unordered_txs && (
              <FormControl>
                <FormLabel color="text">Transaction Timeout</FormLabel>
                  <Input
                    value={formData.tx_timeout}
                    onChange={(e) => setFormData({ ...formData, tx_timeout: e.target.value })}
                    placeholder={defaultValues.tx_timeout || "e.g. 30s, 1m"}
                    bg="surface"
                    color="text"
                    borderColor="divider"
                  />
                <FormHelperText color="textSecondary">Only applicable for unordered transactions</FormHelperText>
              </FormControl>
            )}
            
            {formData.kind === 'cosmos' && (
              <>
                <Divider my={2} borderColor="divider" />
                <Text fontWeight="bold" color="text">Message Types</Text>
              </>
            )}

            {formData.kind === 'eth' && (
              <>
                <Divider my={2} borderColor="divider" />
                <Text fontWeight="bold" color="text">Ethereum Message Types</Text>
              </>
            )}
            
            {/* Display Cosmos messages */}
            {formData.kind === 'cosmos' && formData.msgs.map((msg, index) => (
              <HStack key={index} p={2} bg="surface" borderRadius="md" boxShadow="sm">
                <Text flex="1" color="text">{msg.type}</Text>
                <Text color="text">Weight: {msg.weight}</Text>
                {msg.type === MsgType.MsgArr && (
                  <>
                    <Text color="text">Msgs: {msg.NumMsgs}</Text>
                    <Text color="text">Type: {msg.ContainedType}</Text>
                  </>
                )}
                {msg.type === MsgType.MsgMultiSend && msg.NumOfRecipients && (
                  <Text color="text">Recipients: {msg.NumOfRecipients}</Text>
                )}
                <IconButton
                  aria-label="Remove message"
                  size="sm"
                  icon={<DeleteIcon />}
                  onClick={() => removeMessage(index)}
                />
              </HStack>
            ))}
            
            {/* Display Ethereum messages */}
            {formData.kind === 'eth' && formData.msgs.map((msg, index) => (
              <HStack key={index} p={2} bg="surface" borderRadius="md" boxShadow="sm">
                <VStack align="start" flex="1">
                  <Text fontWeight="bold" color="text">{msg.type}</Text>
                  <HStack spacing={4}>
                    <Text fontSize="sm" color="text">Txs: {msg.num_msgs || msg.NumMsgs || 1}</Text>
                  </HStack>
                </VStack>
                <IconButton
                  aria-label="Remove message"
                  size="sm"
                  icon={<DeleteIcon />}
                  onClick={() => removeMessage(index)}
                />
              </HStack>
            ))}
            
            {formData.kind === 'cosmos' && (
            <FormControl>
              <VStack spacing={3} align="stretch">
                <HStack>
                  <FormControl flex="1">
                    <FormLabel fontSize="sm" color="text">Type</FormLabel>
                    <Select
                      value={newMessage.type}
                      onChange={(e) => setNewMessage({ 
                        ...newMessage, 
                        type: e.target.value as MsgType,
                        // Reset conditional fields when changing type
                        NumMsgs: e.target.value === MsgType.MsgArr ? 0 : undefined,
                        ContainedType: e.target.value === MsgType.MsgArr ? undefined : undefined,
                        NumOfRecipients: e.target.value === MsgType.MsgMultiSend ? 1 : undefined
                      })}
                      bg="surface"
                      color="text"
                      borderColor="divider"
                    >
                      <option value={MsgType.MsgSend}>MsgSend</option>
                      <option value={MsgType.MsgMultiSend}>MsgMultiSend</option>
                      <option value={MsgType.MsgArr}>MsgArr</option>
                    </Select>
                  </FormControl>

                  <FormControl flex="1">
                    <FormLabel fontSize="sm" color="text">Weight</FormLabel>
                    <NumberInput
                      value={newMessage.weight}
                      min={0.1}
                      max={1}
                      step={0.01}
                      precision={6}
                      onChange={(_, value) => setNewMessage({ ...newMessage, weight: value })}
                    >
                      <NumberInputField
                        pattern="[0-9]*(.[0-9]+)?"
                        inputMode="decimal"
                        bg="surface"
                        color="text"
                        borderColor="divider"
                      />
                    </NumberInput>
                  </FormControl>
                </HStack>

                {/* Conditional fields based on message type */}
                {newMessage.type === MsgType.MsgArr && (
                  <HStack>
                    <FormControl flex="1">
                      <FormLabel fontSize="sm" color="text">Number of Messages</FormLabel>
                      <NumberInput
                        value={newMessage.NumMsgs || 0}
                        min={1}
                        onChange={(_, value) => setNewMessage({ ...newMessage, NumMsgs: value })}
                      >
                        <NumberInputField 
                          bg="surface"
                          color="text"
                          borderColor="divider"
                        />
                      </NumberInput>
                    </FormControl>

                    <FormControl flex="1">
                      <FormLabel fontSize="sm" color="text">Contained Type</FormLabel>
                      <Select
                        value={newMessage.ContainedType || ''}
                        onChange={(e) => setNewMessage({ 
                          ...newMessage, 
                          ContainedType: e.target.value as MsgType 
                        })}
                        bg="surface"
                        color="text"
                        borderColor="divider"
                      >
                        <option value="">Select type</option>
                        <option value={MsgType.MsgSend}>MsgSend</option>
                        <option value={MsgType.MsgMultiSend}>MsgMultiSend</option>
                      </Select>
                    </FormControl>
                  </HStack>
                )}

                {(newMessage.type === MsgType.MsgMultiSend || 
                  (newMessage.type === MsgType.MsgArr && newMessage.ContainedType === MsgType.MsgMultiSend)) && (
                  <FormControl>
                    <FormLabel fontSize="sm" color="text">Number of Recipients</FormLabel>
                    <NumberInput
                      value={newMessage.NumOfRecipients || 1}
                      min={1}
                      onChange={(_, value) => setNewMessage({ ...newMessage, NumOfRecipients: value })}
                    >
                      <NumberInputField 
                        bg="surface"
                        color="text"
                        borderColor="divider"
                      />
                    </NumberInput>
                  </FormControl>
                )}

                {newMessage.type === MsgType.MsgMultiSend && (
                  <FormControl>
                    <FormLabel fontSize="sm" color="text">Number of Messages</FormLabel>
                    <NumberInput
                      value={newMessage.NumMsgs || 1}
                      min={1}
                      onChange={(_, value) => setNewMessage({ ...newMessage, NumMsgs: value })}
                    >
                      <NumberInputField 
                        bg="surface"
                        color="text"
                        borderColor="divider"
                      />
                    </NumberInput>
                  </FormControl>
                )}

                <Button leftIcon={<AddIcon />} onClick={addMessage} alignSelf="flex-end">
                  Add Message
                </Button>
              </VStack>
            </FormControl>
            )}
            
            {/* Ethereum message configuration */}
            {formData.kind === 'eth' && (
            <FormControl>
              <VStack spacing={3} align="stretch">
                <HStack>
                  <FormControl flex="1">
                    <FormLabel fontSize="sm" color="text">Type</FormLabel>
                    <Select
                      value={newMessage.type}
                      onChange={(e) => setNewMessage({ 
                        ...newMessage, 
                        type: e.target.value as MsgType,
                        // Reset ethereum-specific fields
                        num_msgs: 1,
                      })}
                      bg="surface"
                      color="text"
                      borderColor="divider"
                    >
                      <option value={MsgType.MsgCreateContract}>MsgCreateContract</option>
                      <option value={MsgType.MsgWriteTo}>MsgWriteTo</option>
                      <option value={MsgType.MsgCrossContractCall}>MsgCrossContractCall</option>
                      <option value={MsgType.MsgCallDataBlast}>MsgCallDataBlast</option>
                    </Select>
                    {/* <FormHelperText color="textSecondary">
                      {newMessage.type === MsgType.MsgCreateContract && "Deploys smart contracts to test contract creation gas costs"}
                      {newMessage.type === MsgType.MsgWriteTo && "Makes contracts iterate and write to storage mappings"}
                      {newMessage.type === MsgType.MsgCrossContractCall && "Calls contract methods that invoke other contracts"}
                      {newMessage.type === MsgType.MsgCallDataBlast && "Sends large calldata payloads to test network limits"}
                    </FormHelperText> */}
                  </FormControl>

                  <FormControl flex="1">
                    <FormLabel fontSize="sm" color="text">Number of Transactions</FormLabel>
                    <NumberInput
                      value={newMessage.num_msgs || 1}
                      min={1}
                      onChange={(_, value) => setNewMessage({ ...newMessage, num_msgs: value })}
                    >
                      <NumberInputField
                        bg="surface"
                        color="text"
                        borderColor="divider"
                      />
                    </NumberInput>
                  </FormControl>
                </HStack>

                <Button leftIcon={<AddIcon />} onClick={addMessage} alignSelf="flex-end">
                  Add Message
                </Button>
              </VStack>
            </FormControl>
            )}
            
          </VStack>
        </ModalBody>
        <ModalFooter>
          <Button variant="ghost" mr={3} onClick={onClose}>
            Cancel
          </Button>
          <Button colorScheme="blue" onClick={handleSubmit}>
            Save
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};

export default LoadTestForm;

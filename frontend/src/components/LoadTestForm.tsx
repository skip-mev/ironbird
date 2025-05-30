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
}

const LoadTestForm = ({ isOpen, onClose, initialData, onSave }: LoadTestFormProps) => {
  const [formData, setFormData] = useState<LoadTestSpec>(initialData);
  const [newMessage, setNewMessage] = useState<Message>({
    type: MsgType.MsgSend,
    weight: 0.5,
  });
  const toast = useToast();

  // Reset form data when modal is opened
  useEffect(() => {
    setFormData(initialData);
  }, [initialData, isOpen]);

  const calculateTotalWeight = (): number => {
    if (!Array.isArray(formData.msgs)) {
      return 0; // Or handle as an error, though 0 is safe for toFixed()
    }
    return formData.msgs.reduce((total, msg) => {
      const weight = Number(msg.weight); // Ensure msg.weight is treated as a number
      return total + (isNaN(weight) ? 0 : weight); // Add 0 if weight is NaN
    }, 0);
  };

  const handleSubmit = () => {
    // Validate total weight should be exactly 1.0 if there are any messages
    if (formData.msgs.length > 0) {
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
    // Basic validation
    if (newMessage.weight <= 0 || newMessage.weight > 1) {
      toast({
        title: 'Validation Error',
        description: 'Weight must be between 0 and 1',
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

    // Check for duplicate message type
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

    const updatedMessages = [...formData.msgs, { ...newMessage }];
    setFormData({
      ...formData,
      msgs: updatedMessages,
    });

    // Reset new message form
    setNewMessage({
      type: MsgType.MsgSend,
      weight: 0.5,
    });
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
                placeholder="e.g. basic-load-test"
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
                placeholder="Brief description of this load test"
                bg="surface"
                color="text"
                borderColor="divider"
              />
            </FormControl>

            <FormControl>
              <FormLabel color="text">Number of Transactions</FormLabel>
              <NumberInput
              value={formData.num_of_txs || 0}
              min={1}
              onChange={(_, value) => setFormData({ ...formData, num_of_txs: value, NumOfTxs: value })}
              >
                <NumberInputField 
                  bg="surface"
                  color="text"
                  borderColor="divider"
                />
              </NumberInput>
            </FormControl>

            <FormControl>
              <FormLabel color="text">Number of Blocks</FormLabel>
              <NumberInput
                value={formData.num_of_blocks}
                min={1}
                onChange={(_, value) => setFormData({ ...formData, num_of_blocks: value, NumOfBlocks: value })}
              >
                <NumberInputField 
                  bg="surface"
                  color="text"
                  borderColor="divider"
                />
              </NumberInput>
            </FormControl>

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
                  placeholder="e.g. 30s, 1m"
                  bg="surface"
                  color="text"
                  borderColor="divider"
                />
                <FormHelperText color="textSecondary">Only applicable for unordered transactions</FormHelperText>
              </FormControl>
            )}
            
            <Divider my={2} borderColor="divider" />

            <Text fontWeight="bold" color="text">Message Types</Text>
            
            {formData.msgs.map((msg, index) => (
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

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
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Textarea,
  useToast,
  FormLabel,
} from '@chakra-ui/react';

interface ChainConfigsModalProps {
  isOpen: boolean;
  onClose: () => void;
  initialAppConfig?: any;
  initialConsensusConfig?: any;
  initialClientConfig?: any;
  onSave: (configs: {
    appConfig?: any;
    consensusConfig?: any;
    clientConfig?: any;
  }) => void;
}

const ChainConfigsModal = ({ 
  isOpen, 
  onClose, 
  initialAppConfig, 
  initialConsensusConfig, 
  initialClientConfig, 
  onSave 
}: ChainConfigsModalProps) => {
  const [appConfigJson, setAppConfigJson] = useState('');
  const [consensusConfigJson, setConsensusConfigJson] = useState('');
  const [clientConfigJson, setClientConfigJson] = useState('');
  const toast = useToast();

  // Initialize form data when modal opens
  useEffect(() => {
    if (isOpen) {
      setAppConfigJson(initialAppConfig ? JSON.stringify(initialAppConfig, null, 2) : '');
      setConsensusConfigJson(initialConsensusConfig ? JSON.stringify(initialConsensusConfig, null, 2) : '');
      setClientConfigJson(initialClientConfig ? JSON.stringify(initialClientConfig, null, 2) : '');
    }
  }, [isOpen, initialAppConfig, initialConsensusConfig, initialClientConfig]);

  const handleSave = () => {
    try {
      let appConfig = undefined;
      let consensusConfig = undefined;
      let clientConfig = undefined;

      if (appConfigJson.trim()) {
        appConfig = JSON.parse(appConfigJson);
      }

      if (consensusConfigJson.trim()) {
        consensusConfig = JSON.parse(consensusConfigJson);
      }

      if (clientConfigJson.trim()) {
        clientConfig = JSON.parse(clientConfigJson);
      }

      onSave({
        appConfig,
        consensusConfig,
        clientConfig,
      });

      toast({
        title: 'Configurations saved',
        description: 'Chain configurations have been applied',
        status: 'success',
        duration: 3000,
      });

      onClose();
    } catch (error) {
      toast({
        title: 'JSON Parse Error',
        description: 'Please ensure all configurations contain valid JSON',
        status: 'error',
        duration: 5000,
      });
    }
  };

  const clearAll = () => {
    setAppConfigJson('');
    setConsensusConfigJson('');
    setClientConfigJson('');
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="4xl">
      <ModalOverlay />
      <ModalContent bg="surface" maxH="90vh">
        <ModalHeader color="text">Set Custom Chain Configurations</ModalHeader>
        <ModalCloseButton />
        <ModalBody overflowY="auto">
          <VStack spacing={6} align="stretch">
            <Tabs variant="enclosed" colorScheme="brand">
              <TabList>
                <Tab>App Config</Tab>
                <Tab>Consensus Config</Tab>
                <Tab>Client Config</Tab>
              </TabList>

              <TabPanels>
                {/* App Config Panel */}
                <TabPanel>
                  <VStack align="stretch" spacing={3}>
                    <FormLabel fontSize="sm" color="text">
                      App Configuration (app.toml)
                    </FormLabel>
                    <Textarea
                      placeholder="Enter any custom app configuration you wish to set as JSON..."
                      value={appConfigJson}
                      onChange={(e) => setAppConfigJson(e.target.value)}
                      resize="none"
                      h="400px"
                      bg="surface"
                      color="text"
                      borderColor="divider"
                      fontFamily="mono"
                      fontSize="sm"
                    />
                  </VStack>
                </TabPanel>

                {/* Consensus Config Panel */}
                <TabPanel>
                  <VStack align="stretch" spacing={3}>
                    <FormLabel fontSize="sm" color="text">
                      Consensus Configuration (config.toml)
                    </FormLabel>
                    <Textarea
                      placeholder="Enter any custom consensus configuration you wish to set as JSON..."
                      value={consensusConfigJson}
                      onChange={(e) => setConsensusConfigJson(e.target.value)}
                      resize="none"
                      h="400px"
                      bg="surface"
                      color="text"
                      borderColor="divider"
                      fontFamily="mono"
                      fontSize="sm"
                    />
                  </VStack>
                </TabPanel>

                {/* Client Config Panel */}
                <TabPanel>
                  <VStack align="stretch" spacing={3}>
                    <FormLabel fontSize="sm" color="text">
                      Client Configuration (client.toml)
                    </FormLabel>
                    <Textarea
                      placeholder="Enter any custom client configuration you wish to set as JSON..."
                      value={clientConfigJson}
                      onChange={(e) => setClientConfigJson(e.target.value)}
                      resize="none"
                      h="400px"
                      bg="surface"
                      color="text"
                      borderColor="divider"
                      fontFamily="mono"
                      fontSize="sm"
                    />
                  </VStack>
                </TabPanel>
              </TabPanels>
            </Tabs>
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
              Save Configurations
            </Button>
          </HStack>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};

export default ChainConfigsModal; 
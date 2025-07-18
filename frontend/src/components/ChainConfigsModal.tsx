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
  Text,
  Box,
  IconButton,
  useToast,
  FormLabel,
  Code,
  Flex,
} from '@chakra-ui/react';
import { CopyIcon } from '@chakra-ui/icons';

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

// Sample configurations that users can copy
const generateSampleAppConfig = () => ({
  "minimum-gas-prices": "0.0025stake",
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
  },
  // EVM configuration (remove if not needed)
  "evm": {
    "tracer": "",
    "max-tx-gas-wanted": 0,
    "cache-preimage": false,
    "evm-chain-id": "4231"
  },
  "json-rpc": {
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
  }
});

const generateSampleConsensusConfig = () => ({
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

const generateSampleClientConfig = () => ({
  "chain-id": "test-chain",
  "keyring-backend": "test",
  "output": "text",
  "node": "http://localhost:26657",
  "broadcast-mode": "sync"
});

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

  const copyToClipboard = (text: string, configType: string) => {
    navigator.clipboard.writeText(text).then(() => {
      toast({
        title: 'Copied to clipboard',
        description: `Sample ${configType} configuration copied`,
        status: 'success',
        duration: 2000,
      });
    });
  };

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
    <Modal isOpen={isOpen} onClose={onClose} size="6xl">
      <ModalOverlay />
      <ModalContent bg="surface" maxH="90vh">
        <ModalHeader color="text">Set Chain Configurations</ModalHeader>
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
                  <Flex gap={6} h="400px">
                    <VStack flex="1" align="stretch" spacing={3}>
                      <FormLabel fontSize="sm" color="text">
                        App Configuration (app.toml)
                      </FormLabel>
                      <Textarea
                        placeholder="Enter your app configuration as JSON..."
                        value={appConfigJson}
                        onChange={(e) => setAppConfigJson(e.target.value)}
                        resize="none"
                        h="300px"
                        bg="surface"
                        color="text"
                        borderColor="divider"
                        fontFamily="mono"
                        fontSize="sm"
                      />
                    </VStack>
                    
                    <VStack flex="1" align="stretch" spacing={3}>
                      <HStack justify="space-between">
                        <FormLabel fontSize="sm" color="text">
                          Sample Configuration
                        </FormLabel>
                        <IconButton
                          aria-label="Copy sample config"
                          icon={<CopyIcon />}
                          size="sm"
                          onClick={() => copyToClipboard(JSON.stringify(generateSampleAppConfig(), null, 2), 'App')}
                        />
                      </HStack>
                      <Box
                        h="300px"
                        overflowY="auto"
                        bg="gray.50"
                        borderRadius="md"
                        p={3}
                        border="1px solid"
                        borderColor="divider"
                      >
                        <Code
                          fontSize="xs"
                          whiteSpace="pre"
                          display="block"
                          bg="transparent"
                          color="gray.800"
                        >
                          {JSON.stringify(generateSampleAppConfig(), null, 2)}
                        </Code>
                      </Box>
                    </VStack>
                  </Flex>
                </TabPanel>

                {/* Consensus Config Panel */}
                <TabPanel>
                  <Flex gap={6} h="400px">
                    <VStack flex="1" align="stretch" spacing={3}>
                      <FormLabel fontSize="sm" color="text">
                        Consensus Configuration (config.toml)
                      </FormLabel>
                      <Textarea
                        placeholder="Enter your consensus configuration as JSON..."
                        value={consensusConfigJson}
                        onChange={(e) => setConsensusConfigJson(e.target.value)}
                        resize="none"
                        h="300px"
                        bg="surface"
                        color="text"
                        borderColor="divider"
                        fontFamily="mono"
                        fontSize="sm"
                      />
                    </VStack>
                    
                    <VStack flex="1" align="stretch" spacing={3}>
                      <HStack justify="space-between">
                        <FormLabel fontSize="sm" color="text">
                          Sample Configuration
                        </FormLabel>
                        <IconButton
                          aria-label="Copy sample config"
                          icon={<CopyIcon />}
                          size="sm"
                          onClick={() => copyToClipboard(JSON.stringify(generateSampleConsensusConfig(), null, 2), 'Consensus')}
                        />
                      </HStack>
                      <Box
                        h="300px"
                        overflowY="auto"
                        bg="gray.50"
                        borderRadius="md"
                        p={3}
                        border="1px solid"
                        borderColor="divider"
                      >
                        <Code
                          fontSize="xs"
                          whiteSpace="pre"
                          display="block"
                          bg="transparent"
                          color="gray.800"
                        >
                          {JSON.stringify(generateSampleConsensusConfig(), null, 2)}
                        </Code>
                      </Box>
                    </VStack>
                  </Flex>
                </TabPanel>

                {/* Client Config Panel */}
                <TabPanel>
                  <Flex gap={6} h="400px">
                    <VStack flex="1" align="stretch" spacing={3}>
                      <FormLabel fontSize="sm" color="text">
                        Client Configuration (client.toml)
                      </FormLabel>
                      <Textarea
                        placeholder="Enter your client configuration as JSON..."
                        value={clientConfigJson}
                        onChange={(e) => setClientConfigJson(e.target.value)}
                        resize="none"
                        h="300px"
                        bg="surface"
                        color="text"
                        borderColor="divider"
                        fontFamily="mono"
                        fontSize="sm"
                      />
                    </VStack>
                    
                    <VStack flex="1" align="stretch" spacing={3}>
                      <HStack justify="space-between">
                        <FormLabel fontSize="sm" color="text">
                          Sample Configuration
                        </FormLabel>
                        <IconButton
                          aria-label="Copy sample config"
                          icon={<CopyIcon />}
                          size="sm"
                          onClick={() => copyToClipboard(JSON.stringify(generateSampleClientConfig(), null, 2), 'Client')}
                        />
                      </HStack>
                      <Box
                        h="300px"
                        overflowY="auto"
                        bg="gray.50"
                        borderRadius="md"
                        p={3}
                        border="1px solid"
                        borderColor="divider"
                      >
                        <Code
                          fontSize="xs"
                          whiteSpace="pre"
                          display="block"
                          bg="transparent"
                          color="gray.800"
                        >
                          {JSON.stringify(generateSampleClientConfig(), null, 2)}
                        </Code>
                      </Box>
                    </VStack>
                  </Flex>
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
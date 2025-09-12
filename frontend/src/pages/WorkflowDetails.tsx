import { useParams, useNavigate } from 'react-router-dom';
import { useEffect, useState } from 'react';
import {
  Box,
  Button,
  Heading,
  Text,
  Stack,
  HStack,
  Badge,
  Link,
  useToast,
  Spinner,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Card,
  CardHeader,
  CardBody,
  SimpleGrid,
  Icon,
  ButtonGroup,
  Collapse,
  Flex,
  IconButton,
  VStack
} from '@chakra-ui/react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { workflowApi } from '../api/workflowApi';
import type { LoadTestSpec, WorkflowStatus, WalletInfo } from '../types/workflow';
import { ExternalLinkIcon, CopyIcon, CloseIcon, ChevronDownIcon, ChevronUpIcon, ChevronLeftIcon, ChevronRightIcon } from '@chakra-ui/icons';

const WorkflowDetails = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();

  // State for collapsible cards
  const [isNodesExpanded, setIsNodesExpanded] = useState(true);
  const [isValidatorsExpanded, setIsValidatorsExpanded] = useState(true);
  const [isLoadBalancersExpanded, setIsLoadBalancersExpanded] = useState(true);
  const [isWalletsExpanded, setIsWalletsExpanded] = useState(true);
  
  // State for wallet pagination
  const [currentWalletPage, setCurrentWalletPage] = useState(0);
  const walletsPerPage = 20;

  const { data: workflow, isLoading, error } = useQuery<WorkflowStatus>({
    queryKey: ['workflow', id],
    queryFn: () => workflowApi.getWorkflow(id!),
    refetchInterval: 10000, // Polling every 5 seconds
    enabled: !!id,
  });

  // Log workflow data when it changes
  useEffect(() => {
    if (workflow) {
      // Add more detailed logging for debugging the gray screen issue
      if (workflow.loadTestSpec) {
        try {          
          // Normalize the LoadTestSpec structure to match the expected interface
          // Use type assertion to avoid TypeScript errors
          const loadTestSpec = workflow.loadTestSpec as any;
          
          const normalizedLoadTestSpec: LoadTestSpec = {
            name: loadTestSpec.Name || loadTestSpec.name || "",
            description: loadTestSpec.Description || loadTestSpec.description || "",
            chain_id: loadTestSpec.ChainID || loadTestSpec.chain_id || "",
            kind: loadTestSpec.kind || (loadTestSpec.ChainID || loadTestSpec.chain_id || "").includes('evm') ? 'eth' : 'cosmos',
            NumOfBlocks: loadTestSpec.NumOfBlocks  || 0,
            NumOfTxs: loadTestSpec.NumOfTxs || 0,
            msgs: Array.isArray(loadTestSpec.Msgs) 
              ? loadTestSpec.Msgs 
              : (Array.isArray(loadTestSpec.msgs) 
                ? loadTestSpec.msgs 
                : []),
            unordered_txs: loadTestSpec.unordered_txs || false,
            tx_timeout: loadTestSpec.tx_timeout || "",
          };
          
          // Replace the original LoadTestSpec with the normalized version
          workflow.loadTestSpec = normalizedLoadTestSpec;
        } catch (error) {
          console.error("Error normalizing LoadTestSpec:", error);
        }
      }
    }
  }, [workflow]);

  const cancelWorkflowMutation = useMutation({
    mutationFn: () => {
      if (!workflow) return Promise.reject('No workflow data available');
      return workflowApi.cancelWorkflow(id!);
    },
    onSuccess: () => {
      toast({
        title: 'Workflow canceled',
        description: 'The workflow has been canceled',
        status: 'success',
        duration: 3000,
      });
      // Invalidate the workflow query to refresh the data
      queryClient.invalidateQueries({ queryKey: ['workflow', id] });
    },
    onError: (error) => {
      toast({
        title: 'Error canceling workflow',
        description: error instanceof Error ? error.message : String(error),
        status: 'error',
        duration: 5000,
      });
    },
  });


  // const handleRunLoadTest = () => {
  //   const loadTestSpec: LoadTestSpec = {
  //     name: 'basic-load-test',
  //     description: 'Basic load test configuration',
  //     chain_id: 'test-chain',
  //     NumOfBlocks: 10,
  //     NumOfTxs: 5,
  //     msgs: [],
  //     unordered_txs: true,
  //     tx_timeout: '30s',
  //   };
  //   runLoadTestMutation.mutate(loadTestSpec);
  // };

  const handleCancelWorkflow = () => {
    if (!workflow) return;
    
    const confirmMessage = 'Are you sure you want to cancel this workflow? This will stop processing abruptly.';
      
    if (window.confirm(confirmMessage)) {
      cancelWorkflowMutation.mutate();
    }
  };

  const handleCloneWorkflow = () => {
    if (!workflow) return;
    
    // Create query parameters with workflow data
    const params = new URLSearchParams();
    
    // Use the config field if available
    if (workflow.config) {
      
      // Basic workflow parameters
      if (workflow.config.Repo) params.append('repo', workflow.config.Repo);
      if (workflow.config.SHA) params.append('sha', workflow.config.SHA);
      if (workflow.config.RunnerType) params.append('runnerType', workflow.config.RunnerType);
      
      // IsEvmChain flag - always include it regardless of value
      params.append('isEvmChain', workflow.config.IsEvmChain === true ? 'true' : 'false');
      
      // Long running testnet and duration
      if (workflow.config.LongRunningTestnet !== undefined) {
        params.append('longRunningTestnet', workflow.config.LongRunningTestnet ? 'true' : 'false');
      }
      
      // Launch load balancer flag
      if (workflow.config.LaunchLoadBalancer !== undefined) {
        params.append('launchLoadBalancer', workflow.config.LaunchLoadBalancer ? 'true' : 'false');
      }
      
      if (workflow.config.TestnetDuration) {
        params.append('testnetDuration', workflow.config.TestnetDuration);
      }
      
      // Number of wallets
      if (workflow.config.NumWallets) {
        params.append('numWallets', workflow.config.NumWallets.toString());
      }
      
      // Chain config
      if (workflow.config.ChainConfig) {
        console.log("ChainConfig exists:", workflow.config.ChainConfig);
        console.log("ChainConfig.Name:", workflow.config.ChainConfig.Name);
        
        if (workflow.config.ChainConfig.Name) {
          console.log("Adding chainName parameter:", workflow.config.ChainConfig.Name);
          params.append('chainName', workflow.config.ChainConfig.Name);
        } else {
          console.log("ChainConfig.Name is empty or undefined");
        }
        
        if (workflow.config.ChainConfig.Image) {
          params.append('image', workflow.config.ChainConfig.Image);
        }
        
        // Handle NumOfNodes and NumOfValidators for Docker deployments
        if (workflow.config.ChainConfig.NumOfNodes !== undefined) {
          params.append('numOfNodes', workflow.config.ChainConfig.NumOfNodes.toString());
        }
        
        if (workflow.config.ChainConfig.NumOfValidators !== undefined) {
          params.append('numOfValidators', workflow.config.ChainConfig.NumOfValidators.toString());
        }
        
        // Handle regional configurations for DigitalOcean deployments
        if (workflow.config.ChainConfig.RegionConfigs && 
            workflow.config.ChainConfig.RegionConfigs.length > 0) {
          params.append('regionConfigs', JSON.stringify(workflow.config.ChainConfig.RegionConfigs));
        }

        // Genesis modifications
        if (workflow.config.ChainConfig.GenesisModifications && 
            workflow.config.ChainConfig.GenesisModifications.length > 0) {
          params.append('genesisModifications', 
            JSON.stringify(workflow.config.ChainConfig.GenesisModifications));
        }
        
        // Custom chain configurations
        if (workflow.config.ChainConfig.AppConfig) {
          params.append('appConfig', JSON.stringify(workflow.config.ChainConfig.AppConfig));
        }
        
        if (workflow.config.ChainConfig.ConsensusConfig) {
          params.append('consensusConfig', JSON.stringify(workflow.config.ChainConfig.ConsensusConfig));
        }
        
        if (workflow.config.ChainConfig.ClientConfig) {
          params.append('clientConfig', JSON.stringify(workflow.config.ChainConfig.ClientConfig));
        }
        
        if (workflow.config.ChainConfig.SetSeedNode !== undefined) {
          params.append('setSeedNode', workflow.config.ChainConfig.SetSeedNode.toString());
        }
        
        if (workflow.config.ChainConfig.SetPersistentPeers !== undefined) {
          params.append('setPersistentPeers', workflow.config.ChainConfig.SetPersistentPeers.toString());
        }
        
        // Handle simapp version for cometbft repos
        if (workflow.config.Repo === 'cometbft') {
          // Version field contains the simapp version (both predefined and custom)
          if (workflow.config.ChainConfig.Version !== undefined && workflow.config.ChainConfig.Version !== '') {
            params.append('version', workflow.config.ChainConfig.Version);
          }
        }
      }
    }
    
    // LoadTestSpec - use EncodedLoadTestSpec (YAML format) as that's what's actually available
    if (workflow.config?.EncodedLoadTestSpec) {
      params.append('encodedLoadTestSpec', workflow.config.EncodedLoadTestSpec);
    }
        
    // Navigate to root path (which is the create workflow page) with parameters
    navigate(`/?${params.toString()}`);
    
    toast({
      title: 'Workflow configuration copied',
      description: 'You can now create a new workflow with the same configuration',
      status: 'success',
      duration: 3000,
    });
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running':
        return 'green';
      case 'completed':
        return 'blue';
      case 'failed':
        return 'red';
      case 'canceled':
        return 'orange';
      case 'terminated':
        return 'red';
      default:
        return 'gray';
    }
  };

  // Helper function to update monitoring dashboard URLs with proper time ranges and provider
  const updateMonitoringUrl = (originalUrl: string, startTime?: string, endTime?: string, provider?: string) => {
    if (!startTime) return originalUrl;
    
    const url = new URL(originalUrl);
    const startTimestamp = new Date(startTime).getTime();
    
    // Update the from parameter
    url.searchParams.set('from', startTimestamp.toString());
    
    // Update the to parameter
    if (endTime) {
      const endTimestamp = new Date(endTime).getTime();
      url.searchParams.set('to', endTimestamp.toString());
    } else {
      url.searchParams.set('to', 'now');
    }
    
    // Add provider variable if provided and not already present
    if (provider && !url.searchParams.has('var-provider')) {
      url.searchParams.set('var-provider', provider);
    }
    
    return url.toString();
  };

  // Export wallets to CSV function
  const exportWalletsToCSV = (wallets: WalletInfo) => {
    const csvContent = [
      ['Type', 'Address', 'Mnemonic'],
      ['Faucet', wallets.faucetAddress, wallets.faucetMnemonic],
      ...wallets.userAddresses.map((address, index) => [
        'User',
        address,
        wallets.userMnemonics[index] || ''
      ])
    ].map(row => row.map(field => `"${field}"`).join(',')).join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', `workflow-${id}-wallets.csv`);
    link.style.visibility = 'hidden';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    
    toast({
      title: 'CSV exported successfully',
      description: `Exported ${wallets.userAddresses.length + 1} wallets`,
      status: 'success',
      duration: 3000,
    });
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" height="400px">
        <Stack align="center" spacing={4}>
          <Spinner size="xl" color="blue.500" />
          <Text fontSize="lg">Loading workflow details...</Text>
        </Stack>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert status="error">
        <AlertIcon />
        <Box>
          <AlertTitle>Error loading workflow!</AlertTitle>
          <AlertDescription>
            {error instanceof Error ? error.message : 'Failed to load workflow details'}
          </AlertDescription>
        </Box>
      </Alert>
    );
  }

  if (!workflow) {
    return (
      <Alert status="warning">
        <AlertIcon />
        <Box>
          <AlertTitle>Workflow not found!</AlertTitle>
          <AlertDescription>
            The workflow with ID "{id}" could not be found.
          </AlertDescription>
        </Box>
      </Alert>
    );
  }

  // Wrap the entire component rendering in a try-catch to prevent white screen errors
  try {
    return (
      <Box maxW="1200px" mx="auto">
        <Heading mb={6} size="lg">Workflow Details</Heading>
      
      <Stack spacing={6}>
        {/* Basic Info Card */}
        <Card>
          <CardHeader>
            <Heading size="md">Basic Information</Heading>
          </CardHeader>
          <CardBody>
            <SimpleGrid columns={{ base: 1, md: 2 }} spacing={4}>
              <Box>
                <Text fontWeight="bold" color="gray.600" fontSize="sm">
                  Workflow ID
                </Text>
                <Text fontFamily="mono" fontSize="sm" wordBreak="break-all">
                  {workflow.WorkflowID}
                </Text>
              </Box>
              <Box>
                <Text fontWeight="bold" color="gray.600" fontSize="sm">
                  Provider
                </Text>
                <Badge colorScheme="blue" variant="subtle" size="lg">
                  {workflow.Provider || 'Unknown'}
                </Badge>
              </Box>
              <Box>
                <Text fontWeight="bold" color="gray.600" fontSize="sm">
                  Status
                </Text>
                <Badge 
                  colorScheme={getStatusColor(workflow.Status)} 
                  variant="solid"
                  size="lg"
                  textTransform="capitalize"
                >
                  {workflow.Status}
                </Badge>
              </Box>
              <Box>
                <Text fontWeight="bold" color="gray.600" fontSize="sm">
                  Temporal Workflow
                </Text>
                <Link 
                  href={(() => {
                    const grpcAddress = import.meta.env.VITE_IRONBIRD_GRPC_ADDRESS;
                    
                    if (!grpcAddress) {
                      // Fallback to localhost when env var is not set
                      return `http://localhost:8233/namespaces/default/workflows/${workflow.WorkflowID}`;
                    }
                    
                    if (grpcAddress.includes('prod')) {
                      return `https://ironbird-temporal.prod.skip-internal.money/namespaces/ironbird/workflows/${workflow.WorkflowID}`;
                    } else if (grpcAddress.includes('dev')) {
                      return `https://ironbird-temporal.dev.skip-internal.money/namespaces/ironbird/workflows/${workflow.WorkflowID}`;
                    } else {
                      // Default fallback to localhost
                      return `http://localhost:8233/namespaces/default/workflows/${workflow.WorkflowID}`;
                    }
                  })()} 
                  target="_blank" 
                  color="blue.500"
                  display="flex"
                  alignItems="center"
                  gap={1}
                  fontSize="sm"
                >
                  View in Temporal
                  <Icon as={ExternalLinkIcon} boxSize={3} />
                </Link>
              </Box>
            </SimpleGrid>
          </CardBody>
        </Card>

        {/* Testnet Setup Card - shown when testnet is still being set up */}
        {workflow.Status === 'running' && 
          (!workflow.Nodes || workflow.Nodes.length === 0) && 
          (!workflow.Validators || workflow.Validators.length === 0) ? (
          <Card>
            <CardHeader>
              <Heading size="md">Testnet Setup</Heading>
            </CardHeader>
            <CardBody>
              <Box 
                display="flex" 
                flexDirection="column" 
                alignItems="center" 
                justifyContent="center" 
                py={10}
                bg="surface"
                borderRadius="md"
                boxShadow="sm"
              >
                <Spinner size="xl" color="brand.500" thickness="3px" mb={4} />
                <Text fontSize="lg" fontWeight="medium" color="text" mb={2}>
                  Testnet Spinup in Progress
                </Text>
                <Text color="textSecondary" textAlign="center" maxW="md">
                  The testnet is being created. This process may take a few minutes.
                  Nodes, validators, and load balancers will appear here when they are ready.
                  The page will automatically update when components are ready.
                </Text>
              </Box>
            </CardBody>
          </Card>
        ) : (
          <>
            {/* Nodes Card - only shown when nodes are ready */}
            {workflow.Nodes && workflow.Nodes.length > 0 && (
              <Card>
                <CardHeader 
                  cursor="pointer"
                  onClick={() => setIsNodesExpanded(!isNodesExpanded)}
                  _hover={{ bg: { base: "gray.50", _dark: "gray.700" } }}
                >
                  <Flex justify="space-between" align="center">
                    <Heading size="md">Full Nodes {`(${workflow.Nodes.length})`}</Heading>
                    <Icon as={isNodesExpanded ? ChevronUpIcon : ChevronDownIcon} boxSize={5} />
                  </Flex>
                </CardHeader>
                <Collapse in={isNodesExpanded}>
                  <CardBody>
                    {workflow.config?.RunnerType === 'DigitalOcean' ? (
                      // Regional view for DigitalOcean
                      (() => {
                        // Group nodes by region (extract region from node name)
                        const nodesByRegion = workflow.Nodes.reduce((acc, node) => {
                          const regionMatch = node.Name.match(/-([a-z]{3}\d)$/);
                          const region = regionMatch ? regionMatch[1] : 'unknown';
                          if (!acc[region]) acc[region] = [];
                          acc[region].push(node);
                          return acc;
                        }, {} as Record<string, typeof workflow.Nodes>);

                        const regionLabels = {
                          'nyc1': 'New York (nyc1)',
                          'sfo2': 'San Francisco (sfo2)',
                          'ams3': 'Amsterdam (ams3)',
                          'fra1': 'Frankfurt (fra1)',
                          'sgp1': 'Singapore (sgp1)',
                        };

                        return (
                          <VStack spacing={6} align="stretch">
                            {Object.entries(nodesByRegion).map(([region, nodes]) => (
                              <Box key={region}>
                                <Text fontWeight="bold" fontSize="lg" mb={3} color="purple.600">
                                  {regionLabels[region as keyof typeof regionLabels] || region} - {nodes.length} node{nodes.length !== 1 ? 's' : ''}
                                </Text>
                                <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={4}>
                                  {nodes.map((node) => (
                                    <Box 
                                      key={node.Name} 
                                      bg="surface" 
                                      p={4} 
                                      borderRadius="md" 
                                      border="1px"
                                      borderColor="divider"
                                    >
                                      <Text fontWeight="bold" fontSize="lg" mb={3} color="blue.600">
                                        {node.Name}
                                      </Text>
                                      <Stack spacing={2}>
                                        <HStack>
                                          <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                            RPC:
                                          </Text>
                                          <Link 
                                            href={node.RPC} 
                                            target="_blank" 
                                            color="blue.500"
                                            fontSize="sm"
                                            fontFamily="mono"
                                            textDecoration="underline"
                                            display="flex"
                                            alignItems="center"
                                            gap={1}
                                          >
                                            {node.RPC}
                                            <Icon as={ExternalLinkIcon} boxSize={3} />
                                          </Link>
                                        </HStack>
                                        <HStack>
                                          <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                            LCD:
                                          </Text>
                                          <Text 
                                            color="blue.500"
                                            fontSize="sm"
                                            fontFamily="mono"
                                            textDecoration="underline"
                                            cursor="default"
                                          >
                                            {node.LCD}
                                          </Text>
                                        </HStack>
                                        {node.GRPC && (
                                          <HStack>
                                            <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                              gRPC:
                                            </Text>
                                            <Text 
                                              fontFamily="mono"
                                              fontSize="sm"
                                              color="blue.500"
                                              textDecoration="underline"
                                              cursor="default"
                                            >
                                              {node.GRPC}
                                            </Text>
                                          </HStack>
                                        )}
                                      </Stack>
                                    </Box>
                                  ))}
                                </SimpleGrid>
                              </Box>
                            ))}
                          </VStack>
                        );
                      })()
                    ) : (
                      // Original flat view for Docker
                      <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={4}>
                        {workflow.Nodes.map((node) => (
                          <Box 
                            key={node.Name} 
                            bg="surface" 
                            p={4} 
                            borderRadius="md" 
                            border="1px"
                            borderColor="divider"
                          >
                            <Text fontWeight="bold" fontSize="lg" mb={3} color="blue.600">
                              {node.Name}
                            </Text>
                            <Stack spacing={2}>
                              <HStack>
                                <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                  RPC:
                                </Text>
                                <Link 
                                  href={node.RPC} 
                                  target="_blank" 
                                  color="blue.500"
                                  fontSize="sm"
                                  fontFamily="mono"
                                  textDecoration="underline"
                                  display="flex"
                                  alignItems="center"
                                  gap={1}
                                >
                                  {node.RPC}
                                  <Icon as={ExternalLinkIcon} boxSize={3} />
                                </Link>
                              </HStack>
                              <HStack>
                                <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                  LCD:
                                </Text>
                                <Text 
                                  color="blue.500"
                                  fontSize="sm"
                                  fontFamily="mono"
                                  textDecoration="underline"
                                  cursor="default"
                                >
                                  {node.LCD}
                                </Text>
                              </HStack>
                              {node.GRPC && (
                                <HStack>
                                  <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                    gRPC:
                                  </Text>
                                  <Text 
                                    fontFamily="mono"
                                    fontSize="sm"
                                    color="blue.500"
                                    textDecoration="underline"
                                    cursor="default"
                                  >
                                    {node.GRPC}
                                  </Text>
                                </HStack>
                              )}
                            </Stack>
                          </Box>
                        ))}
                      </SimpleGrid>
                    )}
                  </CardBody>
                </Collapse>
              </Card>
            )}

            {/* Validators Card - only shown when validators are ready */}
            {workflow.Validators && workflow.Validators.length > 0 && (
              <Card>
                <CardHeader 
                  cursor="pointer"
                  onClick={() => setIsValidatorsExpanded(!isValidatorsExpanded)}
                  _hover={{ bg: { base: "gray.50", _dark: "gray.700" } }}
                >
                  <Flex justify="space-between" align="center">
                    <Heading size="md">Validators {`(${workflow.Validators.length})`}</Heading>
                    <Icon as={isValidatorsExpanded ? ChevronUpIcon : ChevronDownIcon} boxSize={5} />
                  </Flex>
                </CardHeader>
                <Collapse in={isValidatorsExpanded}>
                  <CardBody>
                    {workflow.config?.RunnerType === 'DigitalOcean' ? (
                      // Regional view for DigitalOcean
                      (() => {
                        // Group validators by region (extract region from validator name)
                        const validatorsByRegion = workflow.Validators.reduce((acc, validator) => {
                          const regionMatch = validator.Name.match(/-([a-z]{3}\d)$/);
                          const region = regionMatch ? regionMatch[1] : 'unknown';
                          if (!acc[region]) acc[region] = [];
                          acc[region].push(validator);
                          return acc;
                        }, {} as Record<string, typeof workflow.Validators>);

                        const regionLabels = {
                          'nyc1': 'New York (nyc1)',
                          'sfo2': 'San Francisco (sfo2)',
                          'ams3': 'Amsterdam (ams3)',
                          'fra1': 'Frankfurt (fra1)',
                          'sgp1': 'Singapore (sgp1)',
                        };

                        return (
                          <VStack spacing={6} align="stretch">
                            {Object.entries(validatorsByRegion).map(([region, validators]) => (
                              <Box key={region}>
                                <Text fontWeight="bold" fontSize="lg" mb={3} color="purple.600">
                                  {regionLabels[region as keyof typeof regionLabels] || region} - {validators.length} validator{validators.length !== 1 ? 's' : ''}
                                </Text>
                                <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={4}>
                                  {validators.map((validator) => (
                                    <Box 
                                      key={validator.Name} 
                                      bg="surface" 
                                      p={4} 
                                      borderRadius="md" 
                                      border="1px"
                                      borderColor="divider"
                                    >
                                      <Text fontWeight="bold" fontSize="lg" mb={3} color="blue.600">
                                        {validator.Name}
                                      </Text>
                                      <Stack spacing={2}>
                                        <HStack>
                                          <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                            RPC:
                                          </Text>
                                          <Link 
                                            href={validator.RPC} 
                                            target="_blank" 
                                            color="blue.500"
                                            fontSize="sm"
                                            fontFamily="mono"
                                            textDecoration="underline"
                                            display="flex"
                                            alignItems="center"
                                            gap={1}
                                          >
                                            {validator.RPC}
                                            <Icon as={ExternalLinkIcon} boxSize={3} />
                                          </Link>
                                        </HStack>
                                        <HStack>
                                          <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                            LCD:
                                          </Text>
                                          <Text 
                                            color="blue.500"
                                            fontSize="sm"
                                            fontFamily="mono"
                                            textDecoration="underline"
                                            cursor="default"
                                          >
                                            {validator.LCD}
                                          </Text>
                                        </HStack>
                                        {validator.GRPC && (
                                          <HStack>
                                            <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                              gRPC:
                                            </Text>
                                            <Text 
                                              fontFamily="mono"
                                              fontSize="sm"
                                              color="blue.500"
                                              textDecoration="underline"
                                              cursor="default"
                                            >
                                              {validator.GRPC}
                                            </Text>
                                          </HStack>
                                        )}
                                      </Stack>
                                    </Box>
                                  ))}
                                </SimpleGrid>
                              </Box>
                            ))}
                          </VStack>
                        );
                      })()
                    ) : (
                      // Original flat view for Docker
                      <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={4}>
                        {workflow.Validators.map((validator) => (
                          <Box 
                            key={validator.Name} 
                            bg="surface" 
                            p={4} 
                            borderRadius="md" 
                            border="1px"
                            borderColor="divider"
                          >
                            <Text fontWeight="bold" fontSize="lg" mb={3} color="blue.600">
                              {validator.Name}
                            </Text>
                            <Stack spacing={2}>
                              <HStack>
                                <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                  RPC:
                                </Text>
                                <Link 
                                  href={validator.RPC} 
                                  target="_blank" 
                                  color="blue.500"
                                  fontSize="sm"
                                  fontFamily="mono"
                                  textDecoration="underline"
                                  display="flex"
                                  alignItems="center"
                                  gap={1}
                                >
                                  {validator.RPC}
                                  <Icon as={ExternalLinkIcon} boxSize={3} />
                                </Link>
                              </HStack>
                              <HStack>
                                <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                  LCD:
                                </Text>
                                <Text 
                                  color="blue.500"
                                  fontSize="sm"
                                  fontFamily="mono"
                                  textDecoration="underline"
                                  cursor="default"
                                >
                                  {validator.LCD}
                                </Text>
                              </HStack>
                              {validator.GRPC && (
                                <HStack>
                                  <Text fontWeight="semibold" minW="60px" fontSize="sm">
                                    gRPC:
                                  </Text>
                                  <Text 
                                    fontFamily="mono"
                                    fontSize="sm"
                                    color="blue.500"
                                    textDecoration="underline"
                                    cursor="default"
                                  >
                                    {validator.GRPC}
                                  </Text>
                                </HStack>
                              )}
                            </Stack>
                          </Box>
                        ))}
                      </SimpleGrid>
                    )}
                  </CardBody>
                </Collapse>
              </Card>
            )}
          </>
        )}

        {/* Load Balancers Card */}
        {workflow.LoadBalancers && workflow.LoadBalancers.length > 0 && (
          <Card>
            <CardHeader 
              cursor="pointer"
              onClick={() => setIsLoadBalancersExpanded(!isLoadBalancersExpanded)}
              _hover={{ bg: { base: "gray.50", _dark: "gray.700" } }}
            >
              <Flex justify="space-between" align="center">
                <Heading size="md">Load Balancers {`(${workflow.LoadBalancers.length})`}</Heading>
                <Icon as={isLoadBalancersExpanded ? ChevronUpIcon : ChevronDownIcon} boxSize={5} />
              </Flex>
            </CardHeader>
            <Collapse in={isLoadBalancersExpanded}>
              <CardBody>
                <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={4}>
                  {workflow.LoadBalancers.map((node) => (
                    <Box 
                      key={node.Name} 
                      bg="surface" 
                      p={4} 
                      borderRadius="md" 
                      border="1px"
                      borderColor="divider"
                    >
                      <Text fontWeight="bold" fontSize="lg" mb={3} color="blue.600">
                        {node.Name}
                      </Text>
                      <Stack spacing={2}>
                        <HStack>
                          <Text fontWeight="semibold" minW="60px" fontSize="sm">
                            RPC:
                          </Text>
                          <Link 
                            href={node.RPC} 
                            target="_blank" 
                            color="blue.500"
                            fontSize="sm"
                            fontFamily="mono"
                            textDecoration="underline"
                            display="flex"
                            alignItems="center"
                            gap={1}
                          >
                            {node.RPC}
                            <Icon as={ExternalLinkIcon} boxSize={3} />
                          </Link>
                        </HStack>
                        <HStack>
                          <Text fontWeight="semibold" minW="60px" fontSize="sm">
                            LCD:
                          </Text>
                          <Text 
                            color="blue.500"
                            fontSize="sm"
                            fontFamily="mono"
                            textDecoration="underline"
                            cursor="default"
                          >
                            {node.LCD}
                          </Text>
                        </HStack>
                        {node.GRPC && (
                          <HStack>
                            <Text fontWeight="semibold" minW="60px" fontSize="sm">
                              gRPC:
                            </Text>
                            <Text 
                              fontFamily="mono"
                              fontSize="sm"
                              color="blue.500"
                              textDecoration="underline"
                              cursor="default"
                            >
                              {node.GRPC}
                            </Text>
                          </HStack>
                        )}
                      </Stack>
                    </Box>
                  ))}
                </SimpleGrid>
              </CardBody>
            </Collapse>
          </Card>
        )}

        {/* Monitoring Card */}
        {workflow.Monitoring && Object.keys(workflow.Monitoring).length > 0 && workflow.config?.RunnerType !== 'Docker' && (
          <Card>
            <CardHeader>
              <Heading size="md">Monitoring Dashboards</Heading>
            </CardHeader>
            <CardBody>
              <Stack spacing={4}>
                <Box>
                  <Text fontWeight="bold" color="gray.600" fontSize="md" mb={3}>
                    Monitoring Dashboards
                  </Text>
                  <Stack spacing={3}>
                    {Object.entries(workflow.Monitoring).map(([name, url]) => {
                      const updatedUrl = updateMonitoringUrl(url, workflow.StartTime, workflow.EndTime, workflow.Provider);
                      return (
                        <HStack key={name} spacing={3}>
                          <Text fontWeight="semibold" minW="100px" textTransform="capitalize">
                            {name}
                          </Text>
                          <Link 
                            href={updatedUrl} 
                            target="_blank" 
                            color="blue.500"
                            display="flex"
                            alignItems="center"
                            gap={2}
                          >
                            {updatedUrl}
                            <Icon as={ExternalLinkIcon} boxSize={4} />
                          </Link>
                        </HStack>
                      );
                    })}
                  </Stack>
                </Box>

                {/* Profiling Data Section */}
                {workflow.config?.RunnerType !== 'Docker' && (
                  <Box>
                    <Text fontWeight="bold" color="gray.600" fontSize="md" mb={3}>
                      Profiling Data
                    </Text>
                    <HStack spacing={3}>
                      <Text fontWeight="semibold" minW="100px">
                        Pyroscope
                      </Text>
                      <Link 
                        href={(() => {
                          // Calculate time range based on workflow timing
                          const baseUrl = 'https://skipprotocol.grafana.net/a/grafana-pyroscope-app/explore?searchText=&panelType=time-series&layout=grid&hideNoData=off&explorationType=flame-graph&var-serviceName=ironbird&var-profileMetricId=goroutine:goroutine:count:goroutine:count&var-spanSelector=undefined&var-dataSource=grafanacloud-profiles&var-filters=provider%7C%3D%7C';
                          const provider = workflow.Provider || 'unknown';
                          
                          let fromParam, toParam;
                          if (workflow.StartTime) {
                            // Convert workflow start time to Unix timestamp in milliseconds for Grafana
                            const startTimestamp = new Date(workflow.StartTime).getTime();
                            fromParam = `from=${startTimestamp}`;
                          } else {
                            fromParam = 'from=now-5m';
                          }
                          
                          if (workflow.EndTime) {
                            // Convert workflow end time to Unix timestamp in milliseconds for Grafana
                            const endTimestamp = new Date(workflow.EndTime).getTime();
                            toParam = `to=${endTimestamp}`;
                          } else {
                            // If workflow is still running, use "now"
                            toParam = 'to=now';
                          }
                          
                          return `${baseUrl}${provider}&var-filtersBaseline=&var-filtersComparison=&var-groupBy=&${fromParam}&${toParam}&maxNodes=16384&diffFrom=&diffTo=&diffFrom-2=&diffTo-2=`;
                        })()}
                        target="_blank" 
                        color="blue.500"
                        display="flex"
                        alignItems="center"
                        gap={2}
                      >
                        View Profiling Data
                        <Icon as={ExternalLinkIcon} boxSize={4} />
                      </Link>
                    </HStack>
                  </Box>
                )}
              </Stack>
            </CardBody>
          </Card>
        )}

        {/* Wallets Card */}
        {workflow.wallets && (workflow.wallets.faucetAddress || workflow.wallets.userAddresses?.length > 0) && (
          <Card>
            <CardHeader 
              cursor="pointer"
              onClick={() => setIsWalletsExpanded(!isWalletsExpanded)}
              _hover={{ bg: { base: "gray.50", _dark: "gray.700" } }}
            >
              <Flex justify="space-between" align="center">
                <Heading size="md">Wallets</Heading>
                <Icon as={isWalletsExpanded ? ChevronUpIcon : ChevronDownIcon} boxSize={5} />
              </Flex>
            </CardHeader>
            <Collapse in={isWalletsExpanded}>
              <CardBody>
                <Stack spacing={4}>
                  {/* Faucet Wallet */}
                  {workflow.wallets.faucetAddress && (
                    <Box>
                      <Text fontWeight="bold" color="purple.600" fontSize="lg" mb={2}>
                        Faucet Wallet
                      </Text>
                      <Box bg="surface" p={3} borderRadius="md" border="1px" borderColor="divider">
                        <HStack>
                          <Text fontWeight="semibold" minW="80px" fontSize="sm">
                            Address:
                          </Text>
                          <Text fontFamily="mono" fontSize="sm" wordBreak="break-all" flex="1">
                            {workflow.wallets.faucetAddress}
                          </Text>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => {
                              if (workflow.wallets?.faucetAddress) {
                                navigator.clipboard.writeText(workflow.wallets.faucetAddress);
                                toast({
                                  title: 'Copied to clipboard',
                                  status: 'success',
                                  duration: 2000,
                                });
                              }
                            }}
                          >
                            <CopyIcon />
                          </Button>
                        </HStack>
                      </Box>
                    </Box>
                  )}

                  {/* User Wallets */}
                  {workflow.wallets.userAddresses && workflow.wallets.userAddresses.length > 0 && (() => {
                    const totalWallets = workflow.wallets.userAddresses.length;
                    const totalPages = Math.ceil(totalWallets / walletsPerPage);
                    const startIndex = currentWalletPage * walletsPerPage;
                    const endIndex = Math.min(startIndex + walletsPerPage, totalWallets);
                    const currentPageWallets = workflow.wallets.userAddresses.slice(startIndex, endIndex);

                    return (
                      <Box>
                        <HStack justify="space-between" align="center" mb={3}>
                          <Text fontWeight="bold" color="blue.600" fontSize="lg">
                            User Wallets ({totalWallets.toLocaleString()})
                          </Text>
                          <Button
                            size="sm"
                            colorScheme="green"
                            onClick={() => workflow.wallets && exportWalletsToCSV(workflow.wallets)}
                          >
                            Export CSV
                          </Button>
                        </HStack>
                        
                        {/* Pagination Controls */}
                        {totalPages > 1 && (
                          <HStack justify="space-between" align="center" mb={3}>
                            <HStack>
                              <IconButton
                                aria-label="Previous page"
                                icon={<ChevronLeftIcon />}
                                size="sm"
                                isDisabled={currentWalletPage === 0}
                                onClick={() => setCurrentWalletPage(prev => Math.max(0, prev - 1))}
                              />
                              <Text fontSize="sm">
                                Page {currentWalletPage + 1} of {totalPages}
                              </Text>
                              <IconButton
                                aria-label="Next page"
                                icon={<ChevronRightIcon />}
                                size="sm"
                                isDisabled={currentWalletPage === totalPages - 1}
                                onClick={() => setCurrentWalletPage(prev => Math.min(totalPages - 1, prev + 1))}
                              />
                            </HStack>
                            <Text fontSize="sm" color="gray.500">
                              Showing {startIndex + 1}-{endIndex} of {totalWallets.toLocaleString()}
                            </Text>
                          </HStack>
                        )}

                        <Box 
                          bg="surface" 
                          p={3} 
                          borderRadius="md" 
                          border="1px" 
                          borderColor="divider"
                        >
                          <VStack spacing={2} align="stretch">
                            {currentPageWallets.map((address, index) => {
                              const globalIndex = startIndex + index;
                              return (
                                <HStack key={globalIndex} spacing={2}>
                                  <Text fontSize="xs" color="gray.500" minW="50px">
                                    {(globalIndex + 1).toLocaleString()}.
                                  </Text>
                                  <Text fontFamily="mono" fontSize="sm" flex="1" wordBreak="break-all">
                                    {address}
                                  </Text>
                                  <Button
                                    size="xs"
                                    variant="ghost"
                                    onClick={() => {
                                      navigator.clipboard.writeText(address);
                                      toast({
                                        title: 'Address copied',
                                        status: 'success',
                                        duration: 2000,
                                      });
                                    }}
                                  >
                                    <CopyIcon boxSize={3} />
                                  </Button>
                                </HStack>
                              );
                            })}
                          </VStack>
                        </Box>
                        
                        {/* Bottom Pagination Controls */}
                        {totalPages > 1 && (
                          <HStack justify="center" mt={3}>
                            <IconButton
                              aria-label="First page"
                              icon={<ChevronLeftIcon />}
                              size="sm"
                              variant="outline"
                              isDisabled={currentWalletPage === 0}
                              onClick={() => setCurrentWalletPage(0)}
                            />
                            <IconButton
                              aria-label="Previous page"
                              icon={<ChevronLeftIcon />}
                              size="sm"
                              isDisabled={currentWalletPage === 0}
                              onClick={() => setCurrentWalletPage(prev => Math.max(0, prev - 1))}
                            />
                            <Text fontSize="sm" px={4}>
                              {currentWalletPage + 1} / {totalPages}
                            </Text>
                            <IconButton
                              aria-label="Next page"
                              icon={<ChevronRightIcon />}
                              size="sm"
                              isDisabled={currentWalletPage === totalPages - 1}
                              onClick={() => setCurrentWalletPage(prev => Math.min(totalPages - 1, prev + 1))}
                            />
                            <IconButton
                              aria-label="Last page"
                              icon={<ChevronRightIcon />}
                              size="sm"
                              variant="outline"
                              isDisabled={currentWalletPage === totalPages - 1}
                              onClick={() => setCurrentWalletPage(totalPages - 1)}
                            />
                          </HStack>
                        )}
                      </Box>
                    );
                  })()}
                </Stack>
              </CardBody>
            </Collapse>
          </Card>
        )}

        {/* Actions Card */}
        <Card>
          <CardHeader>
            <Heading size="md">Actions</Heading>
          </CardHeader>
          <CardBody>
            <Stack spacing={4}>
              {/* Action Buttons */}
              <ButtonGroup spacing={4}>
                <Button
                  leftIcon={<CopyIcon />}
                  colorScheme="purple"
                  onClick={handleCloneWorkflow}
                  size="lg"
                >
                  Clone Workflow
                </Button>
                <Button
                  leftIcon={<CloseIcon />}
                  colorScheme="orange"
                  onClick={handleCancelWorkflow}
                  isLoading={cancelWorkflowMutation.isPending}
                  loadingText="Canceling Workflow..."
                  disabled={workflow.Status !== 'running'}
                  size="lg"
                >
                  Cancel Workflow
                </Button>
              </ButtonGroup>
              
              {/* Action Explanations */}
              <Box bg={{ base: "gray.50", _dark: "gray.700" }} p={4} borderRadius="md" border="1px" borderColor="divider">
                <Stack spacing={3}>
                  <Box>
                    <Text fontWeight="semibold" color="orange.600" mb={1}>
                      Cancel Workflow:
                    </Text>
                    <Text fontSize="sm" color={{ base: "gray.700", _dark: "gray.300" }}>
                      Stops the workflow and deletes all associated resources.
                    </Text>
                  </Box>
                </Stack>
              </Box>
              
              {workflow.Status !== 'running' && (
                <Text fontSize="sm" color="gray.500">
                  Actions can only be performed on running workflows
                </Text>
              )}
            </Stack>
          </CardBody>
        </Card>
      </Stack>
      </Box>
    );
  } catch (error) {
    console.error("Error rendering WorkflowDetails component:", error);
    return (
      <Alert status="error" mt={5}>
        <AlertIcon />
        <Box>
          <AlertTitle>Rendering Error</AlertTitle>
          <AlertDescription>
            There was an error rendering the workflow details. This might be related to the LoadTestSpec data.
            <Text mt={2}>Error: {error instanceof Error ? error.message : String(error)}</Text>
          </AlertDescription>
        </Box>
      </Alert>
    );
  }
};

export default WorkflowDetails;

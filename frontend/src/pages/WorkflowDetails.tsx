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
  Flex
} from '@chakra-ui/react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { workflowApi } from '../api/workflowApi';
import type { LoadTestSpec, WorkflowStatus } from '../types/workflow';
import { ExternalLinkIcon, CopyIcon, CloseIcon, ChevronDownIcon, ChevronUpIcon } from '@chakra-ui/icons';

const WorkflowDetails = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();

  // State for collapsible cards
  const [isNodesExpanded, setIsNodesExpanded] = useState(true);
  const [isValidatorsExpanded, setIsValidatorsExpanded] = useState(true);
  const [isLoadBalancersExpanded, setIsLoadBalancersExpanded] = useState(true);

  const { data: workflow, isLoading, error, refetch } = useQuery<WorkflowStatus>({
    queryKey: ['workflow', id],
    queryFn: () => workflowApi.getWorkflow(id!),
    refetchInterval: 10000, // Polling every 5 seconds
    enabled: !!id,
  });

  // Log workflow data when it changes
  useEffect(() => {
    if (workflow) {
      console.log("Workflow data received:", workflow);
      
      // Add more detailed logging for debugging the gray screen issue
      if (workflow.loadTestSpec) {
        console.log("LoadTestSpec found:", workflow.loadTestSpec);
        try {
          // Safely stringify the LoadTestSpec to check for circular references or other issues
          const loadTestSpecString = JSON.stringify(workflow.loadTestSpec);
          console.log("LoadTestSpec stringified successfully:", loadTestSpecString);
          
          // Normalize the LoadTestSpec structure to match the expected interface
          // Use type assertion to avoid TypeScript errors
          const loadTestSpec = workflow.loadTestSpec as any;
          
          const normalizedLoadTestSpec: LoadTestSpec = {
            name: loadTestSpec.Name || loadTestSpec.name || "",
            description: loadTestSpec.Description || loadTestSpec.description || "",
            chain_id: loadTestSpec.ChainID || loadTestSpec.chain_id || "",
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
          
          console.log("Normalized LoadTestSpec:", normalizedLoadTestSpec);
          
          // Replace the original LoadTestSpec with the normalized version
          workflow.loadTestSpec = normalizedLoadTestSpec;
        } catch (error) {
          console.error("Error normalizing LoadTestSpec:", error);
        }
      }
    }
  }, [workflow]);

  const runLoadTestMutation = useMutation({
    mutationFn: (spec: LoadTestSpec) => workflowApi.runLoadTest(id!, spec),
    onSuccess: () => {
      toast({
        title: 'Adhoc load test wen',
        status: 'info',
        duration: 3000,
      });
      refetch();
    },
    onError: (error) => {
      toast({
        title: 'Error starting load test',
        description: error instanceof Error ? error.message : 'Unknown error occurred',
        status: 'error',
        duration: 5000,
      });
    },
  });

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

  const shutdownTestnetMutation = useMutation({
    mutationFn: () => {
      if (!workflow) return Promise.reject('No workflow data available');
      return workflowApi.sendShutdownSignal(id!);
    },
    onSuccess: () => {
      toast({
        title: 'Shutdown signal sent',
        description: 'The shutdown signal has been sent to the workflow',
        status: 'success',
        duration: 3000,
      });
      // Invalidate the workflow query to refresh the data
      queryClient.invalidateQueries({ queryKey: ['workflow', id] });
    },
    onError: (error) => {
      toast({
        title: 'Error sending shutdown signal',
        description: error instanceof Error ? error.message : String(error),
        status: 'error',
        duration: 5000,
      });
    },
  });

  const handleRunLoadTest = () => {
    const loadTestSpec: LoadTestSpec = {
      name: 'basic-load-test',
      description: 'Basic load test configuration',
      chain_id: 'test-chain',
      NumOfBlocks: 10,
      NumOfTxs: 5,
      msgs: [],
      unordered_txs: true,
      tx_timeout: '30s',
    };
    runLoadTestMutation.mutate(loadTestSpec);
  };

  const handleCancelWorkflow = () => {
    if (!workflow) return;
    
    const confirmMessage = 'Are you sure you want to cancel this workflow? This will stop processing abruptly.';
      
    if (window.confirm(confirmMessage)) {
      cancelWorkflowMutation.mutate();
    }
  };

  const handleShutdownTestnet = () => {
    if (!workflow) return;
    
    const confirmMessage = 'Are you sure you want to send a shutdown signal to this testnet? This will gracefully complete the workflow.';
      
    if (window.confirm(confirmMessage)) {
      shutdownTestnetMutation.mutate();
    }
  };

  const handleCloneWorkflow = () => {
    if (!workflow) return;
    
    console.log("Cloning workflow:", workflow);
    
    // Create query parameters with workflow data
    const params = new URLSearchParams();
    
    // Use the config field if available
    if (workflow.config) {
      console.log("Using config field for cloning:", workflow.config);
      
      // Basic workflow parameters
      if (workflow.config.Repo) params.append('repo', workflow.config.Repo);
      if (workflow.config.SHA) params.append('sha', workflow.config.SHA);
      if (workflow.config.RunnerType) params.append('runnerType', workflow.config.RunnerType);
      
      // EVM flag - always include it regardless of value
      params.append('evm', workflow.config.evm === true ? 'true' : 'false');
      
      // Long running testnet and duration
      if (workflow.config.LongRunningTestnet !== undefined) {
        params.append('longRunningTestnet', workflow.config.LongRunningTestnet ? 'true' : 'false');
      }
      
      if (workflow.config.TestnetDuration) {
        // Convert nanoseconds to hours if needed
        let duration = workflow.config.TestnetDuration;
        if (duration > 1000000000) { // If it's in nanoseconds
          duration = duration / (60 * 60 * 1000000000); // Convert to hours
        }
        params.append('testnetDuration', duration.toString());
      }
      
      // Number of wallets
      if (workflow.config.NumWallets) {
        params.append('numWallets', workflow.config.NumWallets.toString());
      }
      
      // Chain config
      if (workflow.config.ChainConfig) {
        if (workflow.config.ChainConfig.Name) {
          params.append('chainName', workflow.config.ChainConfig.Name);
        }
        
        if (workflow.config.ChainConfig.NumOfNodes) {
          params.append('numOfNodes', workflow.config.ChainConfig.NumOfNodes.toString());
        }
        
        if (workflow.config.ChainConfig.NumOfValidators) {
          params.append('numOfValidators', workflow.config.ChainConfig.NumOfValidators.toString());
        }

        
        // Genesis modifications
        if (workflow.config.ChainConfig.GenesisModifications && 
            workflow.config.ChainConfig.GenesisModifications.length > 0) {
          params.append('genesisModifications', 
            JSON.stringify(workflow.config.ChainConfig.GenesisModifications));
        }
      }
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
            </SimpleGrid>
          </CardBody>
        </Card>

        {/* Testnet Setup Card - shown when testnet is still being set up */}
        {workflow.Status === 'running' && 
          ((!workflow.Nodes || workflow.Nodes.length === 0) || 
           (!workflow.Validators || workflow.Validators.length === 0) ||
           (workflow.Nodes.length === 3 && 
            workflow.Nodes[0].Name === 'validator-0' && 
            workflow.Nodes[0].RPC === 'http://validator-0:26657')) ? (
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
                              <Link 
                                href={node.LCD} 
                                target="_blank" 
                                color="blue.500"
                                fontSize="sm"
                                display="flex"
                                alignItems="center"
                                gap={1}
                              >
                                {node.LCD}
                                <Icon as={ExternalLinkIcon} boxSize={3} />
                              </Link>
                            </HStack>                
                          </Stack>
                        </Box>
                      ))}
                    </SimpleGrid>
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
                              <Link 
                                href={validator.LCD} 
                                target="_blank" 
                                color="blue.500"
                                fontSize="sm"
                                display="flex"
                                alignItems="center"
                                gap={1}
                              >
                                {validator.LCD}
                                <Icon as={ExternalLinkIcon} boxSize={3} />
                              </Link>
                            </HStack>
                          </Stack>
                        </Box>
                      ))}
                    </SimpleGrid>
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
                          <Link 
                            href={node.LCD} 
                            target="_blank" 
                            color="blue.500"
                            fontSize="sm"
                            display="flex"
                            alignItems="center"
                            gap={1}
                          >
                            {node.LCD}
                            <Icon as={ExternalLinkIcon} boxSize={3} />
                          </Link>
                        </HStack>
                      </Stack>
                    </Box>
                  ))}
                </SimpleGrid>
              </CardBody>
            </Collapse>
          </Card>
        )}

        {/* Monitoring Card */}
        {(workflow.Monitoring && Object.keys(workflow.Monitoring).length > 0) || 
         (workflow.config?.RunnerType === 'Docker') ? (
          <Card>
            <CardHeader>
              <Heading size="md">Monitoring Dashboards</Heading>
            </CardHeader>
            <CardBody>
              {(workflow.config?.RunnerType === 'Docker') ? (
                <Alert status="info">
                  <AlertIcon />
                  <Box>
                    <AlertTitle>Monitoring Not Available</AlertTitle>
                    <AlertDescription>
                      Monitoring dashboards are only available for testnets running on Digital Ocean. 
                      Docker testnets do not send metrics to Grafana.
                    </AlertDescription>
                  </Box>
                </Alert>
              ) : (
                <Stack spacing={3}>
                  {Object.entries(workflow.Monitoring).map(([name, url]) => (
                    <HStack key={name} spacing={3}>
                      <Text fontWeight="semibold" minW="100px" textTransform="capitalize">
                        {name}:
                      </Text>
                      <Link 
                        href={url} 
                        target="_blank" 
                        color="blue.500"
                        display="flex"
                        alignItems="center"
                        gap={2}
                      >
                        {url}
                        <Icon as={ExternalLinkIcon} boxSize={4} />
                      </Link>
                    </HStack>
                  ))}
                </Stack>
              )}
            </CardBody>
          </Card>
        ) : null}

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
                <Button
                  leftIcon={<CloseIcon />}
                  colorScheme="red"
                  onClick={handleShutdownTestnet}
                  isLoading={shutdownTestnetMutation.isPending}
                  loadingText="Sending Shutdown Signal..."
                  disabled={workflow.Status !== 'running'}
                  size="lg"
                >
                  Shutdown Testnet
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
                      Stops processing the workflow abruptly. For long-running testnets, no resources are deleted or stopped.
                    </Text>
                  </Box>
                  <Box>
                    <Text fontWeight="semibold" color="red.600" mb={1}>
                      Shutdown Testnet:
                    </Text>
                    <Text fontSize="sm" color={{ base: "gray.700", _dark: "gray.300" }}>
                      Sends a shutdown signal to the workflow which gracefully completes the workflow. For long-running testnets, no resources are deleted or stopped.
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

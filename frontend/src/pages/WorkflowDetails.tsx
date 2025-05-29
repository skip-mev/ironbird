import { useParams, useNavigate } from 'react-router-dom';
import { useEffect } from 'react';
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
  Divider,
  ButtonGroup
} from '@chakra-ui/react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { workflowApi } from '../api/workflowApi';
import type { LoadTestSpec, WorkflowStatus } from '../types/workflow';
import { ExternalLinkIcon, CopyIcon } from '@chakra-ui/icons';

const WorkflowDetails = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();

  const { data: workflow, isLoading, error, refetch } = useQuery<WorkflowStatus>({
    queryKey: ['workflow', id],
    queryFn: () => workflowApi.getWorkflow(id!),
    refetchInterval: 5000, // Polling every 5 seconds
    enabled: !!id,
  });

  // Log workflow data when it changes
  useEffect(() => {
    if (workflow) {
      console.log("Workflow data received:", workflow);
      console.log("Config field:", workflow.Config);
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

  const handleRunLoadTest = () => {
    const loadTestSpec: LoadTestSpec = {
      name: 'basic-load-test',
      description: 'Basic load test configuration',
      chain_id: 'test-chain',
      num_of_blocks: 100,
      msgs: [],
      unordered_txs: true,
      tx_timeout: '30s',
    };
    runLoadTestMutation.mutate(loadTestSpec);
  };

  const handleCloneWorkflow = () => {
    if (!workflow) return;
    
    console.log("Cloning workflow:", workflow);
    
    // Create query parameters with workflow data
    const params = new URLSearchParams();
    
    // Add workflow configuration parameters if available
    if (workflow.Config) {
      console.log("Using Config from workflow:", workflow.Config);
      
      // Basic fields
      params.append('repo', workflow.Config.Repo || '');
      params.append('sha', workflow.Config.SHA || '');
      params.append('runnerType', workflow.Config.RunnerType || 'Docker');
      params.append('longRunningTestnet', workflow.Config.LongRunningTestnet ? 'true' : 'false');
      
      if (workflow.Config.TestnetDuration) {
        // Convert nanoseconds to hours if needed
        let duration = workflow.Config.TestnetDuration;
        if (duration > 1000000000) { // If it's in nanoseconds
          duration = duration / (60 * 60 * 1000000000); // Convert to hours
        }
        params.append('testnetDuration', duration.toString());
      } else {
        params.append('testnetDuration', '2');
      }
      
      // Chain config
      if (workflow.Config.ChainConfig) {
        params.append('chainName', workflow.Config.ChainConfig.Name || 'test-chain');
        
        if (workflow.Config.ChainConfig.Image) {
          params.append('image', workflow.Config.ChainConfig.Image);
        }
        
        params.append('numOfNodes', 
          (workflow.Config.ChainConfig.NumOfNodes || 4).toString());
        
        params.append('numOfValidators', 
          (workflow.Config.ChainConfig.NumOfValidators || 3).toString());
        
        // Handle genesis modifications
        if (workflow.Config.ChainConfig.GenesisModifications && 
            workflow.Config.ChainConfig.GenesisModifications.length > 0) {
          params.append('genesisModifications', 
            JSON.stringify(workflow.Config.ChainConfig.GenesisModifications));
        }
      }
      
      // Handle load test spec
      if (workflow.Config.LoadTestSpec) {
        params.append('loadTestSpec', JSON.stringify(workflow.Config.LoadTestSpec));
      }
    } else {
      // If no Config is available, use the individual fields from the database
      // These fields should be populated from the database
      if (workflow.repo) params.append('repo', workflow.repo);
      if (workflow.sha) params.append('sha', workflow.sha);
      if (workflow.chainName) params.append('chainName', workflow.chainName);
      if (workflow.runnerType) params.append('runnerType', workflow.runnerType);
      
      if (workflow.numOfNodes) {
        params.append('numOfNodes', workflow.numOfNodes.toString());
      }
      
      if (workflow.numOfValidators) {
        params.append('numOfValidators', workflow.numOfValidators.toString());
      }
      
      if (workflow.longRunningTestnet !== undefined) {
        params.append('longRunningTestnet', workflow.longRunningTestnet ? 'true' : 'false');
      }
      
      if (workflow.testnetDuration) {
        // Convert nanoseconds to hours if needed
        let duration = workflow.testnetDuration;
        if (duration > 1000000000) { // If it's in nanoseconds
          duration = duration / (60 * 60 * 1000000000); // Convert to hours
        }
        params.append('testnetDuration', duration.toString());
      }
    }
    
    console.log("Final URL parameters:", Object.fromEntries(params.entries()));
    
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

        {/* Nodes Card */}
        <Card>
          <CardHeader>
            <Heading size="md">Network Nodes {workflow.Nodes && workflow.Nodes.length > 0 ? `(${workflow.Nodes.length})` : ''}</Heading>
          </CardHeader>
          <CardBody>
            {/* Check if we're in a running state and the nodes look like mock data */}
            {workflow.Status === 'running' && 
              ((!workflow.Nodes || workflow.Nodes.length === 0) || 
               (workflow.Nodes.length === 3 && 
                workflow.Nodes[0].Name === 'validator-0' && 
                workflow.Nodes[0].RPC === 'http://validator-0:26657')) ? (
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
                  The network nodes are being created. This process may take a few minutes. 
                  The page will automatically update when nodes are ready.
                </Text>
              </Box>
            ) : workflow.Nodes && workflow.Nodes.length > 0 ? (
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
                      <HStack>
                        <Text fontWeight="semibold" minW="60px" fontSize="sm">
                          Metrics:
                        </Text>
                        <Link 
                          href={node.Metrics} 
                          target="_blank" 
                          color="blue.500"
                          fontSize="sm"
                          display="flex"
                          alignItems="center"
                          gap={1}
                        >
                          {node.Metrics}
                          <Icon as={ExternalLinkIcon} boxSize={3} />
                        </Link>
                      </HStack>
                    </Stack>
                  </Box>
                ))}
              </SimpleGrid>
            ) : (
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
                <Text fontSize="lg" fontWeight="medium" color="text" mb={2}>
                  No Network Nodes Available
                </Text>
                <Text color="textSecondary" textAlign="center" maxW="md">
                  There are no network nodes available for this workflow.
                </Text>
              </Box>
            )}
          </CardBody>
        </Card>

        {/* Monitoring Card */}
        {workflow.Monitoring && Object.keys(workflow.Monitoring).length > 0 && (
          <Card>
            <CardHeader>
              <Heading size="md">Monitoring & Dashboards</Heading>
            </CardHeader>
            <CardBody>
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
            </CardBody>
          </Card>
        )}

        {/* Actions Card */}
        <Card>
          <CardHeader>
            <Heading size="md">Actions</Heading>
          </CardHeader>
          <CardBody>
            <Stack spacing={4}>
              <ButtonGroup spacing={4}>
                <Button
                  colorScheme="blue"
                  onClick={handleRunLoadTest}
                  isLoading={runLoadTestMutation.isPending}
                  loadingText="Starting Load Test..."
                  disabled={workflow.Status !== 'running'}
                  size="lg"
                >
                  Run Load Test
                </Button>
                <Button
                  leftIcon={<CopyIcon />}
                  colorScheme="purple"
                  onClick={handleCloneWorkflow}
                  size="lg"
                >
                  Clone Workflow
                </Button>
              </ButtonGroup>
              {workflow.Status !== 'running' && (
                <Text fontSize="sm" color="gray.500">
                  Load test can only be run on running workflows
                </Text>
              )}
            </Stack>
          </CardBody>
        </Card>
      </Stack>
    </Box>
  );
};

export default WorkflowDetails;

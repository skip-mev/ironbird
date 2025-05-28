import { useParams } from 'react-router-dom';
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
  Divider
} from '@chakra-ui/react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { workflowApi } from '../api/workflowApi';
import type { LoadTestSpec } from '../types/workflow';
import { ExternalLinkIcon } from '@chakra-ui/icons';

const WorkflowDetails = () => {
  const { id } = useParams<{ id: string }>();
  const toast = useToast();

  const { data: workflow, isLoading, error, refetch } = useQuery({
    queryKey: ['workflow', id],
    queryFn: () => workflowApi.getWorkflow(id!),
    refetchInterval: 5000, // Polling every 5 seconds
    enabled: !!id,
  });

  const runLoadTestMutation = useMutation({
    mutationFn: (spec: LoadTestSpec) => workflowApi.runLoadTest(id!, spec),
    onSuccess: () => {
      toast({
        title: 'Load test started',
        status: 'success',
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
        {workflow.Nodes && workflow.Nodes.length > 0 && (
          <Card>
            <CardHeader>
              <Heading size="md">Network Nodes ({workflow.Nodes.length})</Heading>
            </CardHeader>
            <CardBody>
              <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={4}>
                {workflow.Nodes.map((node) => (
                  <Box 
                    key={node.Name} 
                    bg="gray.50" 
                    p={4} 
                    borderRadius="md" 
                    border="1px"
                    borderColor="gray.200"
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
            </CardBody>
          </Card>
        )}

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
            {workflow.Status !== 'running' && (
              <Text fontSize="sm" color="gray.500" mt={2}>
                Load test can only be run on running workflows
              </Text>
            )}
          </CardBody>
        </Card>
      </Stack>
    </Box>
  );
};

export default WorkflowDetails; 
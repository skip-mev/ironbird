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
  useToast
} from '@chakra-ui/react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { workflowApi } from '../api/workflowApi';
import type { LoadTestSpec } from '../types/workflow';

const WorkflowDetails = () => {
  const { id } = useParams<{ id: string }>();
  const toast = useToast();

  const { data: workflow, isLoading } = useQuery({
    queryKey: ['workflow', id],
    queryFn: () => workflowApi.getWorkflow(id!),
    refetchInterval: 5000, // Polling every 5 seconds
  });

  const runLoadTestMutation = useMutation({
    mutationFn: (spec: LoadTestSpec) => workflowApi.runLoadTest(id!, spec),
    onSuccess: () => {
      toast({
        title: 'Load test started',
        status: 'success',
        duration: 3000,
      });
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
      Name: 'basic-load-test',
      Description: 'Basic load test configuration',
      ChainID: 'test-chain',
      NumOfBlocks: 100,
      Msgs: [],
      UnorderedTxs: true,
      TxTimeout: '30s',
    };
    runLoadTestMutation.mutate(loadTestSpec);
  };

  if (isLoading) {
    return <Text>Loading...</Text>;
  }

  if (!workflow) {
    return <Text>Workflow not found</Text>;
  }

  return (
    <Box>
      <Heading mb={6}>Workflow Details</Heading>
      <Stack direction="column" gap={4}>
        <Box>
          <Text fontWeight="bold">Workflow ID:</Text>
          <Text>{workflow.WorkflowID}</Text>
        </Box>

        <Box>
          <Text fontWeight="bold">Status:</Text>
          <Badge colorScheme={workflow.Status === 'running' ? 'green' : 'gray'}>
            {workflow.Status}
          </Badge>
        </Box>

        <Box>
          <Text fontWeight="bold" mb={2}>Nodes:</Text>
          {workflow.Nodes.map((node) => (
            <Box key={node.Name} bg="gray.50" p={4} borderRadius="md" mb={2}>
              <Text fontWeight="bold">{node.Name}</Text>
              <HStack mt={2}>
                <Link href={node.RPC} target="_blank" color="blue.500">RPC</Link>
                <Link href={node.LCD} target="_blank" color="blue.500">LCD</Link>
                <Link href={node.Metrics} target="_blank" color="blue.500">Metrics</Link>
              </HStack>
            </Box>
          ))}
        </Box>

        <Box>
          <Text fontWeight="bold" mb={2}>Monitoring:</Text>
          {Object.entries(workflow.Monitoring).map(([name, url]) => (
            <Link key={name} href={url} target="_blank" display="block" color="blue.500">
              {name}
            </Link>
          ))}
        </Box>

        <Button
          colorScheme="blue"
          onClick={handleRunLoadTest}
          disabled={runLoadTestMutation.isPending}
        >
          Run Load Test
        </Button>
      </Stack>
    </Box>
  );
};

export default WorkflowDetails; 
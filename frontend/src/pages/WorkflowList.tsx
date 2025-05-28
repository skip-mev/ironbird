import { useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  Heading,
  Text,
  Stack,
  Badge,
  useToast,
  Spinner,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
  HStack,
  IconButton,
  Tooltip,
} from '@chakra-ui/react';
import { useQuery } from '@tanstack/react-query';
import { workflowApi } from '../api/workflowApi';
import { ViewIcon, RepeatIcon } from '@chakra-ui/icons';

const WorkflowList = () => {
  const navigate = useNavigate();
  const toast = useToast();

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['workflows'],
    queryFn: workflowApi.listWorkflows,
    refetchInterval: 10000, // Refetch every 10 seconds
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'blue';
      case 'completed':
        return 'green';
      case 'failed':
        return 'red';
      case 'canceled':
      case 'terminated':
        return 'orange';
      case 'timed_out':
        return 'yellow';
      default:
        return 'gray';
    }
  };

  const handleViewWorkflow = (workflowId: string) => {
    navigate(`/workflow/${workflowId}`);
  };

  const handleRefresh = () => {
    refetch();
    toast({
      title: 'Refreshed',
      description: 'Workflow list has been refreshed',
      status: 'info',
      duration: 2000,
      isClosable: true,
    });
  };

  if (error) {
    return (
      <Box>
        <Alert status="error">
          <AlertIcon />
          <AlertTitle>Error loading workflows!</AlertTitle>
          <AlertDescription>
            {error instanceof Error ? error.message : 'An unknown error occurred'}
          </AlertDescription>
        </Alert>
        <Button mt={4} onClick={() => refetch()}>
          Try Again
        </Button>
      </Box>
    );
  }

  return (
    <Box>
      <HStack justify="space-between" mb={6}>
        <Heading size="lg">Testnet Workflows</Heading>
        <HStack>
          <Button
            colorScheme="blue"
            onClick={() => navigate('/')}
          >
            Create New Workflow
          </Button>
          <Tooltip label="Refresh">
            <IconButton
              aria-label="Refresh"
              icon={<RepeatIcon />}
              onClick={handleRefresh}
              isLoading={isLoading}
            />
          </Tooltip>
        </HStack>
      </HStack>

      {isLoading ? (
        <Box display="flex" justifyContent="center" alignItems="center" minH="200px">
          <Spinner size="xl" />
          <Text ml={4}>Loading workflows...</Text>
        </Box>
      ) : !data?.Workflows || data.Workflows.length === 0 ? (
        <Box textAlign="center" py={10}>
          <Text fontSize="lg" color="gray.600" mb={4}>
            No workflows found
          </Text>
          <Button colorScheme="blue" onClick={() => navigate('/')}>
            Create Your First Workflow
          </Button>
        </Box>
      ) : (
        <Box>
          <Text mb={4} color="gray.600">
            Found {data.Count} workflow{data.Count !== 1 ? 's' : ''}
          </Text>
          
          <TableContainer>
            <Table variant="simple">
              <Thead>
                <Tr>
                  <Th>Workflow ID</Th>
                  <Th>Repository</Th>
                  <Th>SHA</Th>
                  <Th>Status</Th>
                  <Th>Start Time</Th>
                  <Th>Actions</Th>
                </Tr>
              </Thead>
              <Tbody>
                {data.Workflows.map((workflow) => (
                  <Tr key={workflow.WorkflowID}>
                    <Td>
                      <Text 
                        fontFamily="mono" 
                        fontSize="sm"
                        maxW="200px"
                        isTruncated
                      >
                        {workflow.WorkflowID}
                      </Text>
                    </Td>
                    <Td>
                      <Text fontWeight="medium">
                        {workflow.Repo || '-'}
                      </Text>
                    </Td>
                    <Td>
                      <Text 
                        fontFamily="mono" 
                        fontSize="sm"
                        maxW="100px"
                        isTruncated
                      >
                        {workflow.SHA || '-'}
                      </Text>
                    </Td>
                    <Td>
                      <Badge colorScheme={getStatusColor(workflow.Status)}>
                        {workflow.Status}
                      </Badge>
                    </Td>
                    <Td>
                      <Text fontSize="sm">
                        {workflow.StartTime || '-'}
                      </Text>
                    </Td>
                    <Td>
                      <Tooltip label="View Details">
                        <IconButton
                          aria-label="View workflow details"
                          icon={<ViewIcon />}
                          size="sm"
                          onClick={() => handleViewWorkflow(workflow.WorkflowID)}
                        />
                      </Tooltip>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </TableContainer>
        </Box>
      )}
    </Box>
  );
};

export default WorkflowList; 
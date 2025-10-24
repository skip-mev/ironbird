import { useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  Heading,
  Text,
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
import { ViewIcon, RepeatIcon, ChevronLeftIcon, ChevronRightIcon } from '@chakra-ui/icons';
import { useState } from "react";

const WorkflowList = () => {
  const navigate = useNavigate();
  const toast = useToast();

  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 50;

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['workflows', currentPage],
    queryFn: () => workflowApi.listWorkflows(pageSize, (currentPage - 1) * pageSize),
    refetchInterval: 10000, // Refetch every 10 seconds
  });

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'brand';
      case 'completed':
        return 'success';
      case 'failed':
        return 'red';
      case 'canceled':
      case 'terminated':
        return 'warning';
      case 'timed_out':
        return 'accent';
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
        <Alert 
          status="error" 
          variant="subtle" 
          flexDirection="column" 
          alignItems="center" 
          justifyContent="center" 
          textAlign="center" 
          borderRadius="lg"
          boxShadow="md"
          p={6}
        >
          <AlertIcon boxSize="40px" mr={0} />
          <AlertTitle mt={4} mb={1} fontSize="lg">
            Error loading workflows!
          </AlertTitle>
          <AlertDescription maxWidth="sm">
            {error instanceof Error ? error.message : 'An unknown error occurred'}
          </AlertDescription>
          <Button mt={6} colorScheme="brand" onClick={() => refetch()}>
            Try Again
          </Button>
        </Alert>
      </Box>
    );
  }

  return (
    <Box>
      <HStack justify="space-between" mb={6}>
        <Heading size="lg" color="text">Testnet Workflows</Heading>
        <HStack>
          <Button
            colorScheme="brand"
            onClick={() => navigate('/')}
            leftIcon={<Box w="1em" />}
          >
            Create New Workflow
          </Button>
          <Tooltip label="Refresh">
            <IconButton
              aria-label="Refresh"
              icon={<RepeatIcon />}
              onClick={handleRefresh}
              isLoading={isLoading}
              colorScheme="brand"
            />
          </Tooltip>
        </HStack>
      </HStack>

      {isLoading ? (
        <Box display="flex" justifyContent="center" alignItems="center" minH="200px">
          <Spinner size="xl" color="brand.500" thickness="3px" />
          <Text ml={4} color="textSecondary">Loading workflows...</Text>
        </Box>
      ) : !data?.Workflows || data.Workflows.length === 0 ? (
        <Box textAlign="center" py={10} bg="surface" borderRadius="lg" boxShadow="sm" p={6}>
          <Text fontSize="lg" color="textSecondary" mb={4}>
            No workflows found
          </Text>
          <Button colorScheme="brand" onClick={() => navigate('/')}>
            Create Your First Workflow
          </Button>
        </Box>
      ) : (
        <Box>
          <Text mb={4} color="textSecondary">
            Showing {((currentPage - 1) * pageSize) + 1}-{Math.min(currentPage * pageSize, data.Total)} of {data.Total} workflow{data.Total !== 1 ? 's' : ''}
          </Text>
          
          <TableContainer 
            bg="surface" 
            borderRadius="lg" 
            boxShadow="sm" 
            overflow="hidden"
          >
            <Table variant="simple" size="md">
              <Thead>
                <Tr>
                  <Th>Workflow ID</Th>
                  <Th>Template</Th>
                  <Th>Provider</Th>
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
                      {workflow.TemplateID ? (
                        <Box>
                          <Text fontWeight="medium" fontSize="sm">
                            {workflow.TemplateID}
                          </Text>
                          {workflow.RunName && (
                            <Text fontSize="xs" color="textSecondary">
                              Run: {workflow.RunName}
                            </Text>
                          )}
                        </Box>
                      ) : (
                        <Badge colorScheme="gray" variant="subtle">
                          Manual
                        </Badge>
                      )}
                    </Td>
                    <Td>
                      <Badge colorScheme="blue" variant="subtle">
                        {workflow.Provider || 'Unknown'}
                      </Badge>
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
                          colorScheme="brand"
                          variant="outline"
                          onClick={() => handleViewWorkflow(workflow.WorkflowID)}
                        />
                      </Tooltip>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </TableContainer>

          <HStack justify="space-between" mt={4}>
            <Text fontSize="sm" color="textSecondary">
              Page {currentPage} of {Math.ceil(data.Total / pageSize)}
            </Text>
            <HStack>
              <IconButton
                aria-label="Previous page"
                icon={<ChevronLeftIcon />}
                onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                isDisabled={currentPage === 1}
                colorScheme="brand"
                variant="outline"
                size="sm"
              />
              <IconButton
                aria-label="Next page"
                icon={<ChevronRightIcon />}
                onClick={() => setCurrentPage(currentPage + 1)}
                isDisabled={currentPage >= Math.ceil(data.Total / pageSize)}
                colorScheme="brand"
                variant="outline"
                size="sm"
              />
            </HStack>
          </HStack>
        </Box>
      )}
    </Box>
  );
};

export default WorkflowList;

import { useState } from 'react';
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
  VStack,
  IconButton,
  Tooltip,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
  useDisclosure,
  Input,
  FormControl,
  FormLabel,
} from '@chakra-ui/react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { templateApi } from '../api/templateApi';
import { ViewIcon, DeleteIcon, TriangleUpIcon, InfoIcon } from '@chakra-ui/icons';
import type { WorkflowTemplateSummary } from '../types/workflow';

const TemplateList = () => {
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const { isOpen: isExecuteOpen, onOpen: onExecuteOpen, onClose: onExecuteClose } = useDisclosure();
  const { isOpen: isViewConfigOpen, onOpen: onViewConfigOpen, onClose: onViewConfigClose } = useDisclosure();

  const [selectedTemplate, setSelectedTemplate] = useState<WorkflowTemplateSummary | null>(null);
  const [executeTemplateId, setExecuteTemplateId] = useState<string>('');
  const [executeSha, setExecuteSha] = useState<string>('');
  const [executeRunName, setExecuteRunName] = useState<string>('');
  const [viewConfigTemplateId, setViewConfigTemplateId] = useState<string>('');

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['templates'],
    queryFn: () => templateApi.listTemplates(),
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  // Query for template config when viewing
  const { data: templateConfig, isLoading: isLoadingConfig, error: configError } = useQuery({
    queryKey: ['template-config', viewConfigTemplateId],
    queryFn: () => templateApi.getTemplate(viewConfigTemplateId),
    enabled: !!viewConfigTemplateId && isViewConfigOpen,
  });

  const deleteTemplateMutation = useMutation({
    mutationFn: templateApi.deleteTemplate,
    onSuccess: () => {
      toast({
        title: 'Template deleted',
        description: 'Template has been successfully deleted',
        status: 'success',
        duration: 3000,
      });
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      onClose();
    },
    onError: (error) => {
      toast({
        title: 'Error deleting template',
        description: error instanceof Error ? error.message : 'Unknown error occurred',
        status: 'error',
        duration: 5000,
      });
    },
  });

  const executeTemplateMutation = useMutation({
    mutationFn: templateApi.executeTemplate,
    onSuccess: (data) => {
      toast({
        title: 'Template executed',
        description: `Workflow created with ID: ${data.workflowId}`,
        status: 'success',
        duration: 5000,
      });
      navigate(`/workflow/${data.workflowId}`);
    },
    onError: (error) => {
      toast({
        title: 'Error executing template',
        description: error instanceof Error ? error.message : 'Unknown error occurred',
        status: 'error',
        duration: 5000,
      });
    },
  });

  const handleDeleteTemplate = (template: WorkflowTemplateSummary) => {
    setSelectedTemplate(template);
    onOpen();
  };

  const handleExecuteTemplate = (templateId: string) => {
    setExecuteTemplateId(templateId);
    setExecuteSha('');
    setExecuteRunName('');
    onExecuteOpen();
  };

  const handleViewConfig = (templateId: string) => {
    setViewConfigTemplateId(templateId);
    onViewConfigOpen();
  };

  const confirmDelete = () => {
    if (selectedTemplate) {
      deleteTemplateMutation.mutate(selectedTemplate.templateId);
    }
  };

  const confirmExecute = () => {
    if (executeTemplateId && executeSha.trim()) {
      executeTemplateMutation.mutate({
        templateId: executeTemplateId,
        sha: executeSha.trim(),
        runName: executeRunName.trim() || undefined,
      });
    }
  };

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return dateString;
    }
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
            Error loading templates!
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
        <Heading size="lg" color="text">Workflow Templates</Heading>
      </HStack>

      {isLoading ? (
        <Box display="flex" justifyContent="center" alignItems="center" minH="200px">
          <Spinner size="xl" color="brand.500" thickness="3px" />
          <Text ml={4} color="textSecondary">Loading templates...</Text>
        </Box>
      ) : !data?.templates || data.templates.length === 0 ? (
        <Box textAlign="center" py={10} bg="surface" borderRadius="lg" boxShadow="sm" p={6}>
          <Text fontSize="lg" color="textSecondary" mb={4}>
            No workflow templates found
          </Text>
          <Text fontSize="sm" color="textSecondary" mb={6}>
            Configure your workflow in the "Create Testnet" page and click "Save as Template" to create your first template.
          </Text>
          <Button colorScheme="brand" onClick={() => navigate('/')}>
            Go to Create Testnet
          </Button>
        </Box>
      ) : (
        <Box>
          <Text mb={4} color="textSecondary">
            Found {data.returnedCount} template{data.returnedCount !== 1 ? 's' : ''}
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
                  <Th>Name</Th>
                  <Th>Description</Th>
                  <Th>Runs</Th>
                  <Th>Updated</Th>
                  <Th>Actions</Th>
                </Tr>
              </Thead>
              <Tbody>
                {data.templates.map((template) => (
                  <Tr key={template.templateId}>
                    <Td>
                      <VStack align="start" spacing={1}>
                        <Text fontWeight="semibold" color="text">
                          {template.templateId}
                        </Text>
                        <Text 
                          fontSize="xs" 
                          color="textSecondary"
                          fontFamily="mono"
                        >
                          {template.templateId}
                        </Text>
                      </VStack>
                    </Td>
                    <Td>
                      <Text color="text" maxW="300px" noOfLines={2}>
                        {template.description || '-'}
                      </Text>
                    </Td>
                    <Td>
                      <Badge colorScheme="green" variant="subtle">
                        {template.runCount} runs
                      </Badge>
                    </Td>
                    <Td>
                      <Text fontSize="sm" color="textSecondary">
                        {formatDate(template.createdAt)}
                      </Text>
                    </Td>
                    <Td>
                      <HStack spacing={2}>
                        <Tooltip label="Execute Template">
                          <IconButton
                            aria-label="Execute template"
                            icon={<TriangleUpIcon />}
                            size="sm"
                            colorScheme="green"
                            variant="outline"
                            onClick={() => handleExecuteTemplate(template.templateId)}
                          />
                        </Tooltip>
                        <Tooltip label="View Config">
                          <IconButton
                            aria-label="View template config"
                            icon={<InfoIcon />}
                            size="sm"
                            colorScheme="blue"
                            variant="outline"
                            onClick={() => handleViewConfig(template.templateId)}
                          />
                        </Tooltip>
                        <Tooltip label="View History">
                          <IconButton
                            aria-label="View run history"
                            icon={<ViewIcon />}
                            size="sm"
                            colorScheme="brand"
                            variant="outline"
                            onClick={() => navigate(`/templates/${template.templateId}/runs`)}
                          />
                        </Tooltip>
                        <Tooltip label="Delete Template">
                          <IconButton
                            aria-label="Delete template"
                            icon={<DeleteIcon />}
                            size="sm"
                            colorScheme="red"
                            variant="outline"
                            onClick={() => handleDeleteTemplate(template)}
                          />
                        </Tooltip>
                      </HStack>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </TableContainer>
        </Box>
      )}

      {/* Delete Confirmation Modal */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Delete Template</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <Text>
              Are you sure you want to delete the template "{selectedTemplate?.templateId}"? 
              This action cannot be undone.
            </Text>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>
              Cancel
            </Button>
            <Button 
              colorScheme="red" 
              onClick={confirmDelete}
              isLoading={deleteTemplateMutation.isPending}
            >
              Delete
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Execute Template Modal */}
      <Modal isOpen={isExecuteOpen} onClose={onExecuteClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Execute Template</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>Commit SHA</FormLabel>
                <Input
                  placeholder="Enter commit SHA to use for this run"
                  value={executeSha}
                  onChange={(e) => setExecuteSha(e.target.value)}
                  fontFamily="mono"
                />
              </FormControl>
              <FormControl>
                <FormLabel>Run Name (Optional)</FormLabel>
                <Input
                  placeholder="Custom name for this execution"
                  value={executeRunName}
                  onChange={(e) => setExecuteRunName(e.target.value)}
                />
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onExecuteClose}>
              Cancel
            </Button>
            <Button 
              colorScheme="green" 
              onClick={confirmExecute}
              isLoading={executeTemplateMutation.isPending}
              isDisabled={!executeSha.trim()}
            >
              Execute
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* View Config Modal */}
      <Modal isOpen={isViewConfigOpen} onClose={onViewConfigClose} size="6xl">
        <ModalOverlay />
        <ModalContent maxH="90vh">
          <ModalHeader>Template Configuration</ModalHeader>
          <ModalCloseButton />
          <ModalBody overflow="auto">
            {isLoadingConfig ? (
              <Box display="flex" justifyContent="center" alignItems="center" minH="200px">
                <Spinner size="lg" color="brand.500" />
                <Text ml={4} color="textSecondary">Loading template config...</Text>
              </Box>
            ) : configError ? (
              <Alert status="error">
                <AlertIcon />
                <AlertTitle>Error loading template config!</AlertTitle>
                <AlertDescription>
                  {configError instanceof Error ? configError.message : 'Unknown error occurred'}
                </AlertDescription>
              </Alert>
            ) : templateConfig ? (
              <VStack spacing={4} align="stretch">
                <Box>
                  <Text fontWeight="semibold" mb={2}>Template: {templateConfig.templateId}</Text>
                  <Text fontSize="sm" color="textSecondary" mb={4}>
                    {templateConfig.description || 'No description provided'}
                  </Text>
                </Box>
                <Box>
                  <Text fontWeight="semibold" mb={2}>Configuration:</Text>
                  <Box
                    bg="gray.50"
                    borderRadius="md"
                    p={4}
                    border="1px solid"
                    borderColor="gray.200"
                    maxH="60vh"
                    overflow="auto"
                  >
                    <Text
                      as="pre"
                      fontSize="sm"
                      fontFamily="mono"
                      whiteSpace="pre-wrap"
                      wordBreak="break-word"
                    >
                      {JSON.stringify(templateConfig.templateConfig, null, 2)}
                    </Text>
                  </Box>
                </Box>
              </VStack>
            ) : null}
          </ModalBody>
          <ModalFooter>
            <Button onClick={onViewConfigClose}>
              Close
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  );
};

export default TemplateList;
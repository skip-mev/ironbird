import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  Heading,
  Text,
  Badge,
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
  Link,
  VStack,
  Card,
  CardBody,
  Stat,
  StatLabel,
  StatNumber,
  StatGroup
} from '@chakra-ui/react';
import { useQuery } from '@tanstack/react-query';
import { templateApi } from '../api/templateApi';
import { ViewIcon, ExternalLinkIcon, ArrowBackIcon } from '@chakra-ui/icons';

const TemplateRunHistory = () => {
  const { templateId } = useParams<{ templateId: string }>();
  const navigate = useNavigate();

  const { data, isLoading, error } = useQuery({
    queryKey: ['template-runs', templateId],
    queryFn: () => templateId ? templateApi.getTemplateRunHistory(templateId) : Promise.reject('No template ID'),
    enabled: !!templateId,
  });

  const { data: templateData } = useQuery({
    queryKey: ['template', templateId],
    queryFn: () => templateId ? templateApi.getTemplate(templateId) : Promise.reject('No template ID'),
    enabled: !!templateId,
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

  const formatDate = (dateString: string) => {
    try {
      if (!dateString || dateString.trim() === '') {
        return '-';
      }
      const date = new Date(dateString);
      if (isNaN(date.getTime())) {
        return '-';
      }
      return date.toLocaleString();
    } catch {
      return '-';
    }
  };

  const handleViewWorkflow = (workflowId: string) => {
    navigate(`/workflow/${workflowId}`);
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
            Error loading template runs!
          </AlertTitle>
          <AlertDescription maxWidth="sm">
            {error instanceof Error ? error.message : 'An unknown error occurred'}
          </AlertDescription>
        </Alert>
      </Box>
    );
  }

  return (
    <Box>
      <HStack justify="space-between" mb={6}>
        <HStack>
          <IconButton
            aria-label="Go back to templates"
            icon={<ArrowBackIcon />}
            variant="ghost"
            onClick={() => navigate('/templates')}
          />
          <VStack align="start" spacing={1}>
            <Heading size="lg" color="text">Template Run History</Heading>
            {templateData && (
              <Text color="textSecondary">
                {templateData.name}
              </Text>
            )}
          </VStack>
        </HStack>
      </HStack>

      {isLoading ? (
        <Box display="flex" justifyContent="center" alignItems="center" minH="200px">
          <Spinner size="xl" color="brand.500" thickness="3px" />
          <Text ml={4} color="textSecondary">Loading template runs...</Text>
        </Box>
      ) : !data?.runs || data.runs.length === 0 ? (
        <Box textAlign="center" py={10} bg="surface" borderRadius="lg" boxShadow="sm" p={6}>
          <Text fontSize="lg" color="textSecondary" mb={4}>
            No runs found for this template
          </Text>
          <Button 
            colorScheme="brand" 
            onClick={() => navigate('/templates')}
          >
            Back to Templates
          </Button>
        </Box>
      ) : (
        <VStack spacing={6} align="stretch">
          {/* Summary Stats */}
          <Card>
            <CardBody>
              <StatGroup>
                <Stat>
                  <StatLabel>Total Runs</StatLabel>
                  <StatNumber>{data.count}</StatNumber>
                </Stat>
                <Stat>
                  <StatLabel>Successful</StatLabel>
                  <StatNumber color="green.500">
                    {data.runs.filter(run => run.status === 'completed').length}
                  </StatNumber>
                </Stat>
                <Stat>
                  <StatLabel>Failed</StatLabel>
                  <StatNumber color="red.500">
                    {data.runs.filter(run => run.status === 'failed').length}
                  </StatNumber>
                </Stat>
                <Stat>
                  <StatLabel>Running</StatLabel>
                  <StatNumber color="blue.500">
                    {data.runs.filter(run => run.status === 'running').length}
                  </StatNumber>
                </Stat>
              </StatGroup>
            </CardBody>
          </Card>

          {/* Runs Table */}
          <TableContainer 
            bg="surface" 
            borderRadius="lg" 
            boxShadow="sm" 
            overflow="hidden"
          >
            <Table variant="simple" size="md">
              <Thead>
                <Tr>
                  <Th>Run Name</Th>
                  <Th>Description</Th>
                  <Th>SHA</Th>
                  <Th>Status</Th>
                  <Th>Monitoring</Th>
                  <Th>Actions</Th>
                </Tr>
              </Thead>
              <Tbody>
                {data.runs.map((run) => (
                  <Tr key={run.runId}>
                    <Td>
                      <Text fontWeight="medium">
                        {run.runName || `Run ${run.runId.slice(-8)}`}
                      </Text>
                    </Td>
                    <Td>
                      <Text color="text" maxW="200px" noOfLines={2} fontSize="sm">
                        {templateData?.description || '-'}
                      </Text>
                    </Td>
                    <Td>
                      <Text 
                        fontFamily="mono" 
                        fontSize="sm"
                        maxW="100px"
                        isTruncated
                      >
                        {run.sha}
                      </Text>
                    </Td>
                    <Td>
                      <Badge colorScheme={getStatusColor(run.status)}>
                        {run.status}
                      </Badge>
                    </Td>
                    <Td>
                      <HStack spacing={2}>
                        {Object.entries(run.monitoringLinks || {}).map(([key, url]) => {
                          // Check if this is a performance/monitoring link that should be split
                          const isPerformanceLink = key.toLowerCase().includes('performance') || 
                                                  key.toLowerCase().includes('cometbft') ||
                                                  (!key.toLowerCase().includes('metrics') && 
                                                   !key.toLowerCase().includes('pyroscope') && 
                                                   !key.toLowerCase().includes('grafana') && 
                                                   !key.toLowerCase().includes('profiling'));
                          
                          if (isPerformanceLink) {
                            // Create both metrics and profiles buttons
                            // For metrics, use the original URL (likely Grafana dashboard)
                            // For profiles, construct pyroscope URL matching workflow details implementation
                            const pyroscopeUrl = (() => {
                              const baseUrl = 'https://skipprotocol.grafana.net/a/grafana-pyroscope-app/explore?searchText=&panelType=time-series&layout=grid&hideNoData=off&explorationType=flame-graph&var-serviceName=ironbird&var-profileMetricId=goroutine:goroutine:count:goroutine:count&var-spanSelector=undefined&var-dataSource=grafanacloud-profiles&var-filters=provider%7C%3D%7C';
                              const provider = run.provider || 'unknown';
                              
                              let fromParam, toParam;
                              if (run.startedAt) {
                                // Convert run start time to Unix timestamp in milliseconds for Grafana
                                const startTimestamp = new Date(run.startedAt).getTime();
                                fromParam = `from=${startTimestamp}`;
                              } else {
                                fromParam = 'from=now-5m';
                              }
                              
                              if (run.completedAt) {
                                // Convert run end time to Unix timestamp in milliseconds for Grafana
                                const endTimestamp = new Date(run.completedAt).getTime();
                                toParam = `to=${endTimestamp}`;
                              } else {
                                // If run is still running, use "now"
                                toParam = 'to=now';
                              }
                              
                              return `${baseUrl}${provider}&var-filtersBaseline=&var-filtersComparison=&var-groupBy=&${fromParam}&${toParam}&maxNodes=16384&diffFrom=&diffTo=&diffFrom-2=&diffTo-2=`;
                            })();
                            
                            return (
                              <>
                                <Tooltip key={`${key}-metrics`} label="Open Metrics Dashboard">
                                  <Link href={url} isExternal>
                                    <Button
                                      size="xs"
                                      variant="outline"
                                      colorScheme="green"
                                      leftIcon={<ExternalLinkIcon />}
                                    >
                                      Metrics
                                    </Button>
                                  </Link>
                                </Tooltip>
                                <Tooltip key={`${key}-profiles`} label="Open Pyroscope Profiles">
                                  <Link href={pyroscopeUrl} isExternal>
                                    <Button
                                      size="xs"
                                      variant="outline"
                                      colorScheme="purple"
                                      leftIcon={<ExternalLinkIcon />}
                                    >
                                      Profiles
                                    </Button>
                                  </Link>
                                </Tooltip>
                              </>
                            );
                          } else {
                            // Handle specific metrics or pyroscope links
                            const isMetrics = key.toLowerCase().includes('metrics') || key.toLowerCase().includes('grafana');
                            const isPyroscope = key.toLowerCase().includes('pyroscope') || key.toLowerCase().includes('profiling');
                            
                            return (
                              <Tooltip key={key} label={`Open ${key}`}>
                                <Link href={url} isExternal>
                                  <Button
                                    size="xs"
                                    variant="outline"
                                    colorScheme={isMetrics ? "green" : isPyroscope ? "purple" : "blue"}
                                    leftIcon={<ExternalLinkIcon />}
                                  >
                                    {isMetrics ? "Metrics" : isPyroscope ? "Profiles" : key}
                                  </Button>
                                </Link>
                              </Tooltip>
                            );
                          }
                        })}
                      </HStack>
                    </Td>
                    <Td>
                      <Tooltip label="View Workflow Details">
                        <IconButton
                          aria-label="View workflow details"
                          icon={<ViewIcon />}
                          size="sm"
                          colorScheme="brand"
                          variant="outline"
                          onClick={() => handleViewWorkflow(run.workflowId)}
                        />
                      </Tooltip>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </TableContainer>
        </VStack>
      )}
    </Box>
  );
};

export default TemplateRunHistory;
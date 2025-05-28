import { ChakraProvider, Container, Box, Flex } from '@chakra-ui/react';
import { theme } from '@chakra-ui/theme';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import CreateWorkflow from './pages/CreateWorkflow';
import WorkflowDetails from './pages/WorkflowDetails';
import WorkflowList from './pages/WorkflowList';
import Navigation from './components/Navigation';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ChakraProvider theme={theme}>
        <Router>
          <Flex>
            <Navigation />
            <Box ml="250px" width="calc(100% - 250px)" minH="100vh">
              <Container maxW="container.xl" py={8}>
                <Routes>
                  <Route path="/" element={<CreateWorkflow />} />
                  <Route path="/workflows" element={<WorkflowList />} />
                  <Route path="/workflow/:id" element={<WorkflowDetails />} />
                </Routes>
              </Container>
            </Box>
          </Flex>
        </Router>
      </ChakraProvider>
    </QueryClientProvider>
  );
}

export default App;

import { Box, VStack, Heading, Flex, Button } from '@chakra-ui/react';
import { Link as RouterLink, useLocation } from 'react-router-dom';

const Navigation = () => {
  const location = useLocation();

  return (
    <Box 
      bg="gray.100" 
      h="100vh" 
      w="250px" 
      position="fixed" 
      left={0} 
      top={0} 
      boxShadow="md"
      py={6}
      px={4}
    >
      <VStack spacing={6} align="flex-start">
        <Heading size="md">
          <RouterLink to="/" style={{ color: '#2B6CB0' }}>
            Ironbird
          </RouterLink>
        </Heading>
        
        <VStack spacing={4} align="flex-start" w="100%">
          <Button
            as={RouterLink}
            to="/"
            variant={location.pathname === '/' ? 'solid' : 'ghost'}
            colorScheme="blue"
            justifyContent="flex-start"
          >
            Create Testnet
          </Button>
          <Button
            as={RouterLink}
            to="/workflows"
            variant={location.pathname.startsWith('/workflow') ? 'solid' : 'ghost'}
            colorScheme="blue"
            justifyContent="flex-start"
          >
            View Testnets
          </Button>
        </VStack>
      </VStack>
    </Box>
  );
};

export default Navigation; 
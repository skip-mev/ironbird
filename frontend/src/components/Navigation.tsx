import { Box, VStack, Heading, Flex } from '@chakra-ui/react';
import { Link } from 'react-router-dom';

const Navigation = () => {
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
          <Link to="/" style={{ color: '#2B6CB0' }}>
            Ironbird
          </Link>
        </Heading>
        
        <VStack spacing={4} align="flex-start" w="100%">
          <Link to="/" style={{ color: '#4A5568', width: '100%' }}>
            <Box p={2} _hover={{ bg: 'gray.200' }} borderRadius="md" w="100%">
              Create Testnet
            </Box>
          </Link>
          <Link to="/workflow" style={{ color: '#4A5568', width: '100%' }}>
            <Box p={2} _hover={{ bg: 'gray.200' }} borderRadius="md" w="100%">
              View Testnets
            </Box>
          </Link>
        </VStack>
      </VStack>
    </Box>
  );
};

export default Navigation; 
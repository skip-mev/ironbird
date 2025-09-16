import { Box, VStack, Heading, Flex, Button, IconButton, useColorMode, Tooltip } from '@chakra-ui/react';
import { MoonIcon, SunIcon } from '@chakra-ui/icons';
import { Link as RouterLink, useLocation } from 'react-router-dom';

const Navigation = () => {
  const location = useLocation();
  const { colorMode, toggleColorMode } = useColorMode();

  return (
    <Box 
      bg="sidebar" 
      h="100vh" 
      w="250px" 
      position="fixed" 
      left={0} 
      top={0} 
      boxShadow="md"
      py={6}
      px={4}
      borderRight="1px"
      borderColor="divider"
    >
      <VStack spacing={6} align="flex-start" w="100%">
        <Flex w="100%" justify="space-between" align="center">
          <Heading size="md" color="brand.500">
            <RouterLink to="/">
              Ironbird
            </RouterLink>
          </Heading>
          <Tooltip label={colorMode === 'light' ? 'Switch to dark mode' : 'Switch to light mode'}>
            <IconButton
              aria-label="Toggle color mode"
              icon={colorMode === 'light' ? <MoonIcon /> : <SunIcon />}
              onClick={toggleColorMode}
              variant="ghost"
              colorScheme="brand"
              size="sm"
            />
          </Tooltip>
        </Flex>
        
        <Box w="100%" h="1px" bg="divider" my={2}></Box>
        
        <VStack spacing={4} align="flex-start" w="100%">
          <Button
            as={RouterLink}
            to="/"
            variant={location.pathname === '/' ? 'solid' : 'ghost'}
            colorScheme="brand"
            justifyContent="flex-start"
            w="100%"
            leftIcon={<Box w="1em" />}
            _hover={{ bg: colorMode === 'light' ? 'gray.200' : 'gray.700' }}
          >
            Create Testnet
          </Button>
          <Button
            as={RouterLink}
            to="/workflows"
            variant={location.pathname.startsWith('/workflow') ? 'solid' : 'ghost'}
            colorScheme="brand"
            justifyContent="flex-start"
            w="100%"
            leftIcon={<Box w="1em" />}
            _hover={{ bg: colorMode === 'light' ? 'gray.200' : 'gray.700' }}
          >
            View Testnets
          </Button>
          <Button
            as={RouterLink}
            to="/templates"
            variant={location.pathname.startsWith('/template') ? 'solid' : 'ghost'}
            colorScheme="brand"
            justifyContent="flex-start"
            w="100%"
            leftIcon={<Box w="1em" />}
            _hover={{ bg: colorMode === 'light' ? 'gray.200' : 'gray.700' }}
          >
            Templates
          </Button>
        </VStack>
      </VStack>
    </Box>
  );
};

export default Navigation;

import { extendTheme, type ThemeConfig } from '@chakra-ui/react';

// Color mode config
const config: ThemeConfig = {
  initialColorMode: 'light',
  useSystemColorMode: false,
};

// Custom colors
const colors = {
  brand: {
    50: '#e6f1ff',
    100: '#cce3ff',
    200: '#99c7ff',
    300: '#66aaff',
    400: '#338eff',
    500: '#0072ff', // Primary brand color
    600: '#005bd9',
    700: '#0044b3',
    800: '#002e8c',
    900: '#001766',
  },
  accent: {
    50: '#f0f9ff',
    100: '#e0f2fe',
    200: '#bae6fd',
    300: '#7dd3fc',
    400: '#38bdf8',
    500: '#0ea5e9', // Accent color
    600: '#0284c7',
    700: '#0369a1',
    800: '#075985',
    900: '#0c4a6e',
  },
};

// Custom fonts
const fonts = {
  heading: 'Inter, system-ui, sans-serif',
  body: 'Inter, system-ui, sans-serif',
};

// Component style overrides
const components = {
  Code: {
    baseStyle: (props: { colorMode: string }) => ({
      bg: props.colorMode === 'dark' ? 'gray.700' : 'gray.50',
      color: props.colorMode === 'dark' ? 'gray.100' : 'gray.800',
      padding: '2',
      borderRadius: 'md',
    }),
  },
  Button: {
    baseStyle: {
      fontWeight: 'medium',
      borderRadius: 'md',
    },
    variants: {
      solid: (props: { colorMode: string; colorScheme: string }) => ({
        bg: props.colorScheme === 'brand' 
          ? `${props.colorScheme}.500` 
          : `${props.colorScheme}.500`,
        color: 'white',
        _hover: {
          bg: props.colorScheme === 'brand' 
            ? `${props.colorScheme}.600` 
            : `${props.colorScheme}.600`,
        },
      }),
      outline: (props: { colorMode: string; colorScheme: string }) => ({
        borderColor: props.colorScheme === 'brand' 
          ? `${props.colorScheme}.500` 
          : `${props.colorScheme}.500`,
        color: props.colorMode === 'dark' 
          ? `${props.colorScheme}.300` 
          : `${props.colorScheme}.500`,
      }),
    },
    defaultProps: {
      colorScheme: 'brand',
    },
  },
  Heading: {
    baseStyle: {
      fontWeight: 'semibold',
    },
  },
  Card: {
    baseStyle: (props: { colorMode: string }) => ({
      container: {
        bg: props.colorMode === 'dark' ? 'gray.800' : 'white',
        boxShadow: 'md',
        borderRadius: 'lg',
      },
    }),
  },
  Table: {
    variants: {
      simple: (props: { colorMode: string }) => ({
        th: {
          borderColor: props.colorMode === 'dark' ? 'gray.600' : 'gray.200',
          bg: props.colorMode === 'dark' ? 'gray.700' : 'gray.50',
        },
        td: {
          borderColor: props.colorMode === 'dark' ? 'gray.600' : 'gray.200',
        },
        tr: {
          _odd: {
            bg: props.colorMode === 'dark' ? 'gray.800' : 'white',
          },
          _even: {
            bg: props.colorMode === 'dark' ? 'gray.750' : 'gray.50',
          },
        },
      }),
    },
  },
};

// Semantic tokens
const semanticTokens = {
  colors: {
    // Interface colors
    background: { 
      default: 'white', 
      _dark: 'gray.900' 
    },
    surface: { 
      default: 'gray.50', 
      _dark: 'gray.800' 
    },
    sidebar: {
      default: 'gray.100',
      _dark: 'gray.800'
    },
    divider: {
      default: 'gray.200',
      _dark: 'gray.700'
    },
    // Text colors
    text: {
      default: 'gray.800',
      _dark: 'gray.100'
    },
    textSecondary: {
      default: 'gray.600',
      _dark: 'gray.400'
    },
    // Status colors
    success: {
      default: 'green.500',
      _dark: 'green.300'
    },
    error: {
      default: 'red.500',
      _dark: 'red.300'
    },
    warning: {
      default: 'orange.500',
      _dark: 'orange.300'
    },
    info: {
      default: 'blue.500',
      _dark: 'blue.300'
    },
  },
};

// Create the theme
const theme = extendTheme({
  config,
  colors,
  fonts,
  components,
  semanticTokens,
  styles: {
    global: (props: { colorMode: string }) => ({
      body: {
        bg: props.colorMode === 'dark' ? 'gray.900' : 'white',
        color: props.colorMode === 'dark' ? 'gray.100' : 'gray.800',
      },
    }),
  },
});

export default theme;

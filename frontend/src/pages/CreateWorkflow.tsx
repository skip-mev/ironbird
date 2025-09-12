import { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  Box,
  Button,
  FormControl,
  FormLabel,
  Heading,
  Input,
  NumberInput,
  NumberInputField,
  Select,
  Stack,
  useToast,
  HStack,
  IconButton,
  Switch,
  Text,
  VStack,
  Flex,
  FormHelperText,
  Tooltip,
    Textarea,
} from '@chakra-ui/react';
import { useMutation } from '@tanstack/react-query';
import { workflowApi } from '../api/workflowApi';
import type { TestnetWorkflowRequest, LoadTestSpec } from '../types/workflow';

// Simple YAML parser for load test specs
const parseYamlToLoadTestSpec = (yamlString: string): LoadTestSpec | null => {
  try {
    const lines = yamlString.split('\n');
    const result: any = {};
    let currentMsgIndex = -1;
    let inMsgsSection = false;
    
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) continue;
      
      if (trimmed === 'msgs:') {
        inMsgsSection = true;
        result.msgs = [];
        continue;
      }
      
      if (inMsgsSection && trimmed.startsWith('-')) {
        // New message entry
        currentMsgIndex++;
        result.msgs[currentMsgIndex] = {};
        // Parse the key-value pairs on the same line
        const msgLine = trimmed.substring(1).trim();
        if (msgLine) {
          const [key, value] = msgLine.split(':').map(s => s.trim());
          if (key && value) {
            result.msgs[currentMsgIndex][key] = isNaN(Number(value)) ? value : Number(value);
          }
        }
      } else if (inMsgsSection && trimmed.includes(':') && !trimmed.startsWith('-')) {
        // Message property
        const [key, value] = trimmed.split(':').map(s => s.trim());
        if (key && value && currentMsgIndex >= 0) {
          result.msgs[currentMsgIndex][key] = isNaN(Number(value)) ? value : Number(value);
        }
      } else if (trimmed.includes(':') && !inMsgsSection) {
        // Top-level property
        const [key, value] = trimmed.split(':').map(s => s.trim());
        if (key && value) {
          if (value === '[]') {
            result[key] = [];
          } else if (value.endsWith('s') && !isNaN(Number(value.slice(0, -1)))) {
            // Handle time values like "0s"
            result[key] = value;
          } else if (!isNaN(Number(value))) {
            result[key] = Number(value);
          } else {
            result[key] = value;
          }
        }
      }
    }
    
    // Convert to LoadTestSpec format expected by the frontend
    const loadTestSpec: LoadTestSpec = {
      name: result.name || '',
      description: result.description || '',
      kind: result.kind || 'cosmos',
      chain_id: result.chain_id || '',
      NumOfBlocks: result.num_of_blocks || 0,
      NumOfTxs: result.num_of_txs || 0,
      unordered_txs: result.unordered_txs || false,
      tx_timeout: result.tx_timeout || '',
      send_interval: result.send_interval || '',
      num_batches: result.num_batches || 0,
      gas_denom: result.gas_denom || '',
      bech32_prefix: result.bech32_prefix || '',
      msgs: (result.msgs || []).map((msg: any) => ({
        type: msg.type || '',
        weight: msg.weight || 0,
        NumMsgs: msg.num_msgs || 1,
        num_msgs: msg.num_msgs || 1
      }))
    };
    
    return loadTestSpec;
  } catch (error) {
    console.error('Error parsing YAML:', error);
    return null;
  }
};
import { AddIcon, DeleteIcon, InfoIcon } from '@chakra-ui/icons';
import LoadTestForm from '../components/LoadTestForm';
import ChainConfigsModal from '../components/ChainConfigsModal';
import GenesisModificationsModal from '../components/GenesisModificationsModal';

const CreateWorkflow = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const toast = useToast();
  const [jsonMode, setJsonMode] = useState(false);
  
  // Default JSON content from hack/create-workflow.json
  const defaultJsonContent = `{
  "repo": "evm",
  "sha": "99abc7f3a47af1289ddd90c279e49f885ee06278",
  "isEvmChain": true,
  "chain_config": {
    "region_configs": [
      {
        "name": "nyc1",
        "num_of_validators": 0
      },
      {
        "name": "sfo2",
        "num_of_validators": 1
      },
      {
        "name": "ams3",
        "num_of_validators": 0
      },
      {
        "name": "fra1",
        "num_of_validators": 0
      },
      {
        "name": "sgp1",
        "num_of_validators": 0
      }
    ],
    "name": "foobar-0",
    "image": "evm",
    "custom_consensus_config": "{\\"consensus\\":{\\"timeout_commit\\":\\"800ms\\"}}",
    "custom_app_config": "{\\"json-rpc\\":{\\"enable-profiling\\":true}}",
    "custom_client_config": "{\\"chain-id\\":\\"262144\\"}",
    "set_persistent_peers": true,
    "genesis_modifications":
      [
        {
          "key": "app_state.staking.params.bond_denom",
          "value": "atest"
        },
        {
          "key": "app_state.gov.params.expedited_voting_period",
          "value": "120s"
        },
        {
          "key": "app_state.gov.params.voting_period",
          "value": "300s"
        },
        {
          "key": "app_state.gov.params.expedited_min_deposit.0.amount",
          "value": "1"
        },
        {
          "key": "app_state.gov.params.expedited_min_deposit.0.denom",
          "value": "atest"
        },
        {
          "key": "app_state.gov.params.min_deposit.0.amount",
          "value": "1"
        },
        {
          "key": "app_state.gov.params.min_deposit.0.denom",
          "value": "atest"
        },
        {
          "key": "app_state.evm.params.evm_denom",
          "value": "atest"
        },
        {
          "key": "app_state.mint.params.mint_denom",
          "value": "atest"
        },
        {
          "key": "app_state.bank.denom_metadata",
          "value": "[{\\"description\\":\\"The native staking token for evmd.\\",\\"denom_units\\":[{\\"denom\\":\\"atest\\",\\"exponent\\":0,\\"aliases\\":[\\"attotest\\"]},{\\"denom\\":\\"test\\",\\"exponent\\":18,\\"aliases\\":[]}],\\"base\\":\\"atest\\",\\"display\\":\\"test\\",\\"name\\":\\"Test Token\\",\\"symbol\\":\\"TEST\\",\\"uri\\":\\"\\",\\"uri_hash\\":\\"\\"}]"
        },
        {
          "key": "app_state.erc20.native_precompiles",
          "value": "[\\"0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE\\"]"
        },
        {
          "key": "app_state.evm.params.active_static_precompiles",
          "value": "[\\"0x0000000000000000000000000000000000000100\\",\\"0x0000000000000000000000000000000000000400\\",\\"0x0000000000000000000000000000000000000800\\",\\"0x0000000000000000000000000000000000000801\\",\\"0x0000000000000000000000000000000000000802\\",\\"0x0000000000000000000000000000000000000803\\",\\"0x0000000000000000000000000000000000000804\\",\\"0x0000000000000000000000000000000000000805\\"]"
        },
        {
          "key": "app_state.erc20.token_pairs",
          "value": "[{\\"contract_owner\\": 1,\\"erc20_address\\": \\"0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE\\",\\"denom\\": \\"atest\\",\\"enabled\\": true}]"
        },
        {
          "key": "consensus.params.block.max_gas",
          "value": "550000000"
        },
        {
          "key": "app_state.feemarket.params.no_base_fee",
          "value": "true"
        }
      ]
  },
  "encoded_load_test_spec": "{\\"name\\":\\"eth_loadtest\\",\\"description\\":\\"testing\\",\\"kind\\":\\"eth\\",\\"chain_id\\":\\"262144\\",\\"send_interval\\":\\"900ms\\",\\"num_batches\\":30,\\"msgs\\":[{\\"type\\":\\"MsgNativeTransferERC20\\",\\"num_msgs\\":1000}],\\"chain_config\\":{\\"tx_opts\\":{\\"gas_fee_cap\\":10000000,\\"gas_tip_cap\\":10000000}}}",
  "runner_type": "DigitalOcean",
  "long_running_testnet": false,
  "testnet_duration": "10m",
  "num_wallets": 3000,
  "launch_load_balancer": false
}`;
  
  const [jsonInput, setJsonInput] = useState<string>(defaultJsonContent);
  const [isParsing, setIsParsing] = useState(false);
  // Default values to use as placeholders
  const defaultValues: TestnetWorkflowRequest = {
    Repo: 'cosmos-sdk',
    SHA: '',
    ChainConfig: {
      Name: 'test-chain',
      Image: '',
      GenesisModifications: [],
      NumOfNodes: 4,
      NumOfValidators: 3,
    },
    RunnerType: 'Docker',
    IsEvmChain: false,
    LoadTestSpec: undefined,
    LongRunningTestnet: false,
    LaunchLoadBalancer: false,
    TestnetDuration: '2h', // 2 hours
    NumWallets: 2500, // Default number of wallets
  };

  // Initialize form data with empty values
  const [formData, setFormData] = useState<TestnetWorkflowRequest>({
    Repo: '',
    SHA: '',
    ChainConfig: {
      Name: '',
      Image: '',
      Version: '',
      GenesisModifications: [],
      NumOfNodes: 0,
      NumOfValidators: 0,
      RegionConfigs: [],
    },
    RunnerType: '',
    IsEvmChain: false,
    LoadTestSpec: undefined,
    LongRunningTestnet: false,
    LaunchLoadBalancer: false,
    TestnetDuration: '',
    NumWallets: 2500,
    CatalystVersion: '',
  });



  // Separate display states for better UX
  const [nodeInputValue, setNodeInputValue] = useState('');

  // Sync nodeInputValue with formData when not loading from URL
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    // Only sync if there are no URL parameters (not cloning)
    if (!params.toString()) {
      setNodeInputValue(formData.ChainConfig.NumOfNodes?.toString() || '');
    }
  }, [formData.ChainConfig.NumOfNodes, location.search]);

  // Auto-disable LaunchLoadBalancer when runner type changes away from DigitalOcean
  useEffect(() => {
    if (formData.LaunchLoadBalancer && formData.RunnerType !== 'DigitalOcean') {
      setFormData(prev => ({
        ...prev,
        LaunchLoadBalancer: false
      }));
    }
  }, [formData.RunnerType, formData.LaunchLoadBalancer]);

  // Shared validation between form mode and JSON mode
  const validateRequest = (data: TestnetWorkflowRequest): string | null => {
    const requiredFields: { name: string; value: string | number }[] = [
      { name: 'Repository', value: data.Repo },
      { name: 'Commit SHA', value: data.SHA },
      { name: 'Chain Name', value: data.ChainConfig.Name },
      { name: 'Runner Type', value: data.RunnerType },
    ];

    if (data.RunnerType === 'Docker') {
      if (data.ChainConfig.NumOfValidators === undefined || data.ChainConfig.NumOfValidators < 0) {
        requiredFields.push({ name: 'Number of Validators', value: data.ChainConfig.NumOfValidators || 0 });
      }
      if (data.ChainConfig.NumOfNodes === undefined || data.ChainConfig.NumOfNodes < 0) {
        requiredFields.push({ name: 'Number of Nodes', value: data.ChainConfig.NumOfNodes || 0 });
      }
    } else if (data.RunnerType === 'DigitalOcean') {
      if (!data.ChainConfig.RegionConfigs || data.ChainConfig.RegionConfigs.length === 0) {
        return 'DigitalOcean deployment requires regional configuration';
      }
      const totalValidators = data.ChainConfig.RegionConfigs.reduce((sum, rc) => sum + (rc.numOfValidators || 0), 0);
      if (totalValidators === 0) {
        return 'DigitalOcean deployment requires at least one region to have validators';
      }
    }

    if (data.Repo === 'cometbft') {
      requiredFields.push({ name: 'Chain Image', value: data.ChainConfig.Image });
      // For cometbft, version is required
      requiredFields.push({ name: 'Simapp Version', value: data.ChainConfig.Version || '' });
    }

    if (!data.LongRunningTestnet) {
      requiredFields.push({ name: 'Testnet Duration', value: data.TestnetDuration });
    }

    const empty = requiredFields.filter(field => {
      if (typeof field.value === 'number') {
        return field.value === 0 || field.value === undefined;
      }
      return field.value === '' || field.value === undefined;
    });
    if (empty.length > 0) {
      return `Please fill in all required fields: ${empty.map(f => f.name).join(', ')}`;
    }

    if (!data.ChainConfig.SetSeedNode && !data.ChainConfig.SetPersistentPeers) {
      return 'At least one of "Set Seed Node" or "Set Persistent Peers" must be enabled';
    }

    // Validate that if SetSeedNode is true, there must be at least 1 node configured
    if (data.ChainConfig.SetSeedNode) {
      if (data.RunnerType === 'Docker') {
        if (!data.ChainConfig.NumOfNodes || data.ChainConfig.NumOfNodes < 1) {
          return 'When "Set Seed Node" is enabled, you must configure at least 1 node';
        }
      } else if (data.RunnerType === 'DigitalOcean') {
        if (!data.ChainConfig.RegionConfigs || data.ChainConfig.RegionConfigs.length === 0) {
          return 'When "Set Seed Node" is enabled with DigitalOcean deployment, regional configuration is required';
        }
        const totalNodes = data.ChainConfig.RegionConfigs.reduce((sum, rc) => sum + (rc.numOfNodes || 0), 0);
        if (totalNodes < 1) {
          return 'When "Set Seed Node" is enabled, at least one region must have 1 or more nodes configured';
        }
      }
    }

    if (data.LaunchLoadBalancer && data.RunnerType !== 'DigitalOcean') {
      return 'Launch Load Balancer can only be enabled when using DigitalOcean as the runner type';
    }

    return null;
  };

  // Check for URL parameters (for cloning workflows)
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    
    console.log("URL parameters:", Object.fromEntries(params.entries()));
    
    // Only proceed if we have parameters
    if (params.toString()) {
      // Start with a fresh form data object based on default values
      // This ensures we have proper defaults for any fields not specified in the URL
      const newFormData: TestnetWorkflowRequest = {
        Repo: '',
        SHA: '',
        ChainConfig: {
          Name: '',
          Image: '',
          Version: '',
          GenesisModifications: [],
          NumOfNodes: 0,
          NumOfValidators: 0,
        },
        RunnerType: '',
        IsEvmChain: false,
        LoadTestSpec: undefined,
        LongRunningTestnet: false,
        LaunchLoadBalancer: false,
        TestnetDuration: '',
        NumWallets: 2500,
      };
      
      let hasChanges = false;
      
      if (params.get('repo')) {
        const repo = params.get('repo')!;
        newFormData.Repo = repo;
        newFormData.IsEvmChain = repo === 'evm';
        if (repo === 'evm') {
          newFormData.ChainConfig.Image = 'evm';
        } else if (repo === 'cometbft') {
          newFormData.ChainConfig.Image = 'simapp';
        } else if (repo === 'cosmos-sdk') {
          newFormData.ChainConfig.Image = 'simapp';
        } else if (repo === 'gaia') {
          newFormData.ChainConfig.Image = 'gaia';
        }
        hasChanges = true;
      }
      
      if (params.get('sha')) {
        newFormData.SHA = params.get('sha')!;
        hasChanges = true;
        console.log("Setting SHA:", newFormData.SHA);
      }
      
      if (params.get('runnerType')) {
        newFormData.RunnerType = params.get('runnerType')!;
        hasChanges = true;
        console.log("Setting runnerType:", newFormData.RunnerType);
      }
      
      if (params.get('chainName')) {
        newFormData.ChainConfig.Name = params.get('chainName')!;
        hasChanges = true;
        console.log("Setting chainName:", newFormData.ChainConfig.Name);
      }
      
      if (params.get('image')) {
        newFormData.ChainConfig.Image = params.get('image')!;
        hasChanges = true;
        console.log("Setting image:", newFormData.ChainConfig.Image);
      }
      
      const numOfNodesParam = params.get('numOfNodes');
      if (numOfNodesParam !== null) {
        const numNodes = parseInt(numOfNodesParam, 10);
        if (!isNaN(numNodes)) {
          newFormData.ChainConfig.NumOfNodes = numNodes;
          setNodeInputValue(numNodes.toString());
          hasChanges = true;
          console.log("Setting numOfNodes:", newFormData.ChainConfig.NumOfNodes);
        }
      }
      
      const numOfValidatorsParam = params.get('numOfValidators');
      if (numOfValidatorsParam !== null) {
        const numValidators = parseInt(numOfValidatorsParam, 10);
        if (!isNaN(numValidators)) {
          newFormData.ChainConfig.NumOfValidators = numValidators;
          hasChanges = true;
          console.log("Setting numOfValidators:", newFormData.ChainConfig.NumOfValidators);
        }
      }
      
      // Genesis modifications
      const genesisModsParam = params.get('genesisModifications');
      if (genesisModsParam) {
        try {
          const genesisMods = JSON.parse(genesisModsParam);
          if (Array.isArray(genesisMods)) {
            newFormData.ChainConfig.GenesisModifications = genesisMods;
            hasChanges = true;
            console.log("Setting genesisModifications:", newFormData.ChainConfig.GenesisModifications);
          }
        } catch (e) {
          console.error('Failed to parse genesis modifications', e);
        }
      }
      
      // Custom chain configurations
      const appConfigParam = params.get('appConfig');
      if (appConfigParam) {
        try {
          const appConfig = JSON.parse(appConfigParam);
          newFormData.ChainConfig.AppConfig = appConfig;
          hasChanges = true;
        } catch (e) {
          console.error('Failed to parse app config', e);
        }
      }
      
      const consensusConfigParam = params.get('consensusConfig');
      if (consensusConfigParam) {
        try {
          const consensusConfig = JSON.parse(consensusConfigParam);
          newFormData.ChainConfig.ConsensusConfig = consensusConfig;
          hasChanges = true;
        } catch (e) {
          console.error('Failed to parse consensus config', e);
        }
      }
      
      const clientConfigParam = params.get('clientConfig');
      if (clientConfigParam) {
        try {
          const clientConfig = JSON.parse(clientConfigParam);
          newFormData.ChainConfig.ClientConfig = clientConfig;
          hasChanges = true;
        } catch (e) {
          console.error('Failed to parse client config', e);
        }
      }
      
      if (params.get('longRunningTestnet')) {
        newFormData.LongRunningTestnet = params.get('longRunningTestnet') === 'true';
        hasChanges = true;
        console.log("Setting longRunningTestnet:", newFormData.LongRunningTestnet);
      }
      
      if (params.get('launchLoadBalancer')) {
        newFormData.LaunchLoadBalancer = params.get('launchLoadBalancer') === 'true';
        hasChanges = true;
        console.log("Setting launchLoadBalancer:", newFormData.LaunchLoadBalancer);
      }

      if (params.get('isEvmChain') && !params.get('repo')) {
        newFormData.IsEvmChain = params.get('isEvmChain') === 'true';
        hasChanges = true;
        console.log("Setting evm flag:", newFormData.IsEvmChain);
      }
      
      if (params.get('setSeedNode')) {
        newFormData.ChainConfig.SetSeedNode = params.get('setSeedNode') === 'true';
        hasChanges = true;
        console.log("Setting setSeedNode:", newFormData.ChainConfig.SetSeedNode);
      }

      if (params.get('setPersistentPeers')) {
        newFormData.ChainConfig.SetPersistentPeers = params.get('setPersistentPeers') === 'true';
        hasChanges = true;
        console.log("Setting setPersistentPeers:", newFormData.ChainConfig.SetPersistentPeers);
      }

      if (params.get('testnetDuration')) {
        const duration = params.get('testnetDuration')!;
        newFormData.TestnetDuration = duration;
        hasChanges = true;
        console.log("Setting testnetDuration:", newFormData.TestnetDuration);
      }
      
      if (params.get('numWallets')) {
        const numWallets = parseInt(params.get('numWallets')!, 10);
        if (!isNaN(numWallets)) {
          newFormData.NumWallets = numWallets;
          hasChanges = true;
          console.log("Setting numWallets:", newFormData.NumWallets);
        }
      }
      
      // Handle load test specs - simplified logic
      const encodedLoadTestSpecParam = params.get('encodedLoadTestSpec');
      const loadTestSpecParam = params.get('loadTestSpec');
      
      if (encodedLoadTestSpecParam) {
        try {
          // Parse the YAML-like encoded load test spec and convert to LoadTestSpec object
          const decodedYaml = decodeURIComponent(encodedLoadTestSpecParam);
          console.log("Decoded YAML:", decodedYaml);
          
          // Parse the YAML manually (simple key-value parsing)
          const loadTestSpec = parseYamlToLoadTestSpec(decodedYaml);
          console.log("Parsed LoadTestSpec:", loadTestSpec);
          
          if (loadTestSpec) {
            newFormData.LoadTestSpec = loadTestSpec;
            newFormData.EncodedLoadTestSpec = encodedLoadTestSpecParam; // Keep for submission
            setHasLoadTest(true);
            hasChanges = true;
            console.log("Successfully parsed and set load test spec from encoded YAML");
          } else {
            // Fallback: just set the encoded spec for backend processing
            newFormData.EncodedLoadTestSpec = encodedLoadTestSpecParam;
            setHasLoadTest(true);
            hasChanges = true;
            console.log("Could not parse YAML, using encoded spec for backend processing");
          }
        } catch (e) {
          console.error("Failed to parse encoded load test spec:", e);
          // Fallback: just set the encoded spec for backend processing
          newFormData.EncodedLoadTestSpec = encodedLoadTestSpecParam;
          setHasLoadTest(true);
          hasChanges = true;
        }
      } else if (loadTestSpecParam) {
        try {
          const parsedLoadTestSpec = JSON.parse(loadTestSpecParam);
          console.log("Parsed loadTestSpec from URL:", parsedLoadTestSpec);
          
          // Use the parsed spec directly without complex normalization
          newFormData.LoadTestSpec = parsedLoadTestSpec;
          setHasLoadTest(true);
          hasChanges = true;
        } catch (e) {
          console.error('Failed to parse load test spec', e);
        }
      }
      
      // Handle regional configurations for DigitalOcean
      const regionConfigsParam = params.get('regionConfigs');
      if (regionConfigsParam) {
        try {
          const regionConfigs = JSON.parse(regionConfigsParam);
          if (Array.isArray(regionConfigs)) {
            newFormData.ChainConfig.RegionConfigs = regionConfigs;
            hasChanges = true;
            console.log("Setting regionConfigs:", newFormData.ChainConfig.RegionConfigs);
          }
        } catch (e) {
          console.error('Failed to parse region configs', e);
        }
      }

      // Handle version parameter for dropdown selection
      if (params.get('version')) {
        const version = params.get('version')!;
        // Check if this is a predefined version for the dropdown
        const predefinedVersions = ['v0.47.17', 'v0.50.13', 'v0.53.0'];
        if (predefinedVersions.includes(version)) {
          newFormData.ChainConfig.Version = version;
          setIsCustomVersion(false);
        } else {
          // This is a custom version, store it in the Version field
          newFormData.ChainConfig.Version = version;
          setIsCustomVersion(true);
        }
        hasChanges = true;
      }

      // After all parameters are parsed, ensure proper initialization based on runner type
      if (newFormData.RunnerType === 'DigitalOcean') {
        // If DigitalOcean is selected but no regional configs were provided, initialize defaults
        if (!newFormData.ChainConfig.RegionConfigs || newFormData.ChainConfig.RegionConfigs.length === 0) {
          const defaultRegions = ['nyc1', 'sfo2', 'ams3', 'fra1', 'sgp1'];
          newFormData.ChainConfig.RegionConfigs = defaultRegions.map(region => ({
            name: region,
            numOfNodes: 0,
            numOfValidators: 0,
          }));
          hasChanges = true;
          console.log("Initialized default regional configs for DigitalOcean");
        }
      } else if (newFormData.RunnerType === 'Docker') {
        // For Docker, clear any regional configs and ensure single values are set
        newFormData.ChainConfig.RegionConfigs = [];
        // If numOfNodes wasn't set from URL, ensure it has a default
        if (!params.get('numOfNodes')) {
          newFormData.ChainConfig.NumOfNodes = 0;
        }
        // If numOfValidators wasn't set from URL, ensure it has a default
        if (!params.get('numOfValidators')) {
          newFormData.ChainConfig.NumOfValidators = 0;
        }
        hasChanges = true;
        console.log("Configured for Docker deployment");
      }

      // Update form data only if there were changes
      if (hasChanges) {
        console.log("Updating form data with:", newFormData);
        setFormData(newFormData);
      } else {
        console.log("No changes to form data");
      }
    }
  }, [location.search]); // Only run when URL parameters change

  const [isLoadTestModalOpen, setIsLoadTestModalOpen] = useState(false);
  const [hasLoadTest, setHasLoadTest] = useState(false);

  // Chain configurations modal state
  const [isChainConfigsModalOpen, setIsChainConfigsModalOpen] = useState(false);

  // Genesis modifications modal state
  const [isGenesisModsModalOpen, setIsGenesisModsModalOpen] = useState(false);
  
  // Track when custom version mode is active
  const [isCustomVersion, setIsCustomVersion] = useState(false);

  const createWorkflowMutation = useMutation({
    mutationFn: workflowApi.createWorkflow,
    onSuccess: (data) => {
      toast({
        title: 'Workflow created',
        description: `Workflow ID: ${data.WorkflowID}`,
        status: 'success',
        duration: 5000,
      });
      navigate(`/workflow/${data.WorkflowID}`);
    },
    onError: (error) => {
      toast({
        title: 'Error creating workflow',
        description: error instanceof Error ? error.message : 'Unknown error occurred',
        status: 'error',
        duration: 5000,
      });
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const validationError = validateRequest(formData);
    if (validationError) {
      toast({ title: 'Validation Error', description: validationError, status: 'error', duration: 5000 });
      return;
    }

    const submissionData: TestnetWorkflowRequest = {
      Repo: formData.Repo,
      SHA: formData.SHA,
      ChainConfig: {
        Name: formData.ChainConfig.Name,
        Image: formData.ChainConfig.Image,
        Version: formData.ChainConfig.Version,
        NumOfNodes: formData.ChainConfig.NumOfNodes,
        NumOfValidators: formData.ChainConfig.NumOfValidators,
        GenesisModifications: formData.ChainConfig.GenesisModifications || [],
        RegionConfigs: formData.RunnerType === 'DigitalOcean' ? formData.ChainConfig.RegionConfigs : [],
        AppConfig: formData.ChainConfig.AppConfig,
        ConsensusConfig: formData.ChainConfig.ConsensusConfig,
        ClientConfig: formData.ChainConfig.ClientConfig,
        SetSeedNode: formData.ChainConfig.SetSeedNode,
        SetPersistentPeers: formData.ChainConfig.SetPersistentPeers,
      },
      IsEvmChain: formData.IsEvmChain || false,
      RunnerType: formData.RunnerType,
      LoadTestSpec: formData.LoadTestSpec,
      EthereumLoadTestSpec: formData.LoadTestSpec?.kind === 'eth' ? formData.LoadTestSpec : undefined,
      CosmosLoadTestSpec: formData.LoadTestSpec?.kind === 'cosmos' ? formData.LoadTestSpec : undefined,
      EncodedLoadTestSpec: formData.EncodedLoadTestSpec,
      LongRunningTestnet: formData.LongRunningTestnet,
      LaunchLoadBalancer: formData.LaunchLoadBalancer,
      TestnetDuration: formData.TestnetDuration,
      NumWallets: formData.NumWallets,
    };
    createWorkflowMutation.mutate(submissionData);
  };
  
  const handleLoadTestSave = (loadTestSpec: LoadTestSpec) => {
    setFormData({
      ...formData,
      LoadTestSpec: loadTestSpec,
    });
    setHasLoadTest(true);
  };

  const handleDeleteLoadTest = () => {
    setFormData({
      ...formData,
      LoadTestSpec: undefined
    });
    setHasLoadTest(false);
  };

  const handleChainConfigsSave = (configs: {
    appConfig?: any;
    consensusConfig?: any;
    clientConfig?: any;
  }) => {
    setFormData({
      ...formData,
      ChainConfig: {
        ...formData.ChainConfig,
        AppConfig: configs.appConfig,
        ConsensusConfig: configs.consensusConfig,
        ClientConfig: configs.clientConfig,
      },
    });
  };

  // --- JSON Mode helpers ---
  const parseWorkflowJson = (text: string): TestnetWorkflowRequest => {
    const raw = JSON.parse(text);
    
    // Simplified parsing - use the structure as-is from the JSON
    const repo = raw.repo || raw.Repo || '';
    const isEvmChain = raw.isEvmChain ?? raw.IsEvmChain ?? false;
    
    // Determine correct image based on repository
    let image = raw.chain_config?.image || raw.ChainConfig?.Image || '';
    if (!image) {
      if (repo === 'evm') {
        image = 'evm';
      } else if (repo === 'cometbft') {
        image = 'simapp';
      } else if (repo === 'cosmos-sdk') {
        image = 'simapp';
      } else if (repo === 'gaia') {
        image = 'gaia';
      }
    }
    
    return {
      Repo: repo,
      SHA: raw.sha || raw.SHA || '',
      IsEvmChain: isEvmChain,
      RunnerType: raw.runner_type || raw.RunnerType || '',
      ChainConfig: {
        Name: raw.chain_config?.name || raw.ChainConfig?.Name || '',
        Image: image,
        Version: raw.chain_config?.version || raw.ChainConfig?.Version,
        NumOfNodes: raw.chain_config?.num_of_nodes ?? raw.ChainConfig?.NumOfNodes ?? 0,
        NumOfValidators: raw.chain_config?.num_of_validators ?? raw.ChainConfig?.NumOfValidators ?? 0,
        RegionConfigs: (raw.chain_config?.region_configs || raw.ChainConfig?.RegionConfigs || []).map((rc: any) => ({
          name: rc.name,
          numOfNodes: rc.num_of_nodes ?? rc.numOfNodes ?? 0,
          numOfValidators: rc.num_of_validators ?? rc.numOfValidators ?? 0
        })),
        GenesisModifications: raw.chain_config?.genesis_modifications || raw.ChainConfig?.GenesisModifications || [],
        AppConfig: raw.chain_config?.custom_app_config || raw.ChainConfig?.AppConfig,
        ConsensusConfig: raw.chain_config?.custom_consensus_config || raw.ChainConfig?.ConsensusConfig,
        ClientConfig: raw.chain_config?.custom_client_config || raw.ChainConfig?.ClientConfig,
        SetSeedNode: raw.chain_config?.set_seed_node ?? raw.ChainConfig?.SetSeedNode ?? false,
        SetPersistentPeers: raw.chain_config?.set_persistent_peers ?? raw.ChainConfig?.SetPersistentPeers ?? false,
      },
      LoadTestSpec: raw.load_test_spec || raw.LoadTestSpec,
      EthereumLoadTestSpec: raw.ethereum_load_test_spec || raw.EthereumLoadTestSpec,
      CosmosLoadTestSpec: raw.cosmos_load_test_spec || raw.CosmosLoadTestSpec,
      EncodedLoadTestSpec: raw.encoded_load_test_spec || raw.EncodedLoadTestSpec,
      LongRunningTestnet: raw.long_running_testnet ?? raw.LongRunningTestnet ?? false,
      LaunchLoadBalancer: raw.launch_load_balancer ?? raw.LaunchLoadBalancer ?? false,
      TestnetDuration: raw.testnet_duration || raw.TestnetDuration || '',
      NumWallets: raw.num_wallets || raw.NumWallets || 2500,
      CatalystVersion: raw.catalyst_version || raw.CatalystVersion || '',
    };
  };

  const handleJsonSubmit = async () => {
    setIsParsing(true);
    try {
      const req = parseWorkflowJson(jsonInput);
      const err = validateRequest(req);
      if (err) {
        toast({ title: 'Validation Error', description: err, status: 'error', duration: 5000 });
        return;
      }
      createWorkflowMutation.mutate(req);
    } catch (e: any) {
      toast({ title: 'Invalid JSON', description: e?.message || 'Failed to parse JSON', status: 'error', duration: 6000 });
    } finally {
      setIsParsing(false);
    }
  };

  return (
    <>
      <Box as="form" onSubmit={handleSubmit}>
        <Heading mb={4}>Create New Testnet</Heading>
        <HStack mb={4} align="center" spacing={4}>
          <Text color="text">JSON Mode</Text>
          <Switch isChecked={jsonMode} onChange={(e) => setJsonMode(e.target.checked)} />
        </HStack>
        {jsonMode ? (
          <Stack direction="column" gap={4}>
            <FormControl>
              <FormLabel color="text">Workflow JSON</FormLabel>
              <Textarea
                value={jsonInput}
                onChange={(e) => setJsonInput(e.target.value)}
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder="Paste JSON similar to hack/create-workflow.json"
                minH="80vh"
                maxH="90vh"
                resize="vertical"
                fontFamily="mono"
                fontSize="md"
                lineHeight="1.5"
                w="100%"
              />
            </FormControl>
            <Button colorScheme="blue" isLoading={isParsing || createWorkflowMutation.isPending} onClick={handleJsonSubmit}>
              Create Testnet
            </Button>
          </Stack>
        ) : (
        <Stack direction="column" gap={4}>
          <FormControl isRequired>
            <FormLabel color="text">Repository</FormLabel>
              <Select
                value={formData.Repo}
                onChange={(e) => {
                  const newRepo = e.target.value;
                  const updatedFormData = { 
                    ...formData, 
                    Repo: newRepo,
                    IsEvmChain: newRepo === 'evm'
                  };

                  // Set appropriate image and version based on repository
                  if (newRepo === 'cometbft') {
                    updatedFormData.ChainConfig = {
                      ...updatedFormData.ChainConfig,
                      Version: '', // Default to empty for CometBFT
                      Image: 'simapp'
                    };
                  } else if (newRepo === 'cosmos-sdk') {
                    updatedFormData.ChainConfig = {
                      ...updatedFormData.ChainConfig,
                      Version: undefined,
                      Image: 'simapp'
                    };
                  } else if (newRepo === 'evm') {
                    updatedFormData.ChainConfig = {
                      ...updatedFormData.ChainConfig,
                      Version: undefined,
                      Image: 'evm'
                    };
                  } else if (newRepo === 'gaia') {
                    updatedFormData.ChainConfig = {
                      ...updatedFormData.ChainConfig,
                      Version: undefined,
                      Image: 'gaia'
                    };
                  } else {
                    // Fallback for unknown repos
                    updatedFormData.ChainConfig = {
                      ...updatedFormData.ChainConfig,
                      Version: undefined,
                      Image: ''
                    };
                  }
                                    
                  // Update LoadTestSpec kind if it exists based on new repository
                  if (updatedFormData.LoadTestSpec) {
                    updatedFormData.LoadTestSpec = {
                      ...updatedFormData.LoadTestSpec,
                      kind: newRepo === 'evm' ? 'eth' : 'cosmos'
                    };
                  }
                  
                  setFormData(updatedFormData);
                }}
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder="Select repository"
              >
              <option value="cosmos-sdk">Cosmos SDK</option>
              <option value="cometbft">CometBFT</option>
              <option value="gaia">Gaia</option>
              <option value="evm">EVM</option>
            </Select>
          </FormControl>

          <FormControl isRequired>
            <FormLabel color="text">Commit SHA</FormLabel>
              <Input
                value={formData.SHA}
                onChange={(e) => setFormData({ ...formData, SHA: e.target.value })}
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder="Enter commit SHA"
              />
          </FormControl>

          <FormControl isRequired>
            <FormLabel color="text">Chain Name</FormLabel>
              <Input
                value={formData.ChainConfig.Name}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    ChainConfig: {
                      ...formData.ChainConfig,
                      Name: e.target.value,
                    },
                  })
                }
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder={defaultValues.ChainConfig.Name}
              />
          </FormControl>

                    <FormControl isRequired>
            <FormLabel color="text">Runner Type</FormLabel>
              <Select
                value={formData.RunnerType}
                onChange={(e) => {
                  const newRunnerType = e.target.value;
                  let updatedFormData = { ...formData, RunnerType: newRunnerType };

                  if (newRunnerType === 'DigitalOcean') {
                    // Initialize regional configs for DigitalOcean
                    const defaultRegions = ['nyc1', 'sfo2', 'ams3', 'fra1', 'sgp1'];
                    const regionConfigs = defaultRegions.map(region => ({
                      name: region,
                      numOfNodes: 0,
                      numOfValidators: 0,
                    }));
                    updatedFormData.ChainConfig.RegionConfigs = regionConfigs;
                  } else if (newRunnerType === 'Docker') {
                    // Clear regional configs for Docker and ensure single-region values exist
                    updatedFormData.ChainConfig.RegionConfigs = [];
                    // Set default values if not already set
                    if (updatedFormData.ChainConfig.NumOfNodes === undefined) {
                      updatedFormData.ChainConfig.NumOfNodes = 0;
                    }
                    if (updatedFormData.ChainConfig.NumOfValidators === undefined) {
                      updatedFormData.ChainConfig.NumOfValidators = 0;
                    }
                  }

                  setFormData(updatedFormData);
                }}
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder="Select runner type"
              >
              <option value="Docker">Docker</option>
              <option value="DigitalOcean">DigitalOcean</option>
            </Select>
          </FormControl>

          {(formData.Repo === 'cometbft') && (
            <>
              <FormControl isRequired>
                <FormLabel color="text">Simapp Version</FormLabel>
                  <Select
                    value={isCustomVersion ? 'custom' : (formData.ChainConfig.Version || '')}
                    onChange={(e) => {
                      const selectedValue = e.target.value;
                      
                      if (selectedValue === 'custom') {
                        // Only reset Version to empty if we're switching FROM a predefined version TO custom
                        const shouldResetVersion = !isCustomVersion;
                        setIsCustomVersion(true);
                        setFormData({
                          ...formData,
                          ChainConfig: {
                            ...formData.ChainConfig,
                            Image: 'simapp',
                            Version: shouldResetVersion ? '' : formData.ChainConfig.Version,
                          },
                        });
                      } else {
                        setIsCustomVersion(false);
                        setFormData({
                          ...formData,
                          ChainConfig: {
                            ...formData.ChainConfig,
                            Image: 'simapp',
                            Version: selectedValue,
                          },
                        });
                      }
                    }}
                    bg="surface"
                    color="text"
                    borderColor="divider"
                    placeholder="Select simapp version"
                  >
                  <option value="v0.47.17">Simapp v0.47.17</option>
                  <option value="v0.50.13">Simapp v0.50.13</option>
                  <option value="v0.53.0">Simapp v0.53.0</option>
                  <option value="custom">Custom Version/SHA</option>
                </Select>
              </FormControl>

              {isCustomVersion && (
                <FormControl isRequired>
                  <FormLabel color="text">Custom Simapp Version/SHA</FormLabel>
                  <Input
                    value={formData.ChainConfig.Version || ''}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        ChainConfig: {
                          ...formData.ChainConfig,
                          Version: e.target.value,
                        },
                      })
                    }
                    bg="surface"
                    color="text"
                    borderColor="divider"
                    placeholder="Enter cosmos-sdk version/SHA for simapp build (e.g., v0.50.0 or commit SHA)"
                  />
                  <FormHelperText>
                    This version/SHA will be used to build simapp from cosmos-sdk at the specified commit
                  </FormHelperText>
                </FormControl>
              )}
            </>
          )}



          {formData.RunnerType === 'DigitalOcean' && (
            <Box
              p={4}
              border="1px solid"
              borderColor="divider"
              borderRadius="md"
              bg="surface"
            >
              <VStack spacing={4} align="stretch">
                <HStack justify="space-between">
                  <Text fontWeight="bold" color="text">Region Configuration</Text>
                  <Text fontSize="sm" color="textSecondary">
                    Total: {formData.ChainConfig.RegionConfigs?.reduce((sum, rc) => sum + rc.numOfNodes, 0) || 0} nodes, {formData.ChainConfig.RegionConfigs?.reduce((sum, rc) => sum + rc.numOfValidators, 0) || 0} validators
                  </Text>
                </HStack>

                <VStack spacing={3} align="stretch">
                  {['nyc1', 'sfo2', 'ams3', 'fra1', 'sgp1'].map((region) => {
                    const regionLabels: Record<string, string> = {
                      'nyc1': 'New York (nyc1)',
                      'sfo2': 'San Francisco (sfo2)',
                      'ams3': 'Amsterdam (ams3)',
                      'fra1': 'Frankfurt (fra1)',
                      'sgp1': 'Singapore (sgp1)',
                    };
                    const config = formData.ChainConfig.RegionConfigs?.find(rc => rc.name === region) || { name: region, numOfNodes: 0, numOfValidators: 0 };

                    return (
                      <Box key={region} p={3} border="1px solid" borderColor="divider" borderRadius="md">
                        <HStack spacing={4}>
                          <Text fontWeight="bold" fontSize="sm" minW="150px" color="text">
                            {regionLabels[region]}
                          </Text>

                          <FormControl flex="1">
                            <FormLabel fontSize="xs">Nodes</FormLabel>
                            <NumberInput
                              value={config.numOfNodes}
                              min={0}
                              size="sm"
                              onChange={(_, value) => {
                                const updatedConfigs = formData.ChainConfig.RegionConfigs?.map(rc =>
                                  rc.name === region ? { ...rc, numOfNodes: value || 0 } : rc
                                ) || [];
                                setFormData({
                                  ...formData,
                                  ChainConfig: {
                                    ...formData.ChainConfig,
                                    RegionConfigs: updatedConfigs,
                                  },
                                });
                              }}
                            >
                              <NumberInputField />
                            </NumberInput>
                          </FormControl>

                          <FormControl flex="1">
                            <FormLabel fontSize="xs">Validators</FormLabel>
                            <NumberInput
                              value={config.numOfValidators}
                              min={0}
                              size="sm"
                              onChange={(_, value) => {
                                const updatedConfigs = formData.ChainConfig.RegionConfigs?.map(rc =>
                                  rc.name === region ? { ...rc, numOfValidators: value || 0 } : rc
                                ) || [];
                                setFormData({
                                  ...formData,
                                  ChainConfig: {
                                    ...formData.ChainConfig,
                                    RegionConfigs: updatedConfigs,
                                  },
                                });
                              }}
                            >
                              <NumberInputField />
                            </NumberInput>
                          </FormControl>

                          <Text fontSize="xs" color="textSecondary" minW="80px">
                            Total: {config.numOfNodes + config.numOfValidators}
                          </Text>
                        </HStack>
                      </Box>
                    );
                  })}
                </VStack>
              </VStack>
            </Box>
          )}

          {formData.RunnerType === 'Docker' && (
            <>
              <FormControl>
                <FormLabel color="text">Number of Nodes</FormLabel>
              <Input
                type="number"
                min={0}
                value={nodeInputValue}
                onChange={(e) => {
                  const value = e.target.value;
                  setNodeInputValue(value);
                  setFormData({
                    ...formData,
                    ChainConfig: {
                      ...formData.ChainConfig,
                      NumOfNodes: value === '' ? 0 : parseInt(value) || 0,
                    },
                  });
                }}
                bg="surface"
                color="text"
                borderColor="divider"
                placeholder={defaultValues.ChainConfig.NumOfNodes?.toString() || "4"}
              />
          </FormControl>

          <FormControl>
            <FormLabel color="text">Number of Validators</FormLabel>
              <NumberInput
                value={formData.ChainConfig.NumOfValidators || ''}
                min={0}
                onChange={(_, value) =>
                  setFormData({
                    ...formData,
                    ChainConfig: {
                      ...formData.ChainConfig,
                      NumOfValidators: value || 0,
                    },
                  })
                }
              >
                <NumberInputField 
                  bg="surface" 
                  color="text" 
                  borderColor="divider" 
                  placeholder={defaultValues.ChainConfig.NumOfValidators?.toString() || "3"}
                />
            </NumberInput>
          </FormControl>
            </>
          )}

          <FormControl>
            <FormLabel>Chain Configurations</FormLabel>
            <VStack spacing={3} align="start">
              <HStack width="100%">
                <Button
                  colorScheme="blue"
                  variant="outline"
                  onClick={() => setIsChainConfigsModalOpen(true)}
                >
                  Set Custom Chain Config
                </Button>
              </HStack>

              {/* Display applied configurations */}
              {(formData.ChainConfig.AppConfig || formData.ChainConfig.ConsensusConfig || formData.ChainConfig.ClientConfig) && (
                <Box width="100%" p={3} bg="green.50" borderRadius="md" border="1px solid" borderColor="green.200">
                  <Text fontWeight="bold" color="green.800" mb={2}>Applied Configurations:</Text>
                  {formData.ChainConfig.AppConfig && (
                    <Text fontSize="sm" color="green.700">• App Config: Applied</Text>
                  )}
                  {formData.ChainConfig.ConsensusConfig && (
                    <Text fontSize="sm" color="green.700">• Consensus Config: Applied</Text>
                  )}
                  {formData.ChainConfig.ClientConfig && (
                    <Text fontSize="sm" color="green.700">• Client Config: Applied</Text>
                  )}
                </Box>
              )}
            </VStack>
          </FormControl>

          <FormControl>
            <FormLabel>Genesis Modifications</FormLabel>
            <VStack spacing={3} align="start">
              <HStack width="100%">
                <Button
                  colorScheme="blue"
                  variant="outline"
                  onClick={() => setIsGenesisModsModalOpen(true)}
                >
                  Set Genesis Modifications
                </Button>
              </HStack>
              
              {/* Display applied modifications */}
              {formData.ChainConfig?.GenesisModifications && formData.ChainConfig.GenesisModifications.length > 0 && (
                <Box width="100%" p={3} bg="green.50" borderRadius="md" border="1px solid" borderColor="green.200">
                  <Text fontWeight="bold" color="green.800" mb={2}>
                    Applied Modifications ({formData.ChainConfig.GenesisModifications.length}):
                  </Text>
                  <VStack align="start" spacing={1} maxH="150px" overflowY="auto">
                    {formData.ChainConfig.GenesisModifications.slice(0, 5).map((mod, index) => (
                      <Text key={index} fontSize="sm" color="green.700">
                        • {mod.key}: {typeof mod.value === 'string' ? mod.value : 'Complex value'}
                      </Text>
                    ))}
                    {formData.ChainConfig.GenesisModifications.length > 5 && (
                      <Text fontSize="sm" color="green.600" fontStyle="italic">
                        ... and {formData.ChainConfig.GenesisModifications.length - 5} more
                      </Text>
                    )}
                  </VStack>
                </Box>
              )}
            </VStack>
          </FormControl>

            <FormControl display="flex" alignItems="center">
              <FormLabel mb="0">Set Seed Node</FormLabel>
              <Tooltip label="Set seed node will set a full node (or validator if no nodes exist) as seed for the network" placement="top">
                <HStack>
                  <Switch
                    isChecked={formData.ChainConfig.SetSeedNode || false}
                    onChange={(e) => setFormData({
                      ...formData,
                      ChainConfig: {
                        ...formData.ChainConfig,
                        SetSeedNode: e.target.checked
                      }
                    })}
                  />
                  <InfoIcon />
                </HStack>
              </Tooltip>
            </FormControl>

            <FormControl display="flex" alignItems="center">
              <FormLabel mb="0">Set Persistent Peers</FormLabel>
              <Tooltip label="Set persistent peers will add all nodes and validators of the network as persistent peers to the consensus config" placement="top">
                <HStack>
                  <Switch
                    isChecked={formData.ChainConfig.SetPersistentPeers || false}
                    onChange={(e) => setFormData({
                      ...formData,
                      ChainConfig: {
                        ...formData.ChainConfig,
                        SetPersistentPeers: e.target.checked
                      }
                    })}
                  />
                  <InfoIcon />
                </HStack>
              </Tooltip>
            </FormControl>

          <FormControl display="flex" alignItems="center">
            <FormLabel mb="0">Run Load Test</FormLabel>
            <Switch
              isChecked={hasLoadTest}
              onChange={(e) => {
                if (!e.target.checked) {
                  setFormData({
                    ...formData,
                    LoadTestSpec: undefined
                  });
                  setHasLoadTest(false);
                } else {
                  setFormData({ 
                    ...formData, 
                    LoadTestSpec: {
                      name: 'basic-load-test',
                      description: 'Basic load test configuration',
                      chain_id: 'test-chain',
                      kind: formData.Repo === 'evm' ? 'eth' : 'cosmos',
                      NumOfBlocks: 100,
                      NumOfTxs: 1000,
                      msgs: [],
                      unordered_txs: false,
                      tx_timeout: '',
                      send_interval: '',
                      num_batches: 0,
                      gas_denom: '',
                      bech32_prefix: '',
                    }
                  });
                  setHasLoadTest(true);
                }
              }}
            />
          </FormControl>
          
          {hasLoadTest && (
            <Button 
              colorScheme="blue" 
              variant="outline" 
              onClick={() => setIsLoadTestModalOpen(true)}
              leftIcon={<AddIcon />}
            >
              Configure Load Test
            </Button>
          )}
          
          {hasLoadTest && formData.LoadTestSpec && formData.LoadTestSpec.msgs.length > 0 && (
            <Box p={3} bg="surface" borderRadius="md" boxShadow="sm">
              <Flex justify="space-between" align="center" mb={2}>
                <Text fontWeight="bold" color="text">Load Test Configuration</Text>
                <IconButton
                  aria-label="Delete load test configuration"
                  icon={<DeleteIcon />}
                  size="sm"
                  colorScheme="red"
                  variant="ghost"
                  onClick={handleDeleteLoadTest}
                />
              </Flex>
              <Text color="text">Name: {formData.LoadTestSpec.name}</Text>
              <Text color="text">Type: {formData.LoadTestSpec.kind === 'eth' ? 'Ethereum' : 'Cosmos'}</Text>
              {formData.LoadTestSpec.kind === 'cosmos' && (
                <>
                  <Text color="text">Transactions: {formData.LoadTestSpec.NumOfTxs}</Text>
                  <Text color="text">Blocks: {formData.LoadTestSpec.NumOfBlocks}</Text>
                </>
              )}
              {formData.LoadTestSpec.kind === 'eth' && (
                <>
                  {formData.LoadTestSpec.send_interval && (
                    <Text color="text">Send Interval: {formData.LoadTestSpec.send_interval}</Text>
                  )}
                  {formData.LoadTestSpec.num_batches && (
                    <Text color="text">Batches: {formData.LoadTestSpec.num_batches}</Text>
                  )}
                </>
              )}
              {formData.LoadTestSpec.unordered_txs && (
                <>
                  <Text color="text">Unordered Transactions: Yes</Text>
                  <Text color="text">Transaction Timeout: {formData.LoadTestSpec.tx_timeout}</Text>
                </>
              )}
              <Text color="text">Message Types:</Text>
              {formData.LoadTestSpec.msgs.length > 0 ? (
                <VStack align="start" pl={4} spacing={1}>
                  {formData.LoadTestSpec.msgs.map((msg, idx) => {
                    // Get all set properties for this message
                    const pairs = Object.entries(msg)
                      .filter(([_, value]) => value !== undefined && value !== '')
                      .map(([key, value]) => `${key}: ${value}`);
                    
                    return (
                      <Text key={idx} fontSize="sm" color="textSecondary">
                        • {pairs.join(' | ')}
                      </Text>
                    );
                  })}
                </VStack>
              ) : (
                <Text pl={4} color="gray.500" fontSize="sm">No message types configured</Text>
              )}
            </Box>
          )}
          
          <FormControl display="flex" alignItems="center">
            <FormLabel mb="0">Long Running Testnet</FormLabel>
            <Switch
              isChecked={formData.LongRunningTestnet}
              onChange={(e) => setFormData({ ...formData, LongRunningTestnet: e.target.checked })}
            />
          </FormControl>

          <FormControl display="flex" alignItems="center">
            <FormLabel mb="0">Launch Load Balancer</FormLabel>
            <Switch
              isChecked={formData.LaunchLoadBalancer}
              isDisabled={formData.RunnerType === 'Docker'}
              onChange={(e) => setFormData({ ...formData, LaunchLoadBalancer: e.target.checked })}
            />
            <Tooltip label="Launch a load balancer for the testnet (only available for DigitalOcean runner)">
              <InfoIcon ml={2} cursor="pointer" />
            </Tooltip>
          </FormControl>

          {!formData.LongRunningTestnet && (
            <FormControl>
              <FormLabel>Testnet Duration (Hours)</FormLabel>
              <Input
                type="text"
                value={formData.TestnetDuration}
                onChange={(e) => setFormData({
                  ...formData,
                  TestnetDuration: e.target.value
                })}
                placeholder={defaultValues.TestnetDuration}
              />
              <FormHelperText>Duration in hours (e.g., "2h", "1.5h", "30m"). Default runtime if left blank is 2m.</FormHelperText>
            </FormControl>
          )}

          <FormControl>
            <FormLabel>Number of Wallets</FormLabel>
            <NumberInput
              min={1}
              value={formData.NumWallets || ''}
              onChange={(_, value) => {
                setFormData({ 
                  ...formData, 
                  NumWallets: value || 2500
                });
              }}
            >
              <NumberInputField placeholder="2500" />
            </NumberInput>
            <FormHelperText>Number of wallets to create</FormHelperText>
          </FormControl>

          <FormControl>
            <FormLabel>Catalyst Version</FormLabel>
            <Input
              type="text"
              value={formData.CatalystVersion || ''}
              onChange={(e) => setFormData({
                ...formData,
                CatalystVersion: e.target.value
              })}
              placeholder="latest"
              bg="surface"
              color="text"
              borderColor="divider"
            />
            <FormHelperText>Docker tag for catalyst image (e.g., "main", "v1.2.3"). cDefaults to "latest" if empty.</FormHelperText>
          </FormControl>

          <Button
            mt={4}
            colorScheme="blue"
            disabled={createWorkflowMutation.isPending}
            type="submit"
          >
            Create Testnet
          </Button>
        </Stack>
        )}
      </Box>
      
      <LoadTestForm
        isOpen={isLoadTestModalOpen}
        onClose={() => setIsLoadTestModalOpen(false)}
        initialData={formData.LoadTestSpec || {
          name: 'basic-load-test',
          description: 'Basic load test configuration',
          chain_id: formData.ChainConfig.Name || 'test-chain',
          kind: formData.Repo === 'evm' ? 'eth' : 'cosmos',
          NumOfBlocks: 100,
          NumOfTxs: 1000,
          msgs: [],
          unordered_txs: false,
          tx_timeout: '',
          send_interval: '',
          num_batches: 0,
          gas_denom: '',
          bech32_prefix: '',
        }}
        selectedRepo={formData.Repo}
        onSave={handleLoadTestSave}
      />
      
      <ChainConfigsModal
        isOpen={isChainConfigsModalOpen}
        onClose={() => setIsChainConfigsModalOpen(false)}
        initialAppConfig={formData.ChainConfig.AppConfig}
        initialConsensusConfig={formData.ChainConfig.ConsensusConfig}
        initialClientConfig={formData.ChainConfig.ClientConfig}
        onSave={handleChainConfigsSave}
      />

      <GenesisModificationsModal
        isOpen={isGenesisModsModalOpen}
        onClose={() => setIsGenesisModsModalOpen(false)}
        initialModifications={formData.ChainConfig.GenesisModifications}
        onSave={(newModifications) => setFormData({
          ...formData,
          ChainConfig: {
            ...formData.ChainConfig,
            GenesisModifications: newModifications,
          },
        })}
      />
    </>
  );
};

export default CreateWorkflow;

package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/skip-mev/ironbird/petri/cosmos/node"
	"github.com/skip-mev/ironbird/petri/cosmos/wallet"
	"google.golang.org/protobuf/encoding/protojson"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	basev1beta1 "cosmossdk.io/api/cosmos/base/v1beta1"
	sdkmath "cosmossdk.io/math"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/skip-mev/ironbird/petri/core/provider"
	petritypes "github.com/skip-mev/ironbird/petri/core/types"
)

type PackagedState struct {
	State
	ValidatorStates  [][]byte
	NodeStates       [][]byte
	ValidatorWallets []string
	FaucetWallet     string
}

type State struct {
	Config petritypes.ChainConfig
}

// Chain is a logical representation of a Cosmos-based blockchain
type Chain struct {
	State State

	logger *zap.Logger

	Validators []petritypes.NodeI
	Nodes      []petritypes.NodeI

	FaucetWallet petritypes.WalletI

	ValidatorWallets []petritypes.WalletI

	mu sync.RWMutex

	// useExternalAddresses determines whether to use external addresses (DigitalOcean)
	// or internal addresses (Docker) for peer strings
	useExternalAddresses bool
}

var _ petritypes.ChainI = &Chain{}

// CreateChain creates the Chain object and initializes the node tasks, their backing compute and the validator wallets
func CreateChain(ctx context.Context, logger *zap.Logger, infraProvider provider.ProviderI, config petritypes.ChainConfig, opts petritypes.ChainOptions) (*Chain, error) {
	providerType := infraProvider.GetType()
	if err := config.ValidateBasic(providerType); err != nil {
		return nil, fmt.Errorf("failed to validate chain config: %w", err)
	}

	if err := opts.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("failed to validate chain options: %w", err)
	}

	var chain Chain

	chain.mu = sync.RWMutex{}
	chain.State = State{
		Config: config,
	}
	chain.useExternalAddresses = providerType == petritypes.DigitalOcean

	chain.logger = logger.Named("chain").With(zap.String("chain_id", config.ChainId))
	chain.logger.Info("creating chain")

	var validators, nodes []petritypes.NodeI
	var err error
	if providerType == petritypes.DigitalOcean {
		validators, nodes, err = createRegionalNodes(ctx, logger, &chain, infraProvider, config, opts)
	} else {
		validators, nodes, err = createLocalNodes(ctx, logger, &chain, infraProvider, config, opts)
	}
	if err != nil {
		return nil, err
	}

	logger.Info("created nodes", zap.Int("validators", len(validators)), zap.Int("nodes", len(nodes)))

	chain.Nodes = nodes
	chain.Validators = validators
	chain.ValidatorWallets = make([]petritypes.WalletI, len(validators))

	return &chain, nil
}

func createRegionalNodes(ctx context.Context, logger *zap.Logger, chain *Chain, infraProvider provider.ProviderI,
	config petritypes.ChainConfig, opts petritypes.ChainOptions,
) ([]petritypes.NodeI, []petritypes.NodeI, error) {
	var eg errgroup.Group
	validators := make([]petritypes.NodeI, 0)
	nodes := make([]petritypes.NodeI, 0)

	validatorIndex := 0
	nodeIndex := 0

	for _, regionConfig := range config.RegionConfig {
		region := regionConfig

		for i := 0; i < region.NumValidators; i++ {
			currentValidatorIndex := validatorIndex
			currentRegion := region.Name
			validatorName := fmt.Sprintf("%s-validator-%d-%s", config.Name, currentValidatorIndex, currentRegion)
			eg.Go(func() error {
				validator, err := opts.NodeCreator(ctx, logger, infraProvider, petritypes.NodeConfig{
					Index:       currentValidatorIndex,
					Name:        validatorName,
					ChainConfig: config,
				}, createRegionalNodeOptions(opts.NodeOptions, region))
				if err != nil {
					return err
				}

				chain.mu.Lock()
				validators = append(validators, validator)
				chain.mu.Unlock()
				return nil
			})
			validatorIndex++
		}

		for i := 0; i < region.NumNodes; i++ {
			currentNodeIndex := nodeIndex
			currentRegion := region.Name
			nodeName := fmt.Sprintf("%s-node-%d-%s", config.Name, currentNodeIndex, currentRegion)
			eg.Go(func() error {
				node, err := opts.NodeCreator(ctx, logger, infraProvider, petritypes.NodeConfig{
					Index:       currentNodeIndex,
					Name:        nodeName,
					ChainConfig: config,
				}, createRegionalNodeOptions(opts.NodeOptions, region))
				if err != nil {
					return err
				}

				chain.mu.Lock()
				nodes = append(nodes, node)
				chain.mu.Unlock()
				return nil
			})
			nodeIndex++
		}
	}

	if err := eg.Wait(); err != nil {
		logger.Error("error creating regional nodes", zap.Error(err))
		return nil, nil, err
	}

	return validators, nodes, nil
}

func createLocalNodes(ctx context.Context, logger *zap.Logger, chain *Chain, infraProvider provider.ProviderI,
	config petritypes.ChainConfig, opts petritypes.ChainOptions,
) ([]petritypes.NodeI, []petritypes.NodeI, error) {
	var eg errgroup.Group
	validators := make([]petritypes.NodeI, 0)
	nodes := make([]petritypes.NodeI, 0)

	for i := 0; i < config.NumValidators; i++ {
		currentValidatorIndex := i
		validatorName := fmt.Sprintf("%s-validator-%d", config.Name, currentValidatorIndex)
		eg.Go(func() error {
			validator, err := opts.NodeCreator(ctx, logger, infraProvider, petritypes.NodeConfig{
				Index:       currentValidatorIndex,
				Name:        validatorName,
				ChainConfig: config,
			}, opts.NodeOptions)
			if err != nil {
				return err
			}

			chain.mu.Lock()
			validators = append(validators, validator)
			chain.mu.Unlock()
			return nil
		})
	}

	for i := 0; i < config.NumNodes; i++ {
		currentNodeIndex := i
		nodeName := fmt.Sprintf("%s-node-%d", config.Name, currentNodeIndex)
		eg.Go(func() error {
			node, err := opts.NodeCreator(ctx, logger, infraProvider, petritypes.NodeConfig{
				Index:       currentNodeIndex,
				Name:        nodeName,
				ChainConfig: config,
			}, opts.NodeOptions)
			if err != nil {
				return err
			}

			chain.mu.Lock()
			nodes = append(nodes, node)
			chain.mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		logger.Error("error creating local nodes", zap.Error(err))
		return nil, nil, err
	}

	return validators, nodes, nil
}

func createRegionalNodeOptions(baseOpts petritypes.NodeOptions, region petritypes.RegionConfig) petritypes.NodeOptions {
	applyRegionConfig := func(definition provider.TaskDefinition) provider.TaskDefinition {
		if definition.ProviderSpecificConfig == nil {
			definition.ProviderSpecificConfig = make(map[string]string)
		}
		definition.ProviderSpecificConfig["region"] = region.Name
		definition.ProviderSpecificConfig["image_id"] = "210084437"
		if _, ok := definition.ProviderSpecificConfig["size"]; !ok {
			definition.ProviderSpecificConfig["size"] = "s-4vcpu-8gb"
		}
		return definition
	}

	if baseOpts.NodeDefinitionModifier == nil {
		baseOpts.NodeDefinitionModifier = func(definition provider.TaskDefinition, nodeConfig petritypes.NodeConfig) provider.TaskDefinition {
			return applyRegionConfig(definition)
		}
	} else {
		originalModifier := baseOpts.NodeDefinitionModifier
		baseOpts.NodeDefinitionModifier = func(definition provider.TaskDefinition, nodeConfig petritypes.NodeConfig) provider.TaskDefinition {
			definition = originalModifier(definition, nodeConfig)
			return applyRegionConfig(definition)
		}
	}
	return baseOpts
}

// RestoreChain restores a Chain object from a serialized state
func RestoreChain(ctx context.Context, logger *zap.Logger, infraProvider provider.ProviderI, state []byte,
	nodeRestore petritypes.NodeRestorer, walletConfig petritypes.WalletConfig,
) (*Chain, error) {
	var packagedState PackagedState

	if err := json.Unmarshal(state, &packagedState); err != nil {
		return nil, err
	}

	chain := Chain{
		State:      packagedState.State,
		logger:     logger,
		Validators: make([]petritypes.NodeI, len(packagedState.ValidatorStates)),
		Nodes:      make([]petritypes.NodeI, len(packagedState.NodeStates)),
	}

	eg := new(errgroup.Group)

	for i, vs := range packagedState.ValidatorStates {
		eg.Go(func() error {
			i := i
			v, err := nodeRestore(ctx, logger, vs, infraProvider)
			if err != nil {
				return err
			}

			chain.Validators[i] = v
			return nil
		})
	}

	for i, ns := range packagedState.NodeStates {
		eg.Go(func() error {
			i := i
			v, err := nodeRestore(ctx, logger, ns, infraProvider)
			if err != nil {
				return err
			}

			chain.Nodes[i] = v
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	chain.ValidatorWallets = make([]petritypes.WalletI, len(packagedState.ValidatorWallets))
	for i, mnemonic := range packagedState.ValidatorWallets {
		w, err := wallet.NewWallet(petritypes.ValidatorKeyName, mnemonic, "", walletConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to restore validator wallet: %w", err)
		}
		chain.ValidatorWallets[i] = w
	}

	if packagedState.FaucetWallet != "" {
		w, err := wallet.NewWallet(petritypes.FaucetAccountKeyName, packagedState.FaucetWallet, "", walletConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to restore faucet wallet: %w", err)
		}
		chain.FaucetWallet = w
	}

	return &chain, nil
}

// Height returns the chain's height from the first available full node in the network
func (c *Chain) Height(ctx context.Context) (uint64, error) {
	node := c.GetFullNode()

	client, err := node.GetTMClient(ctx)
	if err != nil {
		return 0, err
	}

	c.logger.Debug("fetching height from", zap.String("node", node.GetDefinition().Name), zap.String("ip", client.Remote()))

	status, err := client.Status(ctx)
	if err != nil {
		return 0, err
	}

	return uint64(status.SyncInfo.LatestBlockHeight), nil
}

// Init initializes the chain. That consists of generating the genesis transactions, genesis file, wallets,
// the distribution of configuration files and starting the network nodes up
func (c *Chain) Init(ctx context.Context, opts petritypes.ChainOptions) error {
	if err := opts.ValidateBasic(); err != nil {
		return fmt.Errorf("failed to validate chain options: %w", err)
	}

	decimalPow := int64(math.Pow10(int(c.GetConfig().Decimals)))

	genesisCoin := types.Coin{
		Amount: sdkmath.NewIntFromBigInt(c.GetConfig().GetGenesisBalance()).MulRaw(decimalPow),
		Denom:  c.GetConfig().Denom,
	}
	c.logger.Info("creating genesis accounts", zap.String("coin", genesisCoin.String()))

	genesisSelfDelegation := types.Coin{
		Amount: sdkmath.NewIntFromBigInt(c.GetConfig().GetGenesisDelegation()).MulRaw(decimalPow),
		Denom:  c.GetConfig().Denom,
	}
	c.logger.Info("creating genesis self-delegations", zap.String("coin", genesisSelfDelegation.String()))

	genesisAmounts := []types.Coin{genesisCoin}

	eg := new(errgroup.Group)

	for idx, v := range c.Validators {
		v := v
		idx := idx
		eg.Go(func() error {
			c.logger.Info("setting up validator", zap.String("validator", v.GetDefinition().Name))

			validatorWallet, validatorAddress, err := v.SetupValidator(ctx, opts.WalletConfig, genesisAmounts, genesisSelfDelegation)
			if err != nil {
				return fmt.Errorf("error in validator setup: %v", err)
			}

			c.ValidatorWallets[idx] = validatorWallet

			c.logger.Info("validator setup finished", zap.String("validator", v.GetDefinition().Name), zap.String("address", validatorAddress))

			return nil
		})
	}

	for _, n := range c.Nodes {
		n := n

		eg.Go(func() error {
			c.logger.Info("setting up node", zap.String("node", n.GetDefinition().Name))

			if err := n.SetupNode(ctx); err != nil {
				return err
			}
			c.logger.Info("node setup finished", zap.String("node", n.GetDefinition().Name))

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	c.logger.Info("adding faucet genesis")
	faucetWallet, err := c.BuildWallet(ctx, petritypes.FaucetAccountKeyName, "", opts.WalletConfig)
	if err != nil {
		return err
	}

	c.FaucetWallet = faucetWallet

	firstValidator := c.Validators[0]

	if err := c.executeGenesisOperations(ctx, opts.WalletConfig, firstValidator, faucetWallet, genesisAmounts, additionalAccountOpts{baseMnemonic: opts.BaseMnemonic, numAdditionalAccounts: opts.AdditionalAccounts}); err != nil {
		return err
	}

	genbz, err := firstValidator.GenesisFileContent(ctx)
	if err != nil {
		return err
	}

	if opts.ModifyGenesis != nil {
		c.logger.Info("modifying genesis")
		genbz, err = opts.ModifyGenesis(genbz)
		if err != nil {
			return err
		}
	}

	var (
		chainConfig     = c.GetConfig()
		persistentPeers PeerSet
		seeds           PeerSet
		seedNode        petritypes.NodeI
	)

	if chainConfig.SetSeedNode {
		if len(c.Nodes) > 0 {
			seedNode = c.Nodes[0]
		} else {
			return fmt.Errorf("no nodes available to be used as seed ")
		}

		if seedNode != nil {
			seeds = NewPeerSet([]petritypes.NodeI{seedNode})
		}
	}

	if chainConfig.SetPersistentPeers {
		persistentPeers = NewPeerSet(append(c.Nodes, c.Validators...))
	}

	for i := range c.Validators {
		v := c.Validators[i]
		eg.Go(func() error {
			c.logger.Info("overwriting genesis for validator", zap.String("validator", v.GetDefinition().Name))
			return configureNode(ctx, v, chainConfig, genbz, persistentPeers, seeds, c.useExternalAddresses, c.logger)
		})
	}

	for i := range c.Nodes {
		n := c.Nodes[i]
		eg.Go(func() error {
			c.logger.Info("overwriting node genesis", zap.String("node", n.GetDefinition().Name))
			return configureNode(ctx, n, chainConfig, genbz, persistentPeers, seeds, c.useExternalAddresses, c.logger)
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if chainConfig.SetSeedNode && seedNode != nil {
		c.logger.Info("configuring seed node mode", zap.String("seed_node", seedNode.GetDefinition().Name))
		if err := seedNode.SetSeedMode(ctx); err != nil {
			return fmt.Errorf("failed to set seed mode on %s: %w", seedNode.GetDefinition().Name, err)
		}
	}

	for i := range c.Validators {
		v := c.Validators[i]
		eg.Go(func() error {
			c.logger.Info("starting validator task", zap.String("validator", v.GetDefinition().Name))
			if err := v.Start(ctx); err != nil {
				return err
			}
			return nil
		})
	}

	for i := range c.Nodes {
		n := c.Nodes[i]
		eg.Go(func() error {
			c.logger.Info("starting node task", zap.String("node", n.GetDefinition().Name))
			if err := n.Start(ctx); err != nil {
				return err
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

// Teardown destroys all resources related to a chain and its' nodes
func (c *Chain) Teardown(ctx context.Context) error {
	c.logger.Info("tearing down chain", zap.String("name", c.GetConfig().ChainId))

	for _, v := range c.Validators {
		if err := v.Destroy(ctx); err != nil {
			return err
		}
	}

	for _, n := range c.Nodes {
		if err := n.Destroy(ctx); err != nil {
			return err
		}
	}

	return nil
}

// GetGRPCClient returns a gRPC client of the first available node
func (c *Chain) GetGRPCClient(ctx context.Context) (*grpc.ClientConn, error) {
	return c.GetFullNode().GetGRPCClient(ctx)
}

// GetTMClient returns a CometBFT client of the first available node
func (c *Chain) GetTMClient(ctx context.Context) (*rpchttp.HTTP, error) {
	return c.GetFullNode().GetTMClient(ctx)
}

// GetFullNode returns the first available full node in the chain
func (c *Chain) GetFullNode() petritypes.NodeI {
	if len(c.Nodes) > 0 {
		// use random full node
		return c.Nodes[rand.Intn(len(c.Nodes))]
	}
	// use random validator
	return c.Validators[rand.Intn(len(c.Validators))]
}

func (c *Chain) WaitForStartup(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := c.WaitForHeight(ctx, 1)
			if err != nil {
				c.logger.Error("error waiting for height", zap.Error(err))
				continue
			}
			ticker.Stop()
			return nil
		}
	}
}

// WaitForBlocks blocks until the chain increases in block height by delta
func (c *Chain) WaitForBlocks(ctx context.Context, delta uint64) error {
	c.logger.Info("waiting for blocks", zap.Uint64("delta", delta))

	start, err := c.Height(ctx)
	if err != nil {
		return err
	}

	return c.WaitForHeight(ctx, start+delta)
}

// WaitForHeight blocks until the chain reaches block height desiredHeight
func (c *Chain) WaitForHeight(ctx context.Context, desiredHeight uint64) error {
	c.logger.Info("waiting for height", zap.Uint64("desired_height", desiredHeight))
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			c.logger.Debug("waiting for height", zap.Uint64("desired_height", desiredHeight))

			height, err := c.Height(ctx)
			if err != nil {
				c.logger.Error("failed to get height", zap.Error(err))
				continue
			}

			if height >= desiredHeight {
				return nil
			}

			// We assume the chain will eventually return a non-zero height, otherwise
			// this may block indefinitely.
			if height == 0 {
				continue
			}
		}
	}
}

// GetValidators returns all of the validating nodes in the chain
func (c *Chain) GetValidators() []petritypes.NodeI {
	return c.Validators
}

// GetNodes returns all of the non-validating nodes in the chain
func (c *Chain) GetNodes() []petritypes.NodeI {
	return c.Nodes
}

// GetValidatorWallets returns the wallets that were used to create the Validators on-chain.
// The ordering of the slice should correspond to the ordering of GetValidators
func (c *Chain) GetValidatorWallets() []petritypes.WalletI {
	return c.ValidatorWallets
}

// GetFaucetWallet retunrs a wallet that was funded and can be used to fund other wallets
func (c *Chain) GetFaucetWallet() petritypes.WalletI {
	return c.FaucetWallet
}

// GetConfig is the configuration structure for a logical chain.
func (c *Chain) GetConfig() petritypes.ChainConfig {
	return c.State.Config
}

// Serialize returns the serialized representation of the chain
func (c *Chain) Serialize(ctx context.Context, p provider.ProviderI) ([]byte, error) {
	state := PackagedState{
		State: c.State,
	}

	for _, v := range c.Validators {
		vs, err := v.Serialize(ctx, p)
		if err != nil {
			return nil, err
		}
		state.ValidatorStates = append(state.ValidatorStates, vs)
	}

	for _, n := range c.Nodes {
		ns, err := n.Serialize(ctx, p)
		if err != nil {
			return nil, err
		}

		state.NodeStates = append(state.NodeStates, ns)
	}

	for _, w := range c.ValidatorWallets {
		state.ValidatorWallets = append(state.ValidatorWallets, w.Mnemonic())
	}

	if c.FaucetWallet != nil {
		state.FaucetWallet = c.FaucetWallet.Mnemonic()
	}

	return json.Marshal(state)
}

type additionalAccountOpts struct {
	baseMnemonic          string
	numAdditionalAccounts int
}

const accountType = "/cosmos.auth.v1beta1.BaseAccount"

type Account struct {
	Type          string      `json:"@type"`
	Address       string      `json:"address"`
	PubKey        interface{} `json:"pub_key"`
	AccountNumber string      `json:"account_number"`
	Sequence      string      `json:"sequence"`
}

type Balance struct {
	Address string      `json:"address"`
	Coins   types.Coins `json:"coins"`
}

// Executes the needed genesis operations for the chain:
// 1. Add the faucet account to the genesis file
// 2. Add the validator accounts to the genesis file
// 3. Collect the gentxs from the validators and create the genesis file
func (c *Chain) executeGenesisOperations(ctx context.Context, walletCfg petritypes.WalletConfig, firstValidator petritypes.NodeI, faucetWallet petritypes.WalletI, genesisAmounts []types.Coin, accountOpts additionalAccountOpts) error {
	c.logger.Info("executing genesis operations", zap.String("validator", firstValidator.GetDefinition().Name))

	var scriptBuilder strings.Builder
	scriptBuilder.WriteString("#!/bin/sh\nset -e\n")
	useGenesisSubCommand := c.GetConfig().UseGenesisSubCommand

	var faucetAmounts []string
	for _, coin := range genesisAmounts {
		faucetAmounts = append(faucetAmounts, fmt.Sprintf("%s%s", coin.Amount.String(), coin.Denom))
	}
	faucetAmount := strings.Join(faucetAmounts, ",")

	firstValidatorNode := firstValidator.(*node.Node)

	var addFaucetCmd []string
	if useGenesisSubCommand {
		addFaucetCmd = append(addFaucetCmd, "genesis")
	}
	addFaucetCmd = append(addFaucetCmd, "add-genesis-account", faucetWallet.FormattedAddress(), faucetAmount, "--keyring-backend", "test")
	faucetCommand := firstValidatorNode.BinCommand(addFaucetCmd...)
	scriptBuilder.WriteString(fmt.Sprintf("%s\n", strings.Join(faucetCommand, " ")))

	for i := 1; i < len(c.Validators); i++ {
		validator := c.Validators[i]
		validatorAddress := c.ValidatorWallets[i].FormattedAddress()

		c.logger.Info("adding validator to genesis script", zap.String("validator", validator.GetDefinition().Name),
			zap.String("address", validatorAddress))

		var addValidatorCmd []string
		if useGenesisSubCommand {
			addValidatorCmd = append(addValidatorCmd, "genesis")
		}
		addValidatorCmd = append(addValidatorCmd, "add-genesis-account", validatorAddress, faucetAmount, "--keyring-backend", "test")
		validatorCommand := firstValidatorNode.BinCommand(addValidatorCmd...)
		scriptBuilder.WriteString(fmt.Sprintf("%s\n", strings.Join(validatorCommand, " ")))

		// Copy gentx from other validators to first validator
		if err := validator.CopyGenTx(ctx, firstValidator); err != nil {
			return fmt.Errorf("failed to copy gentx from %s: %w", validator.GetDefinition().Name, err)
		}
	}

	var collectCmd []string
	if useGenesisSubCommand {
		collectCmd = append(collectCmd, "genesis")
	}
	collectCmd = append(collectCmd, "collect-gentxs")
	collectCommand := firstValidatorNode.BinCommand(collectCmd...)
	scriptBuilder.WriteString(fmt.Sprintf("%s\n", strings.Join(collectCommand, " ")))

	finalScript := scriptBuilder.String()
	c.logger.Info("executing genesis operations script")
	stdout, stderr, exitCode, err := firstValidator.RunCommand(ctx, []string{"/bin/sh", "-c", finalScript})
	if err != nil {
		return fmt.Errorf("failed to run final genesis script: %w", err)
	}

	if exitCode != 0 {
		return fmt.Errorf("final genesis script failed (exit code %d): %s, stdout: %s", exitCode, stderr, stdout)
	}

	if accountOpts.numAdditionalAccounts > 0 {
		if accountOpts.baseMnemonic == "" {
			return fmt.Errorf("base-mnemonic is required when additional accounts > 0")
		}
		accounts, err := buildAccounts(walletCfg, accountOpts.baseMnemonic, len(c.Validators)+1, accountOpts.numAdditionalAccounts)
		if err != nil {
			return fmt.Errorf("failed to build additional accounts: %w", err)
		}
		balances := buildBalances(accounts, genesisAmounts)
		eg := new(errgroup.Group)
		for _, v := range c.Validators {
			eg.Go(func() error {
				genesisBz, err := v.GenesisFileContent(ctx)
				if err != nil {
					return fmt.Errorf("failed to get genesis file: %w", err)
				}
				data := make(map[string]any)
				err = json.Unmarshal(genesisBz, &data)
				if err != nil {
					return err
				}

				data, err = UpdateGenesisAccounts(accounts, data)
				if err != nil {
					return fmt.Errorf("failed to update genesis accounts: %w", err)
				}

				data, err = UpdateGenesisBalances(balances, data)
				if err != nil {
					return fmt.Errorf("failed to update genesis balances: %w", err)
				}

				updatedGenesisBz, err := json.Marshal(data)
				if err != nil {
					return err
				}

				err = v.OverwriteGenesisFile(ctx, updatedGenesisBz)
				if err != nil {
					return fmt.Errorf("failed to overwrite genesis file for validator: %w", err)
				}
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return fmt.Errorf("failed to add additional accounts: %w", err)
		}
	}

	c.logger.Debug("final genesis script completed", zap.String("stdout", stdout), zap.String("stderr", stderr))

	return nil
}

// UpdateGenesisBalances updates the bank balance and supply state with the given balances.
func UpdateGenesisBalances(bals []Balance, data map[string]any) (map[string]any, error) {
	appstate := data["app_state"].(map[string]any)
	bankData := appstate["bank"].(map[string]any)

	// Marshal the bank data to JSON, then unmarshal into the proper type
	bankBytes, err := json.Marshal(bankData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bank data: %w", err)
	}

	var bankGenesis bankv1beta1.GenesisState
	err = protojson.Unmarshal(bankBytes, &bankGenesis)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bank genesis: %w", err)
	}

	// Track additional supply amounts by denomination
	additionalSupply := make(map[string]*big.Int)

	// Convert our Balance type to bank Balance type and add to genesis
	for _, bal := range bals {
		// Convert types.Coins to sdk.Coins
		coins := make([]*basev1beta1.Coin, len(bal.Coins))
		for i, coin := range bal.Coins {
			coins[i] = &basev1beta1.Coin{
				Denom:  coin.Denom,
				Amount: coin.Amount.String(),
			}

			// Track additional supply
			if additionalSupply[coin.Denom] == nil {
				additionalSupply[coin.Denom] = big.NewInt(0)
			}
			additionalSupply[coin.Denom].Add(additionalSupply[coin.Denom], coin.Amount.BigInt())
		}

		// Create bank balance
		bankBalance := &bankv1beta1.Balance{
			Address: bal.Address,
			Coins:   coins,
		}

		bankGenesis.Balances = append(bankGenesis.Balances, bankBalance)
	}

	// Update supply - convert existing supply to map for easier lookup
	supplyMap := make(map[string]*big.Int)
	for _, coin := range bankGenesis.Supply {
		amount, ok := new(big.Int).SetString(coin.Amount, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse supply amount: %s", coin.Amount)
		}
		supplyMap[coin.Denom] = amount
	}

	// Add additional amounts to existing supply or create new entries
	for denom, additionalAmount := range additionalSupply {
		if existingAmount, exists := supplyMap[denom]; exists {
			supplyMap[denom] = new(big.Int).Add(existingAmount, additionalAmount)
		} else {
			supplyMap[denom] = additionalAmount
		}
	}

	// Convert supply map back to coin slice
	bankGenesis.Supply = make([]*basev1beta1.Coin, 0, len(supplyMap))
	for denom, amount := range supplyMap {
		bankGenesis.Supply = append(bankGenesis.Supply, &basev1beta1.Coin{
			Denom:  denom,
			Amount: amount.String(),
		})
	}

	// Marshal the updated bank genesis back to JSON
	updatedBankBytes, err := protojson.Marshal(&bankGenesis)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated bank genesis: %w", err)
	}

	// Convert back to map[string]any for insertion into genesis
	var updatedBankData map[string]any
	err = json.Unmarshal(updatedBankBytes, &updatedBankData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated bank data: %w", err)
	}

	// Update the genesis data
	appstate["bank"] = updatedBankData
	data["app_state"] = appstate

	return data, nil
}

// UpdateGenesisAccounts updates the auth account genesis state with the given accounts.
func UpdateGenesisAccounts(accounts []Account, genesisData map[string]any) (map[string]any, error) {
	appstate := genesisData["app_state"].(map[string]any)
	auth := appstate["auth"].(map[string]any)
	genesisAccounts := auth["accounts"].([]any)

	for _, acc := range accounts {
		genesisAccounts = append(genesisAccounts, map[string]any{
			"@type":          acc.Type,
			"address":        acc.Address,
			"pub_key":        nil,
			"account_number": acc.AccountNumber,
			"sequence":       acc.Sequence,
		})
	}

	// Update the auth module with the modified accounts slice
	auth["accounts"] = genesisAccounts
	// Update the app_state with the modified auth module
	appstate["auth"] = auth
	// Update the main genesisData with the modified app_state
	genesisData["app_state"] = appstate

	return genesisData, nil
}

// builds wallets from the baseMnemonic, with a bipPassphrase of [0,numAdditionalAccounts)
func buildAccounts(walletCfg petritypes.WalletConfig, baseMnemonic string, startingAccNum, numAdditionalAccs int) ([]Account, error) {
	accounts := make([]Account, 0, numAdditionalAccs)
	for i := range numAdditionalAccs {
		keyName := fmt.Sprintf("additionalaccount%d", i)
		w, err := wallet.NewWallet(keyName, baseMnemonic, fmt.Sprintf("%d", i), walletCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet %d: %w", i+1, err)
		}
		account := Account{
			Type:          accountType,
			Address:       w.FormattedAddress(),
			PubKey:        nil,
			AccountNumber: strconv.Itoa(startingAccNum + i),
			Sequence:      "0",
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// builds a balance slice for the accounts and funds.
func buildBalances(accounts []Account, funds types.Coins) []Balance {
	balances := make([]Balance, 0, len(accounts))
	for _, acc := range accounts {
		bal := Balance{
			Address: acc.Address,
			Coins:   funds,
		}
		balances = append(balances, bal)
	}
	return balances
}

func configureNode(
	ctx context.Context,
	node petritypes.NodeI,
	chainConfig petritypes.ChainConfig,
	genbz []byte,
	persistentPeers PeerSet,
	seeds PeerSet,
	useExternalAddress bool,
	logger *zap.Logger,
) error {
	if err := node.OverwriteGenesisFile(ctx, genbz); err != nil {
		return err
	}

	var p2pExternalAddr string
	var err error
	if useExternalAddress {
		p2pExternalAddr, err = node.GetExternalAddress(ctx, "26656")
		if err != nil {
			return fmt.Errorf("failed to get external address for p2p port: %w", err)
		}
	} else {
		p2pExternalAddr, err = node.GetIP(ctx)
		if err != nil {
			return fmt.Errorf("failed to get ip for p2p port: %w", err)
		}
		p2pExternalAddr = fmt.Sprintf("%s:26656", p2pExternalAddr)
	}

	if err := node.SetChainConfigs(ctx, chainConfig.ChainId, p2pExternalAddr); err != nil {
		return err
	}

	persistentPeersString, err := persistentPeers.AsCometPeerString(ctx, useExternalAddress)
	if err != nil {
		return fmt.Errorf("failed to get comet peer string for persistent peers: %w", err)
	}

	logger.Debug("setting persistent peers", zap.String("persistent_peers", persistentPeersString))
	if err := node.SetPersistentPeers(ctx, persistentPeersString); err != nil {
		return err
	}

	seedPeersString, err := seeds.AsCometPeerString(ctx, useExternalAddress)
	if err != nil {
		return fmt.Errorf("failed to get comet peer string for seeds: %w", err)
	}

	logger.Debug("setting seeds", zap.String("seeds", seedPeersString))
	if err := node.SetSeedNode(ctx, seedPeersString); err != nil {
		return err
	}

	useLibP2P, addressBookFile := chainConfig.UseLibP2P()
	if useLibP2P {
		logger.Info("Using lib-p2p!", zap.String("address_book_file", addressBookFile))
		err = writeLibP2PAddressBook(
			ctx,
			node,
			addressBookFile,
			useExternalAddress,
			seeds,
			persistentPeers,
		)
		if err != nil {
			return fmt.Errorf("writeLibP2PAddressBook: %w", err)
		}
	}

	return nil
}

// writes libp2p address book to the node's address book file
func writeLibP2PAddressBook(
	ctx context.Context,
	node petritypes.NodeI,
	addressBookFile string,
	useExternalAddress bool,
	peerSets ...PeerSet,
) error {
	peers := []any{}

	isDocker := !useExternalAddress

	// combine all the peer sets into list of peers
	for _, peerSet := range peerSets {
		if peerSet.Empty() {
			continue
		}

		elements, err := peerSet.AsLibP2PAddressBook(ctx, isDocker)
		if err != nil {
			return fmt.Errorf("failed to get libp2p address book for peer set: %w", err)
		}

		peers = append(peers, elements...)
	}

	// @see https://github.com/cometbft/cometbft/blob/608fe92cbc3774c6cdf36c59c56b6c8362489ef1/lp2p/addressbook.go#L12
	addressBook := map[string]any{
		"peers": peers,
	}

	raw, err := toml.Marshal(addressBook)
	if err != nil {
		return fmt.Errorf("failed to marshal address book {%+v}: %w", addressBook, err)
	}

	return node.WriteFile(ctx, addressBookFile, raw)
}

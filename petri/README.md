# Petri

---

## Overview

Petri is the library that powers Ironbird's testnet deployment capabilities. It provides abstractions for provisioning infrastructure and managing node lifecycles across different environments.

---

## Architecture

### 1. Core Package (`core/`)

The core package contains provider abstractions and tooling for infrastructure management.

#### **Providers** (`core/provider/`)

Petri supports two main infrastructure providers:

##### **1. Docker Provider** (`provider/docker/`)
- **Purpose**: Local testnet deployment
- **Features**:
  - Spins up nodes as Docker containers
  - Network isolation via Docker networks
  - Volume management for chain data
  - Fast iteration cycles

##### **2. DigitalOcean Provider** (`provider/digitalocean/`)
- **Purpose**: Geodistributed testnet deployment
- **Features**:
  - Provisions droplets (VMs) for each node
  - Configures firewalls and networking
  - Supports multi-region deployments
  - Automatic resource tagging for cleanup
  - Integration with Tailscale for networking

#### **Provider Interface**

Both providers implement a common interface defined in `core/provider/provider.go`:

```go
type ProviderI interface {
    // Create and provision a task (node/container/VM)
    CreateTask(context.Context, TaskDefinition) (TaskI, error)
    
    // Serialize a specific task's state
    SerializeTask(context.Context, TaskI) ([]byte, error)
    
    // Restore a task from serialized state
    DeserializeTask(context.Context, []byte) (TaskI, error)
    
    // Clean up all provisioned resources
    Teardown(context.Context) error
    
    // Serialize entire provider state
    SerializeProvider(context.Context) ([]byte, error)
    
    // Get provider type identifier
    GetType() string
    
    // Get provider instance name
    GetName() string
}
```

**Tasks** (nodes, load balancers, etc.) implement `TaskI`, which provides operations like:
- `Start()`, `Stop()`, `Destroy()` - Task lifecycle management
- `WriteFile()`, `ReadFile()`, `DownloadDir()` - File operations
- `RunCommand()` - Execute commands inside the task
- `GetIP()`, `GetPrivateIP()`, `GetExternalAddress()` - Network addressing

This abstraction allows Ironbird to easily switch between local Docker deployments and cloud-based DigitalOcean deployments without changing workflow logic.

### 2. Cosmos Package (`cosmos/`)

The Cosmos package contains chain-specific logic for Cosmos SDK-based networks.

#### **Key Responsibilities**

1. **Genesis Operations** (`cosmos/chain/`)
   - Genesis file generation and modification
   - Validator key management
   - Account provisioning
   - Chain parameter configuration

2. **Chain Launch Logic** (`cosmos/chain/`, `cosmos/node/`)
   - Node initialization (`init` command)
   - Configuration file generation (`app.toml`, `config.toml`, `client.toml`)
   - Validator gentx collection
   - Network bootstrapping (peer discovery, seed nodes)
   - Node startup and health checks

3. **Wallet Management** (`cosmos/wallet/`)
   - Key signing and transaction broadcasting
   - Account funding for load tests

---

## Usage Flow

### Typical Testnet Launch Sequence

1. **Provider Initialization**
   ```go
   // Docker: Connects to local Docker daemon
   provider, err := docker.CreateProvider(ctx, logger, name)
   
   // DigitalOcean: Initializes DO client with API token
   provider, err := digitalocean.NewProvider(ctx, name, token, tailscaleSettings)
   ```

2. **Chain Object Creation**
   ```go
   // Creates Chain object and provisions infrastructure (containers/droplets)
   // This calls provider.CreateTask() for each validator and full node
   chain, err := petrichain.CreateChain(ctx, logger, provider, chainConfig, chainOptions)
   ```

3. **Chain Initialization** (`chain.Init()`) which does the following:
   - **Validator Setup**: For each validator node
     - Runs `<chain-binary> init` to create home directory structure
     - Generates validator keys (priv_validator_key.json, node_key.json)
     - Creates genesis transaction (gentx)
     - Returns validator wallet
   
   - **Node Setup**: For each full node
     - Runs `<chain-binary> init`
     - Generates node keys
   
   - **Genesis Assembly**: On first validator
     - Collects all validator gentxs
     - Adds genesis accounts (validators, faucet, load test wallets)
     - Applies custom genesis modifications if specified
     - Builds final genesis.json
   
   - **Configuration Distribution**: For all nodes
     - Writes genesis.json to all nodes
     - Generates and writes config files:
       - `app.toml` (API, gRPC, telemetry settings)
       - `config.toml` (P2P, consensus, RPC, seed/persistent peers)
       - `client.toml` (chain-id, keyring settings)
   
   - **Network Startup**: Starts all node tasks
     - Calls `task.Start()` to launch chain binary
     - Nodes connect via P2P and begin consensus

4. **Health Check**
   ```go
   // Wait for chain to produce blocks
   err = chain.WaitForStartup(ctx)
   ```

5. **Teardown**
   ```go
   // Cleans up all tasks (stops and destroys containers/droplets)
   err = provider.Teardown(ctx)
   ```

---

## Integration with Ironbird

Petri is consumed by Ironbird's Temporal activities:

- **CreateProvider Activity** → Initializes provider (Docker/DigitalOcean client)
- **LaunchTestnet Activity** → Calls `CreateChain()` and `chain.Init()`
- **TeardownProvider Activity** → Calls `provider.Teardown()`

The provider state is serialized and stored in Temporal workflow context, allowing for distributed execution and recovery.

---

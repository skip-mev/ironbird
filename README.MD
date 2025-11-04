# Ironbird

> A testnet orchestration system for Cosmos SDK and EVM chains deployed geodistributed on Digital Ocean or locally via Docker.

---

## Architecture Overview

Ironbird consists of four main components:

### 1. **Server**
- Exposes gRPC API for workflow management
- Handles workflow submission and state management
- Stores workflow history in SQLite database
- **Dockerfile**: `server.Dockerfile`
- **Config**: `conf/server.yaml`

### 2. **Worker**
- Executes Temporal workflow activities
- Manages the core `LaunchTestnet` workflow with the following activities:
  1. **BuildDockerImage** - Builds chain binaries into Docker images using BuildKit
     - Accepts custom dependency versions for:
       - **Chain repository** (e.g., Cosmos EVM) at a specific Git SHA
       - **Cosmos SDK version** (optional override via `CosmosSdkSha`)
       - **CometBFT version** (optional override via `CometBFTSha`)
     - Automatically injects `go mod edit -replace` commands during the build process to swap dependencies
     - Allows flexible version combinations (e.g., latest EVM with older SDK, or specific CometBFT with any SDK)
     - **Dependency conflict handling**: Build will fail if Go module resolution detects incompatibilities
     - Checks for existing images in ECR (or local Docker) to avoid redundant builds
     - Tags images with dependency versions to ensure cache correctness
  2. **CreateProvider** - Initializes infrastructure provider (Docker or DigitalOcean)
  3. **LaunchTestnet** - Provisions and configures the blockchain network
     - **Infrastructure Provisioning**: Creates compute resources (containers or droplets) for validators and full nodes
     - **Node Distribution**: Supports multi-region deployment for DigitalOcean or local Docker networks
     - **Genesis Generation**: 
       - Creates initial validator keys and wallets
       - Configures genesis accounts with initial token balances
       - Applies custom genesis modifications (staking params, gov params, etc.)
       - Collects and processes validator gentxs
     - **Configuration Management**: Writes and customizes `app.toml`, `config.toml`, and `client.toml`
     - **Peer Setting**: Configures peer discovery (persistent peers or seed nodes)
     - **Wallet Provisioning**: Derives specified number of funded wallets from base mnemonic for load testing
  4. **LaunchLoadBalancer** (optional, DigitalOcean only) - Sets up SSL-enabled load-balanced public endpoints
     - Creates a DigitalOcean load balancer droplet with HAProxy
     - Generates domain endpoints for each protocol:
       - `{chain-name}-rpc.{domain}` - Tendermint RPC (HTTPS)
       - `{chain-name}-grpc.{domain}` - Cosmos gRPC (gRPC over TLS)
       - `{chain-name}-lcd.{domain}` - Cosmos REST API (HTTPS)
       - `{chain-name}-evmrpc.{domain}` - EVM JSON-RPC (HTTPS, for EVM chains only)
       - `{chain-name}-evmws.{domain}` - EVM WebSocket (WSS, for EVM chains only)
     - Load balances requests across all validators and full nodes in the testnet
  5. **RunLoadTest** (optional) - Executes load tests via Catalyst
  6. **TeardownProvider** - Cleans up all provisioned resources
- **Dockerfile**: `worker.Dockerfile`
- **Config**: `conf/worker.yaml`
- **Chain Dockerfiles**: Located in `hack/` directory (these need to be updated separately in ironbird-manifests for dev and prod):
  - `hack/simapp.Dockerfile` - Cosmos SDK simapp
  - `hack/gaia.Dockerfile` - Cosmos Hub
  - `hack/evm.Dockerfile` - EVM-compatible chains

### 3. **Frontend**
- React-based UI for workflow creation and monitoring
- Built with Vite and TypeScript
- Communicates with server via gRPC-Web
- **Location**: `frontend/`

### 4. **Cleanup**
- Runs as a scheduled cron job (currently scheduled in manifests to be daily at midnight)
- Identifies and deletes dangling DigitalOcean resources:
  - Droplets older than 30 minutes
  - Associated firewalls
- **Respects LONG_RUNNING tag** - resources tagged with `LONG_RUNNING` are never deleted
- **Dockerfile**: `cleanup.Dockerfile`

---

## Deployment Configuration

All Kubernetes manifests and Helm charts are located in the **`ironbird-manifests/`** repository

---

## Setup Guide

Ironbird supports two deployment modes:
- **Local Docker Mode**: Run testnets locally using Docker (no cloud dependencies required)
- **Full Mode (DigitalOcean)**: Deploy testnets to DigitalOcean droplets (requires AWS, Tailscale, and DigitalOcean)

### Quick Start (Local Docker Mode)

For running local Docker workflows with **zero cloud dependencies**:

#### 1. First-Time Setup

```bash
make first-time-setup
```

This installs required dependencies (Docker, Temporal, etc.) and generates SSL certificates.

#### 2. Start IronBird

```bash
make local-docker
```

The frontend will be available at `http://localhost:5173`

---

### Full Setup (DigitalOcean Mode)

For deploying testnets to DigitalOcean droplets:

#### 1. First-Time Setup

```bash
make first-time-setup
```

#### 2. Set Environment Variables

```bash
cp env.example .env
```

Set the required environment variables (ICL secrets can be found in Bitwarden under note "Ironbird"):

   ```bash
   set -o allexport
   source .env
   set +o allexport
   ```

#### 3. Authenticate with AWS

```bash
aws-vault exec skip-dev-admin
```

> **Note**: If you haven't set up AWS profiles, follow the guide at https://platform.skip.build/guides/aws/login_to_aws

#### 4. Start IronBird

```bash
make local-full
```


## Testing

You can either create a new testnet through the UI, or edit `hack/workflow.json` to define the desired testnet configs and submit using:
```bash
make test-workflow
```

## Access Points

- **Ironbird UI**: http://localhost:5173
- **Temporal UI**: http://localhost:8233
- **Ironbird GRPC**: localhost:9006
- **Temporal GRPC**: localhost:7233


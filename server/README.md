# Ironbird Server

The Ironbird Server provides a gRPC API with gRPC-Web support for managing testnet environments and load tests. It integrates with Temporal for workflow management.

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Build the server:
```bash
go build -o ironbird-server cmd/main.go
```

3. Run the server:
```bash
./ironbird-server -grpc-addr :9006
```

The server will start on port 9006 by default.

## API Endpoints

### 1. Create a New Testnet Workflow

**Endpoint:** `CreateWorkflow`

Creates a new testnet environment with the specified configuration.

Example request:
```json
{
  "repo": "Cosmos SDK",
  "sha": "commit-hash",
  "chain_config": {
    "name": "testnet-name",
    "image": "docker-image:tag",
    "genesis_modifications": [
      {
        "key": "consensus.params.block.max_gas",
        "value": "75000000"
      }
    ],
    "num_of_nodes": 4,
    "num_of_validators": 3
  },
  "load_test_spec": {
    "name": "basic-load-test",
    "description": "Basic load test configuration",
    "chain_id": "test-chain",
    "num_of_blocks": 100,
    "msgs": [],
    "unordered_txs": true,
    "tx_timeout": "30s"
  },
  "long_running_testnet": false,
  "testnet_duration": "2h"
}
```

### 2. Get Testnet Workflow Status

**Endpoint:** `GetWorkflow`

Retrieves the current status and details of a specific testnet workflow.

### 3. List Testnet Workflows

**Endpoint:** `ListWorkflows`

Lists all testnet workflows with their statuses.

### 4. Cancel Testnet Workflow

**Endpoint:** `CancelWorkflow`

Cancels a running testnet workflow.

### 5. Signal Testnet Workflow

**Endpoint:** `SignalWorkflow`

Sends a signal to a running testnet workflow.

### 6. Run Load Test on Existing Testnet

**Endpoint:** `RunLoadTest`

Initiates a load test on an existing testnet environment.

Example request:
```json
{
  "workflow_id": "workflow-id",
  "load_test_spec": {
    "name": "sustained-load-test",
    "description": "High transaction volume test",
    "chain_id": "test-chain",
    "num_of_txs": 1000,
    "num_of_blocks": 200,
    "msgs": [],
    "unordered_txs": true,
    "tx_timeout": "30s"
  }
}
```

## Development

The server is implemented as a gRPC server with gRPC-Web support and uses the following components:

- `grpc_server.go`: Main server implementation with gRPC service handlers
- `proto/ironbird.proto`: Protocol buffer definitions for the gRPC service
- `cmd/main.go`: Server entry point

To add new endpoints or modify existing ones, update the proto definitions in `proto/ironbird.proto` and implement the corresponding handler functions in `grpc_server.go`.

# Ironbird Server

The Ironbird Server provides a REST API for managing testnet environments and load tests. It uses Caddy as the web server and integrates with Temporal for workflow management.

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
./ironbird-server -caddyfile Caddyfile
```

The server will start on port 8080 by default.

## API Endpoints

### 1. Create a New Testnet Workflow

**Endpoint:** `POST /ironbird/workflow`

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

### 2. Update an Existing Testnet Workflow

**Endpoint:** `PUT /ironbird/workflow/{workflow_id}`

Updates the configuration of an existing testnet environment.

### 3. Get Testnet Workflow Status

**Endpoint:** `GET /ironbird/workflow/{workflow_id}`

Retrieves the current status and details of a specific testnet workflow.

### 4. Run Load Test on Existing Testnet

**Endpoint:** `POST /ironbird/loadtest/{workflow_id}`

Initiates a load test on an existing testnet environment.

Example request:
```json
{
  "name": "sustained-load-test",
  "description": "High transaction volume test",
  "chain_id": "test-chain",
  "num_of_txs": 1000,
  "num_of_blocks": 200,
  "msgs": [],
  "unordered_txs": true,
  "tx_timeout": "30s"
}
```

## Development

The server is implemented as a Caddy module and uses the following components:

- `server.go`: Main server implementation with request handlers
- `types.go`: Type definitions for requests and responses
- `Caddyfile`: Caddy server configuration
- `cmd/main.go`: Server entry point

To add new endpoints or modify existing ones, update the relevant handler functions in `server.go`. 
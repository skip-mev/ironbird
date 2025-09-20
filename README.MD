# Ironbird Setup Guide

## Initial Setup

1. If running for the first time, run:
   ```bash
   make first-time-setup
   ```

## Start Temporal Server

2. Start the Temporal server:
   ```bash
   temporal server start-dev
   ```

## Run BuildKitD

3. Pull and run BuildKitD container:
   ```bash
   docker pull moby/buildkit:latest

   docker run -d --name buildkitd --privileged -p 1234:1234 \
     -v /var/run/docker.sock:/var/run/docker.sock \
     -v buildkitd:/var/lib/buildkit \
     -v ~/.docker/config.json:/root/.docker/config.json \
     moby/buildkit:latest --addr tcp://0.0.0.0:1234
   ```

## Start Worker

4. Authenticate with AWS:
   ```bash
   aws sso login --profile skip-dev-admin
   aws-vault exec skip-dev-admin
   ```
   > **Note**: If you haven't set up AWS profiles, follow the guide at https://platform.skip.build/guides/aws/login_to_aws

5. Copy environment configuration:
   ```bash
   cp env.example .env
   ```

6. Set the required environment variables (ICL secrets can be found in Bitwarden under note "Ironbird"):
   ```bash
   set -o allexport
   source .env
   set +o allexport
   ```

7. Start the worker:
   ```bash
   export TSNET_FORCE_LOGIN=1 && go run ./cmd/worker
   ```

## Start Backend and Frontend 

8. Start the backend:
   ```bash
   make start-backend
   ```

9. Start the frontend:
   ```bash
   make start-frontend
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

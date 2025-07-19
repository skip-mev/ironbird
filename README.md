# Ironbird Local Development Setup

This guide explains how to set up Ironbird for local development using Docker Compose.

## Quick Start

1. **First-time setup:** (only if setting up ironbird for first time)
   ```bash
   make first-time-setup
   ```

2. **Authenticate with AWS:**
   ```bash
   aws sso login --profile <aws_profile>
   aws-vault exec <aws_profile>
   ```

3. **Configure environment:**
   ```bash
   cp env.example .env
   # Edit .env with your DigitalOcean and Tailscale credentials
   ```

4. **Start services:**
   ```bash
   make docker-up
   ```

5. **Shut down services:**
   ```bash
   make docker-down
   ```

6. **Configure test workflow in hack/workflow.json**

7. **Submit a test workflow**
   ```bash
   make test-workflow
   ```

## Access points

- **Ironbird UI**: http://localhost:3001
- **Temporal UI**: http://localhost:8080
- **Ironbird GRPC**: localhost:9006
- **Temporal GRPC**: localhost:7233

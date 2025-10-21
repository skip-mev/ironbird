# Ironbird Setup Guide

## Overview

Ironbird supports two deployment modes:
- **Local Docker Mode**: Run testnets locally using Docker (no cloud dependencies required)
- **Full Mode (DigitalOcean)**: Deploy testnets to DigitalOcean droplets (requires AWS, Tailscale, and DigitalOcean)

## Quick Start (Local Docker Mode)

For running local Docker workflows with **zero cloud dependencies**:

### 1. First-Time Setup

```bash
make first-time-setup
```

This installs required dependencies (Docker, Temporal, mprocs, etc.) and generates SSL certificates.

### 2. Start IronBird

```bash
make local-docker
```

The frontend will be available at `http://localhost:5173`

---

## Full Setup (DigitalOcean Mode)

For deploying testnets to DigitalOcean droplets:

### 1. First-Time Setup

```bash
make first-time-setup
```

### 2. Set Environment Variables

```bash
cp env.example .env
```

Set the required environment variables (ICL secrets can be found in Bitwarden under note "Ironbird"):

   ```bash
   set -o allexport
   source .env
   set +o allexport
   ```

### 3. Authenticate with AWS

```bash
aws-vault exec skip-dev-admin
```

> **Note**: If you haven't set up AWS profiles, follow the guide at https://platform.skip.build/guides/aws/login_to_aws

### 4. Start IronBird

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


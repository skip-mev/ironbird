WORKER_BIN=./build/worker
SERVER_BIN=./build/server
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
GO_DEPS=go.mod go.sum

PROTO_PATH=./server/proto
PROTO_FILE=$(PROTO_PATH)/ironbird.proto
SERVICE=skip.ironbird.IronbirdService
METHOD=CreateWorkflow
ADDRESS=localhost:9006
JSON_FILE=./hack/create-workflow.json

help: ## List of commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

call-grpc: ## Call the gRPC endpoint
	cat ./hack/create-workflow.json | grpcurl \
	-import-path $(PROTO_PATH) \
	-proto $(PROTO_FILE) \
	-plaintext \
	-d @ \
	$(ADDRESS) \
	$(SERVICE)/$(METHOD)

temporal-reset: ## Reset the Temporal server
	temporal workflow list -o json | jq -r '.[] | select (.status == "WORKFLOW_EXECUTION_STATUS_RUNNING") | .execution.workflowId' | xargs -I{} temporal workflow terminate --reason lol --workflow-id "{}"

do-reset: ## Reset the DigitalOcean droplets
	doctl compute droplet list | grep petri-droplet | cut -d' ' -f1 | xargs -I{} doctl compute droplet delete -f {} && doctl compute firewall list | grep petri | cut -d' ' -f1 | xargs -I{} doctl compute firewall delete -f {} && doctl compute ssh-key list | grep petri | cut -d' ' -f1 | xargs -I{} doctl compute ssh-key delete -f {}

test-workflow: ## Test the workflow
	temporal workflow start --task-queue TESTNET_TASK_QUEUE --name Workflow --input-file hack/workflow.json

cancel-workflows: ## Cancel the workflows
	temporal workflow list -o json | jq -r '.[] | select (.status == "WORKFLOW_EXECUTION_STATUS_RUNNING") | .execution.workflowId' | xargs -I{} temporal workflow cancel -w {}

.PHONY: help call-grpc temporal-reset do-reset test-workflow cancel-workflows

###############################################################################
###                                 Builds                                  ###
###############################################################################

tidy: ## Tidy the dependencies
	go mod tidy

deps: ## Download the dependencies
	go env
	go mod download

${WORKER_BIN}: ${GO_FILES} ${GO_DEPS} ## Build the worker binary
	@echo "Building worker binary..."
	@mkdir -p ./build
	go build -o ./build/ github.com/skip-mev/ironbird/cmd/worker

${SERVER_BIN}: ${GO_FILES} ${GO_DEPS} ## Build the server binary
	@echo "Building server binary..."
	@mkdir -p ./build
	cd server && go build -o ../build/server ./cmd

build: ${WORKER_BIN} ${SERVER_BIN} ## Build the worker and server binaries

.PHONY: tidy deps build

###############################################################################
###                                  Proto                                  ###
###############################################################################

proto-gen: ## Generate the gRPC code from the proto files
	@echo "Generating gRPC code from proto files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		server/proto/ironbird.proto
	@echo "Generating frontend TypeScript code from proto files..."
	cd frontend && npm run generate-proto

proto-tools: ## Install the protoc-gen-go and protoc-gen-go-grpc tools
	@echo "Installing protoc-gen-go and protoc-gen-go-grpc..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installing frontend proto tools..."
	cd frontend && npm install --save-dev @bufbuild/protoc-gen-connect-es @bufbuild/protoc-gen-es

.PHONY: proto-gen proto-tools

###############################################################################
###                                  Testing                                ###
###############################################################################

unit-test: ## Run the unit tests
	go test -p 1 -v -count 1 -timeout 30m `go list ./... | grep -v e2e` -race

petri-unit-test: ## Run the petri unit tests
	@docker pull nginx:latest
	@docker pull ghcr.io/cosmos/simapp:v0.47
	@go test -v -count 2 ./petri/core/... -race
	@go test -v -count 2 `go list ./petri/cosmos/... | grep -v e2e` -race

petri-docker-e2e: ## Run the petri Docker E2E tests
	@docker pull nginx:latest
	@docker pull ghcr.io/cosmos/simapp:v0.47
	@go test -v -count 1 ./petri/cosmos/tests/e2e/docker/... -race -v

petri-digitalocean-e2e: ## Run the petri DigitalOcean E2E tests
	@go test -v -count 1 ./petri/cosmos/tests/e2e/digitalocean/... -race -v

petri-e2e-test: petri-docker-e2e petri-digitalocean-e2e ## Run the petri E2E tests

.PHONY: unit-test petri-unit-test petri-docker-e2e petri-digitalocean-e2e petri-e2e-test

###############################################################################
###                       Formatting / Linting                              ###
###############################################################################

format: ## Format the code
	@find . -name '*.go' -type f -not -path "*.git*" -not -path "*/mocks/*" -not -name '*.pb.go' -not -name '*.pulsar.go' -not -name '*.gw.go' | xargs go run mvdan.cc/gofumpt -w .
	@find . -name '*.go' -type f -not -path "*.git*" -not -path "*/mocks/*" -not -name '*.pb.go' -not -name '*.pulsar.go' -not -name '*.gw.go' | xargs go run github.com/client9/misspell/cmd/misspell -w
	@find . -name '*.go' -type f -not -path "*.git*" -not -path "/*mocks/*" -not -name '*.pb.go' -not -name '*.pulsar.go' -not -name '*.gw.go' | xargs go run golang.org/x/tools/cmd/goimports -w -local github.com/skip-mev/ironbird

lint: tidy ## Run the linter
	@echo "--> Running linter"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run --out-format=tab

lint-fix: tidy ## Fix the linter
	@echo "--> Running linter"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run --fix --out-format=tab --issues-exit-code=0

lint-markdown: tidy ## Run the markdown linter
	@echo "--> Running markdown linter"
	@markdownlint **/*.md

govulncheck: tidy ## Run the govulncheck
	@echo "--> Running govulncheck"
	@go run golang.org/x/vuln/cmd/govulncheck -test ./...

.PHONY: format lint lint-fix lint-markdown govulncheck

###############################################################################
###                           Starting Services                             ###
###############################################################################

start-buildkit: ## Start the buildkit container
	docker run -d --name buildkitd --privileged -p 1234:1234 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v buildkitd:/var/lib/buildkit \
		-v ~/.docker/config.json:/root/.docker/config.json \
		moby/buildkit:latest --addr tcp://0.0.0.0:1234

start-temporal: ## Start the Temporal server
	temporal server start-dev

start-worker: ## Start the worker
	go run ./cmd/worker/main.go

start-frontend: ## Start the frontend
	cd frontend && npm install --legacy-peer-deps && npm run dev

start-backend: ## Start the backend
	go run ./server/cmd/main.go

local: ## Start local IronBird instance
	@if [ ! -f .env ]; then \
		echo ".env file not found. Exiting"; \
		exit 1; \
	fi

	@if [ -z "$${AWS_SESSION_TOKEN-}" ]; then \
		echo "AWS auth missing. Use:"; \
		echo "   aws sso login --profile skip-dev-admin"; \
		echo "   aws-vault exec skip-dev-admin"; \
		exit 1; \
	fi

	@echo "Using variables from .env ✔︎";
	@echo "Using AWS auth ✔︎";

	@set -a && source .env && mprocs -c mprocs.yaml

.PHONY: start-buildkit start-temporal start-worker start-frontend start-backend local

###############################################################################
###                           First time setup                             ###
###############################################################################

install-deps: ## Install the dependencies
	@echo "📦 Installing dependencies via Homebrew..."
	@brew install docker docker-compose awscli aws-vault openssl make temporal mprocs || echo "⚠️  Some packages may already be installed"
	@echo "✅ All dependencies installed!"
	@echo ""

generate-certs: ## Generate the SSL certificates
	@echo "🔐 Generating SSL certificates..."
	@mkdir -p conf
	@if [ -f "conf/ib-local-key.pem" ] && [ -f "conf/ib-local-cert.pem" ]; then \
		echo "⚠️  Certificates already exist, skipping generation"; \
	else \
		openssl genrsa -out conf/ib-local-key.pem 2048 && \
		openssl req -x509 -new -nodes -key conf/ib-local-key.pem -sha256 -days 1825 -out conf/ib-local-cert.pem \
			-subj "/C=/ST=/L=/O=/OU=/CN=localhost" && \
		echo "✅ SSL certificates generated successfully"; \
	fi

first-time-setup: install-deps generate-certs ## First-time setup
	@echo "🎉 First-time setup complete!"

.PHONY: install-deps generate-certs first-time-setup
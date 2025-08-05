WORKER_BIN=./build/worker
SERVER_BIN=./build/server
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
GO_DEPS=go.mod go.sum

temporal-reset:
	temporal workflow list -o json | jq -r '.[] | select (.status == "WORKFLOW_EXECUTION_STATUS_RUNNING") | .execution.workflowId' | xargs -I{} temporal workflow terminate --reason lol --workflow-id "{}"

do-reset:
	doctl compute droplet list | grep petri-droplet | cut -d' ' -f1 | xargs -I{} doctl compute droplet delete -f {} && doctl compute firewall list | grep petri | cut -d' ' -f1 | xargs -I{} doctl compute firewall delete -f {} && doctl compute ssh-key list | grep petri | cut -d' ' -f1 | xargs -I{} doctl compute ssh-key delete -f {}

test-workflow:
	temporal workflow start --task-queue TESTNET_TASK_QUEUE --name Workflow --input-file hack/workflow.json

cancel-workflows:
	temporal workflow list -o json | jq -r '.[] | select (.status == "WORKFLOW_EXECUTION_STATUS_RUNNING") | .execution.workflowId' | xargs -I{} temporal workflow cancel -w {}

reset: do-reset temporal-reset

.PHONY: reset temporal-reset do-reset test-workflow cancel-workflows

###############################################################################
###                                 Builds                                  ###
###############################################################################

.PHONY: tidy deps
tidy:
	go mod tidy

deps:
	go env
	go mod download


${WORKER_BIN}: ${GO_FILES} ${GO_DEPS}
	@echo "Building worker binary..."
	@mkdir -p ./build
	go build -o ./build/ github.com/skip-mev/ironbird/cmd/worker

${SERVER_BIN}: ${GO_FILES} ${GO_DEPS}
	@echo "Building server binary..."
	@mkdir -p ./build
	cd server && go build -o ../build/server ./cmd

.PHONY: build
build: ${WORKER_BIN} ${SERVER_BIN}

###############################################################################
###                                  Proto                                  ###
###############################################################################

.PHONY: proto-gen
proto-gen:
	@echo "Generating gRPC code from proto files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		server/proto/ironbird.proto
	@echo "Generating frontend TypeScript code from proto files..."
	cd frontend && npm run generate-proto

.PHONY: proto-tools
proto-tools:
	@echo "Installing protoc-gen-go and protoc-gen-go-grpc..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installing frontend proto tools..."
	cd frontend && npm install --save-dev @bufbuild/protoc-gen-connect-es @bufbuild/protoc-gen-es

###############################################################################
###                                  Testing                                ###
###############################################################################

.PHONY: unit-test
unit-test:
	go test -p 1 -v -count 1 -timeout 30m ./... -race

###############################################################################
###                                Formatting                               ###
###############################################################################

format:
	@find . -name '*.go' -type f -not -path "*.git*" -not -path "*/mocks/*" -not -name '*.pb.go' -not -name '*.pulsar.go' -not -name '*.gw.go' | xargs go run mvdan.cc/gofumpt -w .
	@find . -name '*.go' -type f -not -path "*.git*" -not -path "*/mocks/*" -not -name '*.pb.go' -not -name '*.pulsar.go' -not -name '*.gw.go' | xargs go run github.com/client9/misspell/cmd/misspell -w
	@find . -name '*.go' -type f -not -path "*.git*" -not -path "/*mocks/*" -not -name '*.pb.go' -not -name '*.pulsar.go' -not -name '*.gw.go' | xargs go run golang.org/x/tools/cmd/goimports -w -local github.com/skip-mev/ironbird

.PHONY: format


###############################################################################
###                                Linting                                  ###
###############################################################################

lint: tidy
	@echo "--> Running linter"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run --out-format=tab

lint-fix: tidy
	@echo "--> Running linter"
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run --fix --out-format=tab --issues-exit-code=0

lint-markdown: tidy
	@echo "--> Running markdown linter"
	@markdownlint **/*.md

govulncheck: tidy
	@echo "--> Running govulncheck"
	@go run golang.org/x/vuln/cmd/govulncheck -test ./...

.PHONY: lint lint-fix lint-markdown govulncheck ⏎

.PHONY: start-frontend
start-frontend:
	cd frontend && npm install --legacy-peer-deps && npm run dev

.PHONY: start-backend
start-backend:
	go run ./server/cmd/main.go

.PHONY: dev
dev:
	make -j2 start-frontend start-backend

###############################################################################
###                                Docker                                   ###
###############################################################################

.PHONY: docker-build docker-up docker-down docker-logs docker-dev

# Build Docker images
docker-build:
	@echo "--> Building Docker images..."
	docker-compose build

# Start services in production mode
docker-up:
	@echo "--> Starting services with Docker Compose..."
	docker-compose up -d

# Stop and remove services
docker-down:
	@echo "--> Stopping services..."
	docker-compose down

# View logs from all services
docker-logs:
	@echo "--> Showing logs..."
	docker-compose logs -f

# Start services in development mode (with logs)
docker-dev:
	@echo "--> Starting services in development mode..."
	docker-compose up --build

# Clean up Docker resources
docker-clean:
	@echo "--> Cleaning up Docker resources..."
	docker-compose down --volumes --remove-orphans
	docker system prune -f

###############################################################################
###                           Local Development                             ###
###############################################################################

.PHONY: install-deps generate-certs first-time-setup

install-deps:
	@echo "📦 Installing dependencies via Homebrew..."
	@brew install docker docker-compose awscli aws-vault openssl make temporal || echo "⚠️  Some packages may already be installed"
	@echo "✅ All dependencies installed!"
	@echo ""

generate-certs:
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

first-time-setup: install-deps generate-certs
	@echo "🎉 First-time setup complete!"

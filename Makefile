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

call-grpc:
	cat ./hack/create-workflow.json | grpcurl \
	-import-path $(PROTO_PATH) \
	-proto $(PROTO_FILE) \
	-plaintext \
	-d @ \
	$(ADDRESS) \
	$(SERVICE)/$(METHOD)

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

.PHONY: unit-test petri-unit-test petri-docker-e2e petri-digitalocean-e2e petri-e2e-test
unit-test:
	go test -p 1 -v -count 1 -timeout 30m `go list ./... | grep -v e2e` -race

petri-unit-test:
	@docker pull nginx:latest
	@docker pull ghcr.io/cosmos/simapp:v0.47
	@go test -v -count 2 ./petri/core/... -race
	@go test -v -count 2 `go list ./petri/cosmos/... | grep -v e2e` -race

petri-docker-e2e:
	@docker pull nginx:latest
	@docker pull ghcr.io/cosmos/simapp:v0.47
	@go test -v -count 1 ./petri/cosmos/tests/e2e/docker/... -race -v

petri-digitalocean-e2e:
	@go test -v -count 1 ./petri/cosmos/tests/e2e/digitalocean/... -race -v

petri-e2e-test: petri-docker-e2e petri-digitalocean-e2e

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

###############################################################################
###                           Starting Ssrvices                             ###
###############################################################################

.PHONY: start-buildkit
start-buildkit:
	docker run -d --name buildkitd --privileged -p 1234:1234 -v /var/run/docker.sock:/var/run/docker.sock -v buildkitd:/var/lib/buildkit -v ~/.docker/config.json:/root/.docker/config.json moby/buildkit:latest --addr tcp://0.0.0.0:1234

.PHONY: start-temporal
start-temporal:
	temporal server start-dev

.PHONY: start-worker
start-worker:
	go run ./cmd/worker/main.go

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
###                           First time setup                             ###
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

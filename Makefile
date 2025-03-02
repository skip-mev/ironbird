WORKER_BIN=./build/worker
APP_BIN=./build/app
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")
GO_DEPS=go.mod go.sum

temporal-reset:
	temporal workflow list -o json | jq -r '.[] | select (.status == "WORKFLOW_EXECUTION_STATUS_RUNNING") | .execution.workflowId' | xargs -I{} temporal workflow terminate --reason lol --workflow-id "{}"

do-reset:
	doctl compute droplet list | grep petri-droplet | cut -d' ' -f1 | xargs -I{} doctl compute droplet delete -f {} && doctl compute firewall list | grep petri | cut -d' ' -f1 | xargs -I{} doctl compute firewall delete -f {} && doctl compute ssh-key list | grep petri | cut -d' ' -f1 | xargs -I{} doctl compute ssh-key delete -f {}

test-workflow:
	temporal workflow start --task-queue TESTNET_TASK_QUEUE --name Workflow --input-file hack/workflow.json

reset: do-reset temporal-reset

.PHONY: reset temporal-reset do-reset test-workflow

###############################################################################
###                                 Builds                                  ###
###############################################################################

.PHONY: tidy deps
tidy:
	go mod tidy

deps:
	go env
	go mod download

${APP_BIN}: ${GO_FILES} ${GO_DEPS}
	@echo "Building application binary..."
	@mkdir -p ./build
	go build -o ./build/ github.com/skip-mev/ironbird/cmd/app

${WORKER_BIN}: ${GO_FILES} ${GO_DEPS}
	@echo "Building worker binary..."
	@mkdir -p ./build
	go build -o ./build/ github.com/skip-mev/ironbird/cmd/worker

.PHONY: build
build: ${APP_BIN} ${WORKER_BIN}

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

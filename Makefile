.DEFAULT_GOAL:=help
SHELL:=/bin/sh
GOPATH := $(shell go env GOPATH)

help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

########################################################################################################################
##@ Common
########################################################################################################################

.PHONY: build
build: ## Builds the application for production
	go build -ldflags="-w -s" -v -o ./bin/echo-mcp .

.PHONY: clean
clean: ## Runs mod tidy
	go mod tidy

.PHONY: update
update: ## Update go modules
	go get -t -u ./...

########################################################################################################################
##@ Run
########################################################################################################################

.PHONY: run
run: ## Execute the application locally
	go run -race .

.PHONY: swag
swag: ## Run the swaggo example application (root module, Echo v4)
	go run -race ./examples/simple/main.go

.PHONY: openapi
openapi: ## Run the openapi example application (root module, Echo v4)
	go run -race ./examples/complex/main.go

.PHONY: run-v5
run-v5: ## Execute the Echo v5 module application locally
	go run -race ./v5

.PHONY: swag-v5
swag-v5: ## Run the swaggo example application (Echo v5 module)
	go run -race ./v5/examples/simple/main.go

.PHONY: openapi-v5
openapi-v5: ## Run the openapi example application (Echo v5 module)
	go run -race ./v5/examples/complex/main.go

########################################################################################################################
##@ Testing
########################################################################################################################


.PHONY: mcp
mcp: ## Run the mcp client inspector
	npx @modelcontextprotocol/inspector http://localhost:8080/mcp

.PHONY: test
test: ## Runs tests
	go install github.com/mfridman/tparse@main | go mod tidy
	PROFILE=local go test -parallel 10 -json -cover ./... | tparse -all -pass -trimpath=github.com/BrunoKrugel/echo-mcp/
	cd v5 && PROFILE=local go test -parallel 10 -json -cover ./... | tparse -all -pass -trimpath=github.com/BrunoKrugel/echo-mcp/v5/

########################################################################################################################
##@ Quality
########################################################################################################################

.PHONY: format
format: ## Format code and organize imports
	goimports -w .
	go fmt ./...
	fieldalignment -fix ./...
	cd v5 && goimports -w . && go fmt ./... && fieldalignment -fix ./...

.PHONY: modernize
modernize: ## Modernize code
	go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix -test ./...
	cd v5 && go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix -test ./...

.PHONY: lint
lint: ## Runs golangci-lint (root module and Echo v5 module)
	golangci-lint run --fix
	cd v5 && golangci-lint run --fix



.DEFAULT_GOAL:=help
SHELL:=/bin/sh
GOPATH := $(shell go env GOPATH)

help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\033[36m\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

########################################################################################################################
##@ OpenAPI
########################################################################################################################

.PHONY: openapi
openapi: ## Generate OpenAPI docs
openapi:
	go run -race . > ./docs/api/openapi.yaml

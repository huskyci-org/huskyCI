.SILENT:
.DEFAULT_GOAL := help

GO ?= go
GOROOT ?= $(shell $(GO) env GOROOT)
GOPATH ?= $(shell $(GO) env GOPATH)
GOBIN ?= $(GOPATH)/bin
GOLINT ?= $(GOBIN)/golint
GOSEC ?= $(GOBIN)/gosec
GOVERALLS ?= $(GOBIN)/goveralls

HUSKYCI-API-BIN ?= huskyci-api-bin
HUSKYCI-CLIENT-BIN ?= huskyci-client-bin
HUSKYCI-CLI-BIN ?= huskyci-cli-bin
HUSKYCI-RUNNER-BIN ?= huskyci-runner-bin

COLOR_RESET = \033[0m
COLOR_COMMAND = \033[36m
COLOR_YELLOW = \033[33m
COLOR_GREEN = \033[32m
COLOR_RED = \033[31m

PROJECT := huskyCI

TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || git describe --always --abbrev=0)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT := $(shell git rev-parse HEAD)
LDFLAGS := '-X "main.version=$(TAG)" -X "main.commit=$(COMMIT)" -X "main.date=$(DATE)"'

## Builds all project binaries
build-all: build-api build-api-linux build-client build-client-linux build-cli build-cli-linux

## Builds API code into a binary
build-api:
	cd api && $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-API-BIN)" server.go
	@if [ "$$(uname -s)" = "Darwin" ] && [ -f "api/$(HUSKYCI-API-BIN)" ]; then xattr -c "api/$(HUSKYCI-API-BIN)" 2>/dev/null || true; fi

## Builds API code using linux architecture into a binary
build-api-linux:
	cd api && GOOS=linux GOARCH=amd64 $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-API-BIN)" server.go

## Builds client code into a binary
build-client:
	cd client/cmd && $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-CLIENT-BIN)" main.go

## Builds client code using linux architecture into a binary
build-client-linux:
	cd client/cmd && GOOS=linux GOARCH=amd64 $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-CLIENT-BIN)" main.go

## Builds cli code into a binary
build-cli:
	cd cli && $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-CLI-BIN)" main.go
	@if [ "$$(uname -s)" = "Darwin" ] && [ -f "cli/$(HUSKYCI-CLI-BIN)" ]; then xattr -c "cli/$(HUSKYCI-CLI-BIN)" 2>/dev/null || true; fi

## Builds cli code using macOS (darwin) architecture into a binary
build-cli-darwin:
	cd cli && GOOS=darwin GOARCH=$(shell go env GOARCH) $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-CLI-BIN)" main.go

## Builds cli code using macOS (darwin) architecture for Intel Macs (amd64)
build-cli-darwin-amd64:
	cd cli && GOOS=darwin GOARCH=amd64 $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-CLI-BIN)" main.go

## Builds cli code using macOS (darwin) architecture for Apple Silicon Macs (arm64)
build-cli-darwin-arm64:
	cd cli && GOOS=darwin GOARCH=arm64 $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-CLI-BIN)" main.go

## Builds cli code using linux architecture into a binary
build-cli-linux:
	cd cli && GOOS=linux GOARCH=amd64 $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-CLI-BIN)" main.go

## Builds the remote runner service binary (for HUSKYCI_RUNNER_TYPE=remote)
build-runner:
	cd cmd/runner && $(GO) build -ldflags $(LDFLAGS) -o "$(HUSKYCI-RUNNER-BIN)" .
	@if [ "$$(uname -s)" = "Darwin" ] && [ -f "cmd/runner/$(HUSKYCI-RUNNER-BIN)" ]; then xattr -c "cmd/runner/$(HUSKYCI-RUNNER-BIN)" 2>/dev/null || true; fi

## Builds all securityTest containers locally with the latest tags
build-containers:
	chmod +x deployments/scripts/build-containers.sh
	./deployments/scripts/build-containers.sh

## Checks dependencies
check-deps:
	cd api && $(GO) mod tidy && $(GO) mod verify
	cd cli && $(GO) mod tidy && $(GO) mod verify
	cd client && $(GO) mod tidy && $(GO) mod verify

## Runs a security static analysis using Gosec
check-sec:
	$(GO) get -u github.com/securego/gosec/cmd/gosec
	cd api && $(GOSEC) ./...
	cd client && $(GOSEC) ./...
	cd cli && $(GOSEC) ./...

## Checks .env file from huskyCI
check-env:
	cat .env

## Checks every securityTest version from their container images
check-containers-version:
	chmod +x deployments/scripts/check-containers-version.sh
	./deployments/scripts/check-containers-version.sh

## Composes huskyCI environment using docker-compose
compose:
	docker-compose -f deployments/docker-compose.yml down -v
	docker-compose -f deployments/docker-compose.yml up -d --build --force-recreate

## Builds the extract image and loads it into the running Docker API (DinD). Required for file:// (zip) analysis when using Docker Compose. Run once after 'make compose'.
load-extract-image:
	docker build -t huskyciorg/extract:latest deployments/dockerfiles/extract
	docker save huskyciorg/extract:latest | docker exec -i huskyCI_Docker_API docker load

## Composes down
compose-down:
	docker-compose -f deployments/docker-compose.yml down -v

## Creates certs and sets all config to huskyCI_Docker_API
create-certs:
	chmod +x deployments/scripts/run-create-certs.sh
	./deployments/scripts/run-create-certs.sh

## Generates a local token to be used in a local environment
# generate-local-token:
#     chmod +x deployments/scripts/generate-local-token.sh
#     ./deployments/scripts/generate-local-token.sh

## Generates passwords and set them as environment variables
generate-passwords:
	chmod +x deployments/scripts/generate-env.sh
	./deployments/scripts/generate-env.sh

## Sends coverage report to coveralls
goveralls:
	$(GO) get -u github.com/mattn/goveralls
	$(GOVERALLS) -coverprofile=c.out -service=circle-ci -repotoken=$COVERALLS_TOKEN

## Prints help message
help:
	printf "\n${COLOR_YELLOW}${PROJECT}\n------\n${COLOR_RESET}"
	awk '/^[a-zA-Z\-\_0-9\.%]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "${COLOR_COMMAND}$$ make %s${COLOR_RESET} %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST) | sort
	printf "\n"

## Installs a development environment using docker-compose
# generate-local-token has removed
install: create-certs compose generate-passwords

## Runs all huskyCI lint
lint:
	$(GO) install -u golang.org/x/lint/golint
	$(GOLINT) ./...

## Push securityTest containers to hub.docker
push-containers:
	chmod +x deployments/scripts/push-containers.sh
	./deployments/scripts/push-containers.sh

## Restarts only huskyCI_API container
restart-huskyci-api:
	chmod +x deployments/scripts/restart-huskyci-api.sh
	./deployments/scripts/restart-huskyci-api.sh

## Runs huskyci-client
run-cli: build-cli
	./cli/"$(HUSKYCI-CLI-BIN)" run

## Run huskyci-client compiling it in Linux arch
run-cli-linux: build-cli-linux
	./cli/"$(HUSKYCI-CLI-BIN)" run

## Runs huskyci-client
run-client: build-client
	./client/cmd/"$(HUSKYCI-CLIENT-BIN)"

## Runs huskyci-client with JSON output
run-client-json: build-client
	./client/cmd/"$(HUSKYCI-CLIENT-BIN)" JSON

## Run huskyci-client compiling it in Linux arch
run-client-linux: build-client-linux
	./client/cmd/"$(HUSKYCI-CLIENT-BIN)"

## Run huskyci-client compiling it in Linux arch with JSON output
run-client-linux-json: build-client-linux
	./client/cmd/"$(HUSKYCI-CLIENT-BIN)" JSON

## Performs all unit tests using ginkgo
test:
	cd api && $(GO) test -coverprofile=c.out ./...
	cd api && $(GO) tool cover -func=c.out
	cd api && $(GO) tool cover -html=c.out -o coverage.html
	cd client && $(GO) test -coverprofile=d.out ./...
	cd client && $(GO) tool cover -func=d.out
	cd cli && $(GO) test -coverprofile=e.out ./...
	cd cli && $(GO) tool cover -func=e.out

## Builds and push securityTest containers with the latest tags
update-containers: build-containers push-containers

## Runs end-to-end tests
test-e2e:
	chmod +x tests/e2e/run-e2e-tests.sh
	./tests/e2e/run-e2e-tests.sh

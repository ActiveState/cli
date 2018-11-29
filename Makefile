# Go parameters
GOROOT := $(shell go env GOROOT)
GOCMD   = go

ifneq ($(OS),Windows_NT)
	ifdef GOROOT
		GOCMD=${GOROOT}/bin/go
	endif
else
	GOCMD=${GOROOT}bin\\\go
endif

GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

PACKRCMD=packr

ifneq ($(OS),Windows_NT)
	ifndef $(shell command -v packr 2> /dev/null)
		PACKRCMD=${GOPATH}/bin/packr
	endif
else
	PACKRCMD=${GOPATH}bin\\\packr
endif

STATE=state
BINARY_NAME=$(STATE)
ifeq ($(OS),Windows_NT)
    BINARY_NAME=$(STATE).exe
endif

.PHONY: build build-dev test install deploy-updates deploy-artifacts generate-artifacts packr preprocess

all: test build
init:
		git config core.hooksPath .githooks
		go get -u github.com/gobuffalo/packr/...

packr:
	$(PACKRCMD)
preprocess:
	CLIENV=$(CLIENV) $(GOCMD) run scripts/constants-generator/main.go

build: packr preprocess
	cd $(STATE) && $(GOBUILD) -ldflags="-s -w" -o ../build/$(BINARY_NAME) $(STATE).go
	mkdir -p public/update
	$(GOCMD) run scripts/update-generator/main.go -o public/update build/$(BINARY_NAME)

build-dev: CLIENV=dev
build-dev: build

install: packr
	cd $(STATE) && $(GOINSTALL) $(STATE).go

generate-artifacts:
	$(GOCMD) run scripts/artifact-generator/main.go 
deploy-updates:
	$(GOCMD) run scripts/s3-deployer/main.go public/update ca-central-1 cli-update update/state
	$(GOCMD) run scripts/s3-deployer/main.go public/install.sh ca-central-1 cli-update update/state/install.sh
deploy-artifacts:
	$(GOCMD) run scripts/s3-deployer/main.go public/distro ca-central-1 cli-artifacts distro

generate-api-client:
	cd internal && swagger generate client -f https://staging.activestate.com/swagger.json -A api
generate-secrets-client:
	# CURRENTLY A PLACE HOLDER REMINDER
	cd internal/secrets-api && swagger generate client -f ../../../secrets-svc/api/swagger.yml -A secrets-api
generate-clients: generate-api-client generate-secrets-client

test: preprocess
	$(GOTEST) -parallel 12 `$(GOCMD) list ./... | grep -vE "(secrets-)?api/(client|model)"`
clean: 
	$(GOCLEAN)
	rm -Rf build
run:
	cd $(STATE) && $(GOBUILD) -o ../build/$(BINARY_NAME) $(STATE).go
	build/$(BINARY_NAME) --help

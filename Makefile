# Go parameters
GOCMD=go

ifndef $(shell command -v go 2> /dev/null)
    GOCMD=${GOROOT}/bin/go
endif

GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

PACKRCMD=packr

ifndef $(shell command -v packr 2> /dev/null)
    PACKRCMD=${GOPATH}/bin/packr
endif

BINARY_NAME=state
BINARY_UNIX=$(BINARY_NAME)_unix

VERSION=`grep -m1 "^const Version" internal/constants/generated.go | cut -d ' ' -f4 | tr -d '"'`

.PHONY: build test install deploy-updates deploy-artifacts generate-artifacts

all: test build
init:
		git config core.hooksPath .githooks
		go get -u github.com/gobuffalo/packr/...
build: 
		$(PACKRCMD)
		$(GOCMD) run scripts/constants-generator/main.go 
		cd $(BINARY_NAME) && $(GOBUILD) -ldflags="-s -w" -o ../build/$(BINARY_NAME) $(BINARY_NAME).go
		mkdir -p public/update
		$(GOCMD) run scripts/update-generator/main.go -o public/update build/state $(VERSION) 
install: 
		$(PACKRCMD)
		cd $(BINARY_NAME) && $(GOINSTALL) $(BINARY_NAME).go
generate-artifacts:
		$(GOCMD) run scripts/artifact-generator/main.go 
deploy-updates:
		$(GOCMD) run scripts/s3-deployer/main.go public/update ca-central-1 cli-update update/state
		$(GOCMD) run scripts/s3-deployer/main.go public/install.sh ca-central-1 cli-update update/state/install.sh
deploy-artifacts:
		$(GOCMD) run scripts/s3-deployer/main.go public/distro ca-central-1 cli-artifacts distro
generate-api-client:
		cd internal
		swagger generate client -f https://staging.activestate.com/swagger.json -A api
test: 
		$(GOCMD) run scripts/constants-generator/main.go 
		$(GOTEST) ./...
clean: 
		$(GOCLEAN)
		rm -Rf build
run:
		cd $(BINARY_NAME) && $(GOBUILD) -o ../build/$(BINARY_NAME) $(BINARY_NAME).go
		build/$(BINARY_NAME) --help

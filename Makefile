# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

BINARY_NAME=state
BINARY_UNIX=$(BINARY_NAME)_unix

VERSION=`grep -m1 "^const Version" internal/constants/constants.go | cut -d ' ' -f4 | tr -d '"'`

.PHONY: build test install deploy-updates deploy-artefacts generate-artefacts

all: test build
init:
		git config core.hooksPath .githooks
build: 
		cd $(BINARY_NAME) && $(GOBUILD) -o ../build/$(BINARY_NAME) $(BINARY_NAME).go
		mkdir -p public/update
		go run scripts/update-generator/main.go -o public/update build/state $(VERSION) 
install: 
		cd $(BINARY_NAME) && $(GOINSTALL) $(BINARY_NAME).go
generate-artefacts:
		go run scripts/artefact-generator/main.go 
deploy-updates:
		go run scripts/s3-deployer/main.go public/update ca-central-1 cli-update update/state
		go run scripts/s3-deployer/main.go public/install.sh ca-central-1 cli-update update/state/install.sh
deploy-artefacts:
		go run scripts/s3-deployer/main.go public/distro ca-central-1 cli-artefacts distro
test: 
		$(GOTEST) ./...
clean: 
		$(GOCLEAN)
		rm -Rf build
run:
		cd $(BINARY_NAME) && $(GOBUILD) -o ../build/$(BINARY_NAME) $(BINARY_NAME).go
		build/$(BINARY_NAME) --help

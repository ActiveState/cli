# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

BINARY_NAME=state
BINARY_UNIX=$(BINARY_NAME)_unix

VERSION=`grep -m1 "^const Version" internal/constants/generated.go | cut -d ' ' -f4 | tr -d '"'`

.PHONY: build test install deploy-updates deploy-artifacts generate-artifacts

all: test build
init:
		git config core.hooksPath .githooks
build: 
		go run scripts/constants-generator/main.go 
		cd $(BINARY_NAME) && $(GOBUILD) -ldflags="-s -w" -o ../build/$(BINARY_NAME) $(BINARY_NAME).go
		upx build/$(BINARY_NAME)
		mkdir -p public/update
		go run scripts/update-generator/main.go -o public/update build/state $(VERSION) 
install: 
		cd $(BINARY_NAME) && $(GOINSTALL) $(BINARY_NAME).go
generate-artifacts:
		go run scripts/artifact-generator/main.go 
deploy-updates:
		go run scripts/s3-deployer/main.go public/update ca-central-1 cli-update update/state
		go run scripts/s3-deployer/main.go public/install.sh ca-central-1 cli-update update/state/install.sh
deploy-artifacts:
		go run scripts/s3-deployer/main.go public/distro ca-central-1 cli-artifacts distro
test: 
		go run scripts/constants-generator/main.go 
		$(GOTEST) ./...
clean: 
		$(GOCLEAN)
		rm -Rf build
run:
		cd $(BINARY_NAME) && $(GOBUILD) -o ../build/$(BINARY_NAME) $(BINARY_NAME).go
		build/$(BINARY_NAME) --help

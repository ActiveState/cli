# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

BINARY_NAME=state
BINARY_UNIX=$(BINARY_NAME)_unix

all: test build
build: 
		cd $(BINARY_NAME) && $(GOBUILD) -o ../build/$(BINARY_NAME) $(BINARY_NAME).go
test: 
		$(GOTEST) -v ./...
clean: 
		$(GOCLEAN)
		rm -Rf build
run:
		cd $(BINARY_NAME) && $(GOBUILD) -o ../build/$(BINARY_NAME) $(BINARY_NAME).go
		build/$(BINARY_NAME) --help
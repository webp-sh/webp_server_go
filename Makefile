GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=webp-server
BINARY_LINUX=$(BINARY_NAME)_linux-amd64

all: build
build: 
		$(GOBUILD) -o $(BINARY_LINUX) -v
test: 
		$(GOTEST) -v ./...
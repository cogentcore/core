# Basic Go makefile

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get


all: build

build: 
	$(GOBUILD) -v
dbg-build:
	$(GOBUILD) -v -gcflags=all="-N -l" -tags debug
test: 
	$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)

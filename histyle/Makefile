# Basic Go makefile

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get


all: build

build: 
	$(GOBUILD) -v

test: 
	$(GOTEST) -v ./...

clean: 
	$(GOCLEAN)

bindata:
	go-bindata -pkg histyle defaults.histys
	

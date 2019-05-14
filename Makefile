# Basic Go makefile

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# exclude python from std builds
DIRS=`go list ./... | grep -v python`

all: build

build: 
	$(GOBUILD) -v $(DIRS)
test: 
	$(GOTEST) -v $(DIRS)
clean: 
	$(GOCLEAN) ./...
	
release:
	$(MAKE) -C gi release


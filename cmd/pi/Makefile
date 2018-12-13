# Basic Go makefile

GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get


all: build

build: 
	$(GOBUILD) -v
install:
	$(GOINSTALL) -v
dbg-build:
	$(GOBUILD) -v -gcflags=all="-N -l" -tags debug
test: 
	$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)
app-install:
	/bin/cp -av gide.app /Applications
mac-sh-install:
	/bin/cp gide-mac.sh /usr/local/bin/gide



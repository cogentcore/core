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
	# This is the best test: compare full go library parse against "good" standard
	# Takes roughly 40sec real time on Randy's macbook pro 15 inch late 2016
	time ./pi -r -ex cmd,runtime /usr/local/Cellar/go/1.11.3/libexec/src >& go_src_1_11_3_cur.out
	diff go_src_1_11_3_std.out go_src_1_11_3_cur.out
clean: 
	$(GOCLEAN)
app-install:
	/bin/cp -av gide.app /Applications
mac-sh-install:
	/bin/cp gide-mac.sh /usr/local/bin/gide



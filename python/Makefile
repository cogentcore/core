# Makefile for gopy pkg generation of python bindings to gi

PYTHON=python3
PIP=$(PYTHON) -m pip

PBGV=`$(PIP) list | grep PyBindGen`

all: prereq gen

.PHONY: prereq gen all build install install-pkg install-exe clean

prereq:
	@echo "Installing go prerequisites:"
	- go get golang.org/x/tools/cmd/goimports  # this installs into ~/go/bin
	- go get github.com/go-python/gopy@v0.4.5
	@echo "Installing python prerequisites -- ignore err if already installed:"
	- $(PIP) install -r requirements.txt
	@echo
	@echo "if this fails, you may see errors like this:"
	@echo "    Undefined symbols for architecture x86_64:"
	@echo "    _PyInit__gi, referenced from:..."
	@echo

install: install-pkg install-exe

# note: it is important that gi3d come after giv, otherwise gi3dcaptures all the common types
# unfortunately this means that all sub-packages need to be explicitly listed.
gen:
	gopy exe -name=gi -vm=python3 -no-warn -exclude=driver,oswin -main="runtime.LockOSThread(); gimain.Main(func() {  GoPyMainRun() })" goki.dev/ki/v2/ki goki.dev/ki/v2/kit goki.dev/mat32/v2 goki.dev/gi/v2/units goki.dev/gi/v2/gist goki.dev/gi/v2/girl  goki.dev/gi/v2/gi goki.dev/gi/v2/svg goki.dev/gi/v2/giv goki.dev/gi/v2/gi3d goki.dev/gi/v2/gimain

build:
	$(MAKE) -C gi build

install-pkg:
	# this does a local install of the package, building the sdist and then directly installing it
	cp pygiv/pygiv.py gi/
	rm -rf dist build */*.egg-info *.egg-info
	$(PYTHON) setup.py sdist
	$(PIP) install dist/*.tar.gz

install-exe:
	# install executable into /usr/local/bin
	cp gi/pygi /usr/local/bin/

install-win:
	# windows version: install executable into gopath too, add .exe
	- mkdir -p /usr/local/bin
	- cp gi/pygi $(GOPATH)/bin/pygi.exe
	- cp gi/pygi C:/usr/local/bin/pygi.exe
	
clean:
	rm -rf gi dist build */*.egg-info *.egg-info

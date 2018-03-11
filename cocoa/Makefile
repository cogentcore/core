STUBS = stub_amd64.syso stub_386.syso
SOURCES = $(wildcard *.m)

OBJDIR = _obj
OBJS32 = $(foreach source, $(SOURCES), $(OBJDIR)/$(source).386.o)
OBJS64 = $(foreach source, $(SOURCES), $(OBJDIR)/$(source).amd64.o)

CC32 = $(shell GOARCH=386 go env CC) $(shell GOARCH=386 go env GOGCCFLAGS)
CC64 = $(shell GOARCH=amd64 go env CC) $(shell GOARCH=amd64 go env GOGCCFLAGS)

all : ${STUBS}

clean :
	rm -rf $(OBJDIR)

distclean : clean
	rm -f $(STUBS)

$(OBJDIR)/%.m.amd64.o : %.m
	@mkdir -p $(OBJDIR)
	$(CC64) -o $@ -c $<

$(OBJDIR)/%.m.386.o : %.m
	@mkdir -p $(OBJDIR)
	$(CC32) -o $@ -c $<
	
stub_amd64.syso : $(OBJS64)
	$(LD) -r -o $@ $^

stub_386.syso : $(OBJS32)
	$(LD) -r -o $@ $^

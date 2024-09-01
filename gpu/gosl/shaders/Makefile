# Makefile for dxc compiling of WGSL files for compute

all: basic.spv

%.spv : %.wgsl
	dxc -spirv -O3 -T cs_6_0 -E main -Fo $@ $<


# Makefile for glslc compiling of HLSL files for compute

all: rand.spv

%.spv : %.hlsl
	dxc -spirv -O3 -T cs_6_0 -E main -Fo $@ $<
	

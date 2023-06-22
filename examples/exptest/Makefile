# Makefile for glslc compiling of HLSL files for compute

all: gpu_exptest.spv

%.spv : %.hlsl
	dxc -spirv -O3 -T cs_6_0 -E main -Fo $@ $<


# Makefile for glslc compiling of HLSL files for compute

all: gpu_exptest.spv

%.spv : %.hlsl
	dxc -spirv -O3 -Ges -T cs_6_2 -E main -Fo $@ $<


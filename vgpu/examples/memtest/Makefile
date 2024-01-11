# Makefile for dxc compiling of HLSL files for compute

all: gpu_memtest.spv

%.spv : %.hlsl
	dxc -spirv -O3 -Ges -T cs_6_2 -E main -Fo $@ $<


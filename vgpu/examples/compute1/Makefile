# Makefile for glslc compiling of HLSL files for compute

all: sqvecel.spv

%.spv : %.hlsl
	glslc -fshader-stage=compute -o $@ $<


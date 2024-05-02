# Makefile for glslc compiling of HLSL files for compute

all: slrand.spv

%.spv : %.hlsl
	glslc -fshader-stage=compute -o $@ $<



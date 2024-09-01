# Makefile for glslc compiling of WGSL files for compute

all: slrand.spv

%.spv : %.wgsl
	glslc -fshader-stage=compute -o $@ $<



# Makefile for glslc compiling of gsl HLSL files for compute

all: texture_vert.spv texture_frag.spv

%.spv : %.hlsl
	glslc -fshader-stage=compute -o $@ $<

%.spv : %.vert
	glslc -fshader-stage=vertex -o $@ $<

%.spv : %.frag
	glslc -fshader-stage=fragment -o $@ $<
	

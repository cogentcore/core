# Makefile for glslc compiling of GSL, HLSL files for compute

all: comp_vert.spv comp_frag.spv fill_vert.spv fill_frag.spv

%.spv : %.hlsl
	glslc -fshader-stage=compute -o $@ $<

%.spv : %.vert
	glslc -fshader-stage=vertex -o $@ $<

%.spv : %.frag
	glslc -fshader-stage=fragment -o $@ $<
	

# Makefile for glslc compiling of HLSL files for compute

all: indexed.spv vtxcolor.spv

%.spv : %.hlsl
	glslc -fshader-stage=compute -o $@ $<

%.spv : %.vert
	glslc -fshader-stage=vertex -o $@ $<

%.spv : %.frag
	glslc -fshader-stage=fragment -o $@ $<
	

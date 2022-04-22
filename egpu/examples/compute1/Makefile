# Makefile for glslc compiling of HLSL files for compute

%.spv : %.hlsl
	glslc -fshader-stage=compute -o $@ $<


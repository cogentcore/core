# Makefile for glslc compiling of GSL, HLSL files for compute

all: texture_vert.spv texture_frag.spv onecolor_vert.spv onecolor_frag.spv pervertex_vert.spv pervertex_frag.spv

texture_frag.spv : phong_frag.frag

onecolor_frag.spv : phong_frag.frag

pervertex_frag.spv : phong_frag.frag

%.spv : %.hlsl
	glslc -fshader-stage=compute -o $@ $<

%.spv : %.vert
	glslc -fshader-stage=vertex -o $@ $<

%.spv : %.frag
	glslc -fshader-stage=fragment -o $@ $<
	

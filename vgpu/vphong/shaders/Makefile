# Makefile for glslc compiling of GSL, HLSL files for compute

all: texture_vert.spv texture_frag.spv onecolor_vert.spv onecolor_frag.spv pervertex_vert.spv pervertex_frag.spv

texture_frag.spv : phong_frag.frag

onecolor_frag.spv : phong_frag.frag

pervertex_frag.spv : phong_frag.frag

%.spv : %.hlsl
	glslc -fshader-stage=compute --target-env=vulkan1.2 -O -o $@ $<

%.spv : %.vert
	glslc -fshader-stage=vertex --target-env=vulkan1.2 -O -o $@ $<

%.spv : %.frag
	glslc -fshader-stage=fragment --target-env=vulkan1.2 -O -o $@ $<

rebuild:
	rm *.spv


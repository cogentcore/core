#version 450

layout(set = 2, binding = 0) uniform MatsU {
	vec3 Specular;
	vec3 ShinyBright; // x = Shiny, y = Bright
};

// https://www.khronos.org/registry/vulkan/specs/1.3-extensions/man/html/VK_EXT_descriptor_indexing.html
// this is a key set of extensions that would allow textures to be fully dynamic
// for now, we use a fixed set of textures
// 1.2: https://github.com/goki/vulkan/issues/43
layout(set = 3, binding = 0) uniform sampler2D TexSampler[];

layout(push_constant) uniform TexIdxU {
	int TexIdx;
};

// uniform vec2 TexRepeat;
// uniform vec2 TexOff;

layout(location = 0) in vec4 Pos;
layout(location = 1) in vec3 Norm;
layout(location = 2) in vec3 CamDir;
layout(location = 3) in vec2 TexCoord;

layout(location = 0) out vec4 outputColor;

#include "phong_frag.frag"
			
void main() {
	vec4 Color = texture(TexSampler[TexIdx], TexCoord); //  * TexRepeat + TexOff);
	float opacity = Color.a;
	vec3 clr = Color.rgb;	
	
	// Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
	float Shiny = ShinyBright.x;
	float Bright = ShinyBright.y;
	vec3 Ambdiff, Spec;
	PhongModel(Pos, Norm, CamDir, clr, clr, Specular, Shiny, Ambdiff, Spec);

	// Final fragment color -- premultiplied alpha
	outputColor = min(vec4((Bright * Ambdiff + Spec) * opacity, opacity), vec4(1.0));
}


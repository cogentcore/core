#version 450
#extension GL_EXT_nonuniform_qualifier : require

layout(set = 1, binding = 0) uniform ColorU {
	vec3 Color;
	vec3 Emissive;
	vec3 Specular;
	vec3 ShinyBright; // x = Shiny, y = Bright
};

layout(set = 4, binding = 0) uniform sampler2D TexSampler[];

layout(push_constant) uniform TexIdxU {
	vec2 TexRepeat;
	vec2 TexOff;
	int TexIdx;
};

layout(location = 0) in vec4 Pos;
layout(location = 1) in vec3 Norm;
layout(location = 2) in vec3 CamDir;
layout(location = 3) in vec2 TexCoord;

layout(location = 0) out vec4 outputColor;

#include "phong_frag.frag"
			
void main() {
	vec4 TColor = texture(TexSampler[TexIdx], TexCoord * TexRepeat + TexOff);
	float opacity = TColor.a;
	vec3 clr = TColor.rgb;	
	
	// Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
	float Shiny = ShinyBright.x;
	float Bright = ShinyBright.y;
	vec3 Ambdiff, Spec;
	PhongModel(Pos, Norm, CamDir, clr, clr, Specular, Shiny, Ambdiff, Spec);

	// Final fragment color -- premultiplied alpha
	outputColor = min(vec4((Bright * Ambdiff + Spec) * opacity, opacity), vec4(1.0));
	// outputColor = vec4(clr, 1);
}


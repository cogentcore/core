#version 450
#extension GL_EXT_nonuniform_qualifier : require

layout(set = 1, binding = 0) uniform ColorU {
	vec4 Color;
	vec3 Emissive;
	vec3 Specular;
	vec3 ShinyBright; // x = Shiny, y = Bright
};

layout(location = 0) in vec4 Pos;
layout(location = 1) in vec3 Norm;
layout(location = 2) in vec3 CamDir;
layout(location = 3) in vec2 TexCoord;
layout(location = 4) in vec4 VtxColor;

layout(location = 0) out vec4 outputColor;

#include "phong_frag.frag"
			
void main() {
	float opacity = VtxColor.a;
	vec3 clr = VtxColor.rgb;	
	
	// Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
	float Shiny = ShinyBright.x;
	float Bright = ShinyBright.y;
	vec3 Ambdiff, Spec;
	PhongModel(Pos, Norm, CamDir, clr, clr, Specular, Shiny, Ambdiff, Spec);

	// Final fragment color -- premultiplied alpha
	outputColor = min(vec4((Bright * Ambdiff + Spec) * opacity, opacity), vec4(1.0));
}


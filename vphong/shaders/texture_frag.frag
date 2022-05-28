#version 450
#extension GL_EXT_nonuniform_qualifier : require

// must be <= 128 bytes -- contains all per-object data
layout(push_constant) uniform PushU {
	mat4 ModelMtx; // 64 bytes, [3][3] = TexPct.X
	vec4 Color; // 16
	vec4 ShinyBright; // 16 x = Shiny, y = Reflect, z = Bright, w = TexIdx
	vec4 Emissive; // 16 rgb, a = TexPct.Y
	vec4 TexRepeatOff; // 16 xy = Repeat, zw = Offset
};

layout(set = 0, binding = 0) uniform MtxsU {
    mat4 ViewMtx;
    mat4 PrjnMtx;
};

layout(set = 3, binding = 0) uniform sampler2DArray TexSampler[];

layout(location = 0) in vec4 Pos;
layout(location = 1) in vec3 Norm;
layout(location = 2) in vec3 CamDir;
layout(location = 3) in vec2 TexCoord;

layout(location = 0) out vec4 outputColor;

#include "phong_frag.frag"

void TexWin(vec2 pct, out vec2 tc) {
	tc = mod(pct * (TexCoord * TexRepeatOff.xy + TexRepeatOff.zw), pct);
}

void main() {
	int TexIdxP = int(ShinyBright.w);
	int TexIdx = TexIdxP / 1024;
	int TexLay = TexIdxP % 1024;
	vec2 TexPct = vec2(ModelMtx[3][3], Emissive.a);
	vec2 tc;
	TexWin(TexPct, tc);
	vec4 TColor = texture(TexSampler[TexIdx], vec3(tc, TexLay));
	float opacity = TColor.a;
	vec3 clr = TColor.rgb;	
	
	// Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
	float Shiny = ShinyBright.x;
	float Reflect = ShinyBright.y;
	float Bright = ShinyBright.z;
	vec3 Specular = vec3(1,1,1);
	vec3 Ambdiff, Spec;
	PhongModel(Pos, Norm, CamDir, clr, clr, Specular, Shiny, Reflect, Ambdiff, Spec);

	// Final fragment color -- premultiplied alpha
	outputColor = min(vec4((Bright * Ambdiff + Spec) * opacity, opacity), vec4(1.0));
}


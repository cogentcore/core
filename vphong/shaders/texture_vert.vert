#version 450

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

layout(location = 0) in vec3 VtxPos;
layout(location = 1) in vec3 VtxNorm;
layout(location = 2) in vec2 VtxTex;
// layout(location = 3) in vec4 VtxColor;

layout(location = 0) out vec4 Pos;
layout(location = 1) out vec3 Norm;
layout(location = 2) out vec3 CamDir;
layout(location = 3) out vec2 TexCoord;

void main() {
	vec4 vPos = vec4(VtxPos, 1.0);
	vec4 vNorm = vec4(VtxNorm, 1.0);
	mat4 MMtx = ModelMtx;
	MMtx[3][3] = 1;
	mat4 MVMtx = ViewMtx * MMtx;
	Pos = MVMtx * vPos;
	mat3 NormMtx = transpose(inverse(mat3(MVMtx)));
	Norm = normalize(NormMtx * VtxNorm).xyz;
	CamDir = normalize(-Pos.xyz);
	TexCoord = VtxTex;
	gl_Position = PrjnMtx * MVMtx * vPos;
}


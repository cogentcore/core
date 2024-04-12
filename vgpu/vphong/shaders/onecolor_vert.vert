#version 450

// must be <= 128 bytes -- contains all per-object data
layout(push_constant) uniform PushU {
	mat4 ModelMtx; // 64 bytes, [3][3] = TexPct.X
	vector4 Color; // 16
	vector4 ShinyBright; // 16 x = Shiny, y = Reflect, z = Bright, w = TexIndex
	vector4 Emissive; // 16 rgb, a = TexPct.Y
	vector4 TexRepeatOff; // 16 xy = Repeat, zw = Offset
};

layout(set = 0, binding = 0) uniform MtxsU {
    mat4 ViewMtx;
    mat4 PrjnMtx;
};

layout(location = 0) in vector3 VtxPos;
layout(location = 1) in vector3 VtxNorm;
// layout(location = 2) in vector2 VtxTex;
// layout(location = 3) in vector4 VtxColor;

layout(location = 0) out vector4 Pos;
layout(location = 1) out vector3 Norm;
layout(location = 2) out vector3 CamDir;
// layout(location = 3) out vector2 TexCoord;

void main() {
	vector4 vPos = vector4(VtxPos, 1.0);
	mat4 MVMtx = ViewMtx * ModelMtx;
	Pos = MVMtx * vPos;
	mat3 NormMtx = transpose(inverse(mat3(MVMtx)));
	Norm = normalize(NormMtx * VtxNorm).xyz;
	CamDir = normalize(-Pos.xyz);
	// TexCoord = VtxTex;
	gl_Position = PrjnMtx * MVMtx * vPos;
}



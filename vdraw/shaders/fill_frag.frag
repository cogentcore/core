#version 450

// must use mat4 -- mat3 alignment issues are horrible.
// each mat4 = 64 bytes, so full 128 byte total, but only using mat3.
// pack the tex, layer indexes into [3][0-1] of mvp,
// and the fill color into [3][0-3] of uvp
layout(push_constant) uniform Mtxs {
	mat4 mvp;
	mat4 uvp;
};

layout(location = 0) out vec4 outputColor;

void main() {
	vec4 color = uvp[3];
	outputColor = color;
}


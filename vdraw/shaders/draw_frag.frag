#version 450
#extension GL_EXT_nonuniform_qualifier : require

// must use mat4 -- mat3 alignment issues are horrible.
// each mat4 = 64 bytes, so full 128 byte total, but only using mat3.
// pack the tex index into [0][3] of mvp,
// and the fill color into [3][0-3] of uvp
layout(push_constant) uniform Mtxs {
	mat4 mvp;
	mat4 uvp;
};

layout(set = 0, binding = 0) uniform sampler2D Tex[];

layout(location = 0) in vec2 uv;
layout(location = 0) out vec4 outputColor;

void main() {
	int idx = int(mvp[0][3]);
	outputColor = texture(Tex[idx], uv);
}


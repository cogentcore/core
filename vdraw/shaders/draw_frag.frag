#version 450
#extension GL_EXT_nonuniform_qualifier : require

// note: must use mat4 -- mat3 alignment issues are horrible
// this takes 128 bytes, so we need to pack the tex index
// into [0][3] of mvp (only using 3x3 anyway)
layout(push_constant) uniform Mats {
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


#version 450

// must use mat4 -- mat3 alignment issues are horrible.
// each mat4 = 64 bytes, so full 128 byte total, but only using mat3.
// pack the tex, layer indexes into [3][0-1] of mvp,
// and the fill color into [3][0-3] of uvp
layout(push_constant) uniform Mtxs {
	mat4 mvp;
	mat4 uvp;
};

layout(location = 0) in vec2 pos;

void main() {
	vec3 p = vec3(pos, 1);
	gl_Position = vec4(mat3(mvp) * p, 1);
}


#version 450

// note: must use mat4 -- mat3 alignment issues are horrible
// this takes 128 bytes, so we need to pack the tex index
// into [0][3] of mvp (only using 3x3 anyway)
layout(push_constant) uniform Mats {
	mat4 mvp;
	mat4 uvp;
};

layout(location = 0) in vec2 pos;
layout(location = 0) out vec2 uv;

void main() {
	vec3 p = vec3(pos, 1);
	gl_Position = vec4(mat3(mvp) * p, 1);
	uv = (mat3(uvp) * p).xy;
}


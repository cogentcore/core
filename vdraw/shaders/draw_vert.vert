#version 450

// note: must use mat4 -- mat3 alignment issues are horrible
layout(push_constant) uniform Mats {
	mat4 mvp;
	mat4 uvp;
};

layout(location = 0) in vec2 pos;
layout(location = 0) out vec2 uv;

void main() {
	vec4 p = vec4(pos, 1, 1);
	gl_Position = mvp * p;
	uv = (uvp * p).xy;
}


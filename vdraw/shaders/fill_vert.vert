#version 450

// note: must use mat4 -- mat3 alignment issues are horrible
layout(binding = 0) uniform Mats {
	mat4 mvp;
	mat4 uvp;
};

layout(location = 0) in vec2 pos;

void main() {
	vec4 p = vec4(pos, 1, 1);
	gl_Position = mvp * p;
}


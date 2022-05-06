#version 450

layout(binding = 0) uniform Mats {
	mat3 mvp;
	mat3 uvp;
};

layout(location = 0) in vec2 pos;

void main() {
	vec3 p = vec3(pos, 1);
	gl_Position = vec4(mvp * p, 1);
}


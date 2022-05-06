#version 450

layout(binding = 0) uniform Mats {
	mat3 mvp;
	mat3 uvp;
};

layout(location = 0) in vec2 pos;
layout(location = 0) out vec2 uv;

void main() {
	vec3 p = vec3(pos, 0);
	// gl_Position = vec4(mvp * p, 1);
	gl_Position = vec4(p, 1);
	// uv = (uvp * vec3(pos, 1)).xy;
	uv = (vec3(pos, 1)).xy;
}


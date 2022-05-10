#version 450

layout(set = 0, binding = 0) uniform sampler2D tex;

layout(location = 0) in vec2 uv;
layout(location = 0) out vec4 outputColor;

void main() {
	outputColor = texture(tex, uv);
}


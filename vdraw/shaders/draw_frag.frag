#version 450

// todo: not sure if this is needed or what the point is
// precision mediump float;

layout(binding = 2) uniform sampler2D tex;

layout(location = 0) in vec2 uv;

layout(location = 0) out vec4 outputColor;

void main() {
	outputColor = texture(tex, uv);
}


#version 450

precision mediump float;
uniform sampler2D tex;

layout(location = 0) in vec2 uv;

layout(location = 0) out vec4 outputColor;

void main() {
	outputColor = texture(tex, uv);
}


#version 450

// todo: not sure if this is needed or what the point is
// precision mediump float;

layout(binding = 1) uniform ColorIn {
	vec4 color;
};

layout(location = 0) out vec4 outputColor;

void main() {
	outputColor = color;
}


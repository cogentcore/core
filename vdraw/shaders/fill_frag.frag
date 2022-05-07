#version 450

layout(binding = 1) uniform ColorIn {
	vec4 color;
};

layout(location = 0) out vec4 outputColor;

void main() {
	outputColor = color;
}


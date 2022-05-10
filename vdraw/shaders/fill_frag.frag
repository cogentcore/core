#version 450

layout(set = 1, binding = 0) uniform ColorIn {
	vec4 color;
};

layout(location = 0) out vec4 outputColor;

void main() {
	outputColor = vec4(0.5, 0.5, 0.5, 1); // color;
}


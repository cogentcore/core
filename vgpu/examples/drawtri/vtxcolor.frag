#version 450

layout(location = 0) in vector3 fragColor;

layout(location = 0) out vector4 outColor;

void main() {
    outColor = vector4(fragColor, 1.0);
}


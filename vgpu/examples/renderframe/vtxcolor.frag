#version 450

layout(location = 0) in vector3 FragColor;

layout(location = 0) out vector4 OutColor;

void main() {
    OutColor = vector4(FragColor, 1.0);
}


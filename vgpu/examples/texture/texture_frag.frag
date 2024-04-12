#version 450 core
#extension GL_EXT_nonuniform_qualifier : require

layout(push_constant) uniform TexIndexUni {
    int TexIndex;
};

layout(set = 1, binding = 0) uniform sampler2DArray TexSampler[];

layout(location = 0) in vector3 FragColor;
layout(location = 1) in vector2 FragTexCoord;

layout(location = 0) out vector4 OutColor;

void main() {
    OutColor = texture(TexSampler[TexIndex], vector3(FragTexCoord, 0));
    // OutColor = vector4(FragColor, 1.0);
}


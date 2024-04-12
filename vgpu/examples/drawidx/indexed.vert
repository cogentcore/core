#version 450

layout(binding = 0) uniform UniformBufferObject {
    mat4 Model;
    mat4 View;
    mat4 Proj;
} Camera;

layout(location = 0) in vector3 Pos;
layout(location = 1) in vector3 Color;
// layout(location = 2) in vector2 TexCoord;

layout(location = 0) out vector3 FragColor;
// layout(location = 1) out vector2 FragTexCoord;

void main() {
   vector4 pos = vector4(Pos, 1.0);
   gl_Position = Camera.Proj * Camera.View * Camera.Model * pos;
    FragColor = Color;
   // FragTexCoord = TexCoord;
}



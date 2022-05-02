#version 450

layout(binding = 0) uniform UniformBufferObject {
    mat4 Model;
    mat4 View;
    mat4 Proj;
} Camera;

layout(location = 0) in vec3 Pos;
layout(location = 1) in vec3 Color;
// layout(location = 2) in vec2 TexCoord;

layout(location = 0) out vec3 FragColor;
// layout(location = 1) out vec2 FragTexCoord;

void main() {
    gl_Position = Camera.Proj * Camera.View * Camera.Model * vec4(Pos, 1.0);
    FragColor = Color;
   // FragTexCoord = TexCoord;
}



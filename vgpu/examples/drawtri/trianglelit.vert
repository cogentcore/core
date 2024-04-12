#version 450

layout(location = 0) out vector3 fragColor;

// note: original VulkanTutorial uses CW instead of CCW winding order!
vector2 positions[3] = vector2[](
    vector2(-0.5, 0.5),
    vector2(0.5, 0.5),
    vector2(0.0, -0.5)
);

vector3 colors[3] = vector3[](
    vector3(1.0, 0.0, 0.0),
    vector3(0.0, 1.0, 0.0),
    vector3(0.0, 0.0, 1.0)
);

void main() {
    gl_Position = vector4(positions[gl_VertexIndex], 0.0, 1.0);
    fragColor = colors[gl_VertexIndex];
}


#version 450

layout(set = 0, binding = 0) uniform Camera {
    mat4 MVMatrix;
    mat4 MVPMatrix;
    mat4 NormMatrix;
};

layout(location = 0) in vec3 VtxPos;
layout(location = 1) in vec3 VtxNorm;
layout(location = 2) in vec2 VtxTex;
// layout(location = 3) in vec4 VtxColor;

// uniform bool FlipY;
layout(location = 0) out vec4 Pos;
layout(location = 1) out vec3 Norm;
layout(location = 2) out vec3 CamDir;
layout(location = 3) out vec2 TexCoord;

void main() {
	vec4 vPos = vec4(VtxPos, 1.0);
	vec4 vNorm = vec4(VtxNorm, 1.0);
	Pos = MVMatrix * vPos;
	Norm = normalize(NormMatrix * vNorm).xyz;
	CamDir = normalize(-Pos.xyz);
	TexCoord = VtxTex;
// 	if(FlipY) {
// 		TexCoord.y = 1 - TexCoord.y;
// 	}
	
	gl_Position = MVPMatrix * vPos;
}


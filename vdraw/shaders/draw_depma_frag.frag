#version 450
#extension GL_EXT_nonuniform_qualifier : require

// must use mat4 -- mat3 alignment issues are horrible.
// each mat4 = 64 bytes, so full 128 byte total, but only using mat3.
// pack the tex, layer indexes into [3][0-1] of mvp,
// and the fill color into [3][0-3] of uvp
layout(push_constant) uniform Mtxs {
	mat4 mvp;
	mat4 uvp;
};

layout(set = 0, binding = 0) uniform sampler2DArray Tex[];

layout(location = 0) in vec2 uv;
layout(location = 0) out vec4 outputColor;

void main() {
	int idx = int(mvp[3][0]);
	int layer = int(mvp[3][1]);
	outputColor = texture(Tex[idx], vec3(uv,layer));
	// de-pre-multiplied-alpha version: undo
	if (outputColor.a > 0) {
		outputColor.r = outputColor.r / outputColor.a;
		outputColor.g = outputColor.g / outputColor.a;		
		outputColor.b = outputColor.b / outputColor.a;
    }
}


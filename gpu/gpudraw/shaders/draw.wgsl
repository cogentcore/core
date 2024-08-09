// draw.wgsl for gpudraw image draw case

struct MatrixUniform {
	mvp: mat4x4<f32>,
	uvp: mat4x4<f32>,
};

@group(0) @binding(0)
var<uniform> matrix: MatrixUniform;

struct VertexInput {
	@location(0) position: vec2<f32>,
}

struct VertexOutput {
	@builtin(position) clip_position: vec4<f32>,
	@location(0) uv: vec2<f32>,
}

@vertex
fn vs_main(
	model: VertexInput,
) -> VertexOutput {
	var out: VertexOutput;
	let p3 = vec3<f32>(model.position, 1.0);
	let mv3 = mat3x3<f32>(matrix.mvp[0].xyz, matrix.mvp[1].xyz, matrix.mvp[2].xyz);
	out.clip_position = vec4<f32>(mv3 * p3, 1.0);
	let mu3 = mat3x3<f32>(matrix.uvp[0].xyz, matrix.uvp[1].xyz, matrix.uvp[2].xyz);
	out.uv = (mu3 * p3).xy;
	return out;
}

// Fragment

@group(1) @binding(0)
var t_tex: texture_2d<f32>;
@group(1) @binding(1)
var s_tex: sampler;

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	return textureSample(t_tex, s_tex, in.uv);
}


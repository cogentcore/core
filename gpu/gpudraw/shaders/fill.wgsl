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
}

@vertex
fn vs_main(
	model: VertexInput,
) -> VertexOutput {
	var out: VertexOutput;
	let p4 = vec4<f32>(model.position, 0.0, 0.0);
	out.clip_position = matrix.mvp * p4;
	return out;
}

// Fragment

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	return matrix.uvp[3];
}


struct CameraUniform {
	model: mat4x4<f32>,
	view: mat4x4<f32>,
	proj: mat4x4<f32>,
};

@group(1) @binding(0)
var<uniform> camera: CameraUniform;

struct VertexInput {
	@location(0) position: vec3<f32>,
	@location(1) color: vec3<f32>,
	@location(2) tex_coord: vec2<f32>,
}

struct VertexOutput {
	@builtin(position) clip_position: vec4<f32>,
	@location(0) color: vec3<f32>,
	@location(1) tex_coord: vec2<f32>,
}

@vertex
fn vs_main(
	model: VertexInput,
) -> VertexOutput {
	var out: VertexOutput;
	out.clip_position = camera.proj * camera.view * camera.model * vec4<f32>(model.position, 1.0);
	out.color = model.color;
	out.tex_coord = model.tex_coord;
	return out;
}

@group(0) @binding(0)
var t_tex: texture_2d<f32>;
@group(0) @binding(1)
var s_tex: sampler;

@fragment
fn fs_main(in: VertexOutput) -> @location(0) vec4<f32> {
	let itc = vec2<f32>(in.tex_coord.x, 1.0-in.tex_coord.y); // flipy
   return textureSample(t_tex, s_tex, itc);
}

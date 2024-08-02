#include "phong.wgsl"

struct VertexInput {
	@location(0) position: vec3<f32>,
	@location(1) normal: vec3<f32>,
// @location(2) tex_coord: vec2<f32>,
	@location(3) vertex_color: vec4<f32>,
};

struct VertexOutput {
	@builtin(position) clip_position: vec4<f32>,
	@location(0) normal: vec3<f32>,
	@location(1) cam_dir: vec3<f32>,
// @location(2) tex_coord: vec2<f32>,
	@location(3) vertex_color: vec4<f32>,
};

@vertex
fn vs_main(
	model: VertexInput,
) -> VertexOutput {
	var out: VertexOutput;

	let mvm = camera.view * object.matrix;
	let cpos = mvm * vec4<f32>(model.position, 1.0);
	
   out.clip_position = camera.prjn * mvm * vec4<f32>(model.position, 1.0);
	out.normal = (object.world * vec4<f32>(model.normal, 0.0)).xyz;
	out.cam_dir = normalize(-cpos.xyz);
	// out.tex_coord = model.tex_coord;
  	out.vertex_color = model.vertex_color;
	return out;
}

// Fragment

struct FragmentInput {
	@builtin(position) clip_position: vec4<f32>,
	@builtin(front_facing) front_face: bool,
	@location(0) normal: vec3<f32>,
	@location(1) cam_dir: vec3<f32>,
	// @location(2) tex_coord: vec2<f32>,
	@location(3) vertex_color: vec4<f32>,
};

@fragment
fn fs_main(in: FragmentInput) -> @location(0) vec4<f32> {
	let opacity = in.vertex_color.a;
	let clr = SRGBToLinear(in.vertex_color.xyz);
	let shiny  = object.shinyBright.x;
	let reflct = object.shinyBright.y;
	let bright = object.shinyBright.z;
	var normal = in.normal;
	if (in.front_face) {
		normal = -normal;
	}
	return phongModel(in.clip_position, normal, in.cam_dir, clr, clr, shiny, reflct, bright, opacity);
}



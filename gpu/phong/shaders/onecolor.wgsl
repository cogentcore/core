
struct CameraUniform {
   view: mat4x4<f32>,
   prjn: mat4x4<f32>,
};

struct ObjectStorage {
	color: vec4<f32>,
	shinyBright: vec4<f32>,
	emissive: vec4<f32>,
	textureRepeatOff: vec4<f32>,
	matrix: mat4x4<f32>,
};

@group(0) @binding(0)
var<uniform> camera: CameraUniform;

@group(1) @binding(0)
var<uniform> object: ObjectStorage;

struct VertexInput {
	@location(0) position: vec3<f32>,
	@location(1) norm: vec3<f32>,
   @location(2) tex_coord: vec2<f32>,
//	@location(3) vertex_color: vec4<f32>,
};

struct VertexOutput {
	@builtin(position) clip_position: vec4<f32>,
	@location(0) norm: vec3<f32>,
	@location(1) cam_dir: vec3<f32>,
   @location(2) tex_coord: vec2<f32>,
//	@location(3) vertex_color: vec4<f32>,
};

@vertex
fn vs_main(
	model: VertexInput,
) -> VertexOutput {
	var out: VertexOutput;
	
	let mvm = camera.view * object.matrix;
	let cpos = mvm * vec4<f32>(model.position, 1.0);
	// todo: no transpose on GPU, upload in object instead.
	// let normMtx = transpose(inverse(mat3x3<f32>(mvm)));
	
   // out.clip_position = camera.prjn * camera.view * vec4<f32>(model.position, 1.0);
   out.clip_position = camera.prjn * mvm * vec4<f32>(model.position, 1.0);
	// out.norm = normalize(normMtx * model.norm);
	out.norm = model.norm;
	out.tex_coord = model.tex_coord;
	out.cam_dir = normalize(-cpos.xyz);
   // out.vertex_color = model.vertex_color;
	return out;
}

// lights

const MaxLights = 8;
const eps: f32 = 0.00001;

struct NLights {
	ambient: i32,
	dir: i32,
	point: i32,
	spot: i32,
};

struct Ambient {
	color: vec3<f32>,
};

struct Dir {
	color: vec3<f32>, 
	pos: vec3<f32>,
};

struct Point {
	color: vec3<f32>,
	pos: vec3<f32>,
	decay: vec3<f32>, // x = Lin, y = Quad
};

struct Spot {
	color: vec3<f32>,
	pos: vec3<f32>,
	dir: vec3<f32>,
	decay: vec4<f32>, // x = Ang, y = CutAngle, z = Lin, w = Quad
};

@group(2) @binding(0)
var<uniform> nLights: NLights;

@group(2) @binding(1)
var<uniform> ambient: array<Ambient, MaxLights>;

@group(2) @binding(2)
var<uniform> dir: array<Dir, MaxLights>;

@group(2) @binding(3)
var<uniform> point: array<Point, MaxLights>;

@group(2) @binding(4)
var<uniform> spot: array<Spot, MaxLights>;

fn phongModel(pos: vec4<f32>, norm: vec3<f32>, camDir: vec3<f32>, matAmbient: vec3<f32>,  matDiffuse: vec3<f32>, shiny: f32, reflct: f32, bright: f32, opacity: f32) -> vec4<f32> {
	var ambientTotal: vec3<f32>;
	var diffuseTotal: vec3<f32>;
	var specularTotal: vec3<f32>;

	let matSpecular: vec3<f32> = vec3<f32>(reflct);
	
	for (var i = 0; i < nLights.ambient; i++) {
		ambientTotal += ambient[i].color * matAmbient;
	}

	for (var i = 0; i < nLights.dir; i++) {
		// LightDir is the position = - direction of the current light
		let lp4: vec4<f32> = vec4<f32>(dir[i].pos, 0.0);
 		let lightDir: vec3<f32> = normalize(camera.view * lp4).xyz;
		// Calculates the dot product between the light direction and this vertex normal.
		let dotNormal: f32 = dot(lightDir, norm);
		if (dotNormal > eps) {
			diffuseTotal += dir[i].color * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			let refl: vec3<f32> = reflect(-lightDir, norm);
			specularTotal += dir[i].color * matSpecular * pow(max(dot(refl, camDir), 0.0), shiny);
		}
	}

	for (var i = 0; i < nLights.point; i++) {
		// Calculates the direction and distance from the current vertex to this point light.
		let lp4: vec4<f32> = vec4<f32>(point[i].pos, 1.0); // 1 = offset
 		let lightPos: vec3<f32> = (camera.view * lp4).xyz;
		var lightDir: vec3<f32> = lightPos - pos.xyz;
		let lightDist: f32 = length(lightDir);
		// Normalizes the lightDir
		lightDir = lightDir / lightDist;
		// Calculates the attenuation due to the distance of the light
		// Diffuse reflection
		let dotNormal: f32 = dot(lightDir, norm);
		if (dotNormal > eps) {
			let linDecay: f32 = point[i].decay.x;
			let quadDecay: f32 = point[i].decay.y;
			let attenuation: f32 = 1.0 / (1.0 + lightDist * (linDecay +
				quadDecay * lightDist));
			let attenColor: vec3<f32> = point[i].color * attenuation;
			diffuseTotal += attenColor * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			let refl: vec3<f32> = reflect(-lightDir, norm);
			specularTotal += attenColor * matSpecular * pow(max(dot(refl, camDir), 0.0), shiny);
		}
	}
	/*
	for (var i = 0; i < NSpot; i++) {
		// Calculates the direction and distance from the current vertex to this spot light.
		vec4<f32> lp4 = vec4<f32>(SpotLights[i].Pos, 1.0); // 1 = offset
 		vec3<f32> lightPos = (camera.view * lp4).xyz;
		vec3<f32> lightDir = lightPos - vec3<f32>(pos);
		float lightDist = length(lightDir);
		lightDir = lightDir / lightDist;

		// Calculates the angle between the vertex direction and spot direction
		// If this angle is greater than the cutoff the spotlight will not contribute
		// to the final color.
		vec4<f32> ld4 = vec4<f32>(SpotLights[i].Dir, 0.0); // 0 = no offset
 		vec3<f32> lDir = (camera.view * ld4).xyz;

		float angle = acos(dot(-lightDir, lDir));
		float cutAng = SpotLights[i].Decay.y;
		float cutoff = radians(clamp(cutAng, 0.0, 90.0));

		if (angle < cutoff) {
			// Diffuse reflection
			float dotNormal = dot(lightDir, norm);
			if (dotNormal > eps) {
				// Calculates the attenuation due to the distance of the light
				vec4<f32> dk = SpotLights[i].Decay;
				float angDecay = dk.x;
				float linDecay = dk.z;
				float quadDecay = dk.w;
				float attenuation = 1.0 / (1.0 + lightDist * (linDecay +	quadDecay * lightDist));
				float spotFactor = pow(dot(-lightDir, SpotLights[i].Dir), angDecay);
				vec3<f32> attenColor = SpotLights[i].Color * attenuation * spotFactor;
				diffuseTotal += attenColor * matDiffuse * dotNormal;
				// Specular reflection
				vec3<f32> ref = reflect(-lightDir, norm);
				specularTotal += attenColor * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
			}
		}
	}
	*/

	let ambdiff: vec3<f32> = ambientTotal + object.emissive.rgb + diffuseTotal;
	return min(vec4<f32>((bright * ambdiff + specularTotal) * opacity, opacity), vec4<f32>(1.0));
}

// Fragment

struct FragmentInput {
	@builtin(position) clip_position: vec4<f32>,
	@builtin(front_facing) front_face: bool,
	@location(0) norm: vec3<f32>,
	@location(1) cam_dir: vec3<f32>,
   @location(2) tex_coord: vec2<f32>,
//	@location(3) vertex_color: vec4<f32>,
};

/*
@group(3) @binding(0)
var t_tex: texture_2d<f32>;
@group(3) @binding(1)
var s_tex: sampler;
*/

@fragment
fn fs_main(in: FragmentInput) -> @location(0) vec4<f32> {
	let opacity: f32 = object.color.a;
	let clr: vec3<f32> = object.color.rgb;
	let shiny: f32 = object.shinyBright.x;
	let reflct: f32 = object.shinyBright.y;
	let bright: f32 = object.shinyBright.z;
	var norm: vec3<f32> = in.norm;
	if (in.front_face) {
		norm = -norm;
	}
	return phongModel(in.clip_position, norm, in.cam_dir, clr, clr, shiny, reflct, bright, opacity);
	// return textureSample(t_tex, s_tex, in.tex_coords);
}


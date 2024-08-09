// phong.wgsl implements the Blinn-Phong lighting model
// and has the standard Camera and Object structs
// for the phong package.

const MaxLights = 8;
const eps: f32 = 0.00001;

struct NLights {
	ambient: i32,
	directional: i32,
	point: i32,
	spot: i32,
};

struct Ambient {
	color: vec3<f32>,
};

struct Directional {
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
var<uniform> directional: array<Directional, MaxLights>;

@group(2) @binding(3)
var<uniform> point: array<Point, MaxLights>;

@group(2) @binding(4)
var<uniform> spot: array<Spot, MaxLights>;

fn phongModel(pos: vec4<f32>, normal: vec3<f32>, camDir: vec3<f32>, matAmbient: vec3<f32>,  matDiffuse: vec3<f32>, shiny: f32, reflct: f32, bright: f32, opacity: f32) -> vec4<f32> {
	var ambientTotal: vec3<f32>;
	var diffuseTotal: vec3<f32>;
	var specularTotal: vec3<f32>;
	
	let norm = normalize(normal); // make sure

	let matSpecular = vec3<f32>(reflct);
	
	for (var i = 0; i < nLights.ambient; i++) {
		ambientTotal += ambient[i].color * matAmbient;
	}

	for (var i = 0; i < nLights.directional; i++) {
		// LightDir is the position = - direction of the current light
		let lp4 = vec4<f32>(directional[i].pos, 0.0);
 		let lightDir = normalize(camera.view * lp4).xyz;
		// Calculates the dot product between the light direction and this vertex normal.
		let dotNormal = dot(lightDir, norm);
		if (dotNormal > eps) {
			diffuseTotal += directional[i].color * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			let refl = reflect(-lightDir, norm);
			specularTotal += directional[i].color * matSpecular * pow(max(dot(refl, camDir), 0.0), shiny);
		}
	}

	for (var i = 0; i < nLights.point; i++) {
		// Calculates the direction and distance from the current vertex to this point light.
		let lp4 = vec4<f32>(point[i].pos, 1.0); // 1 = offset
 		let lightPos = (camera.view * lp4).xyz;
		var lightDir = lightPos - pos.xyz;
		let lightDist = length(lightDir);
		// Normalizes the lightDir
		lightDir = lightDir / lightDist;
		// Calculates the attenuation due to the distance of the light
		// Diffuse reflection
		let dotNormal = dot(lightDir, norm);
		if (dotNormal > eps) {
			let linDecay = point[i].decay.x;
			let quadDecay = point[i].decay.y;
			let attenuation = 1.0 / (1.0 + lightDist * (linDecay +
				quadDecay * lightDist));
			let attenColor = point[i].color * attenuation;
			diffuseTotal += attenColor * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			let refl = reflect(-lightDir, norm);
			specularTotal += attenColor * matSpecular * pow(max(dot(refl, camDir), 0.0), shiny);
		}
	}

	for (var i = 0; i < nLights.spot; i++) {
		// Calculates the direction and distance from the current vertex to this spot light.
		var lp4 = vec4<f32>(spot[i].pos, 1.0); // 1 = offset
 		let lightPos = (camera.view * lp4).xyz;
		var lightDir = lightPos - pos.xyz;
		let lightDist = length(lightDir);
		lightDir = lightDir / lightDist;

		// Calculates the angle between the vertex direction and spot direction
		// If this angle is greater than the cutoff the spotlight will not contribute
		// to the final color.
		let ld4 = vec4<f32>(spot[i].dir, 0.0); // 0 = no offset
 		let lDir = (camera.view * ld4).xyz;

		let angle = acos(dot(-lightDir, lDir));
		let cutAng = spot[i].decay.y;
		let cutoff = radians(clamp(cutAng, 0.0, 90.0));

		if (angle < cutoff) {
			// Diffuse reflection
			let dotNormal = dot(lightDir, norm);
			if (dotNormal > eps) {
				// Calculates the attenuation due to the distance of the light
				let dk = spot[i].decay;
				let angDecay = dk.x;
				let linDecay = dk.z;
				let quadDecay = dk.w;
				let attenuation = 1.0 / (1.0 + lightDist * (linDecay +	quadDecay * lightDist));
				let spotFactor = pow(dot(-lightDir, spot[i].dir), angDecay);
				let attenColor = spot[i].color * attenuation * spotFactor;
				diffuseTotal += attenColor * matDiffuse * dotNormal;
				// Specular reflection
				let refl = reflect(-lightDir, norm);
				specularTotal += attenColor * matSpecular * pow(max(dot(refl, camDir), 0.0), shiny);
			}
		}
	}


	let ambdiff = ambientTotal + object.emissive.rgb + diffuseTotal;
	return min(vec4<f32>((bright * ambdiff + specularTotal) * opacity, opacity), vec4<f32>(1.0));
}

fn SRGBToLinearComp(value: f32) -> f32 {
	if (value <= 0.04045) {
		return value * 0.0773993808;
	}
 	return pow((value + 0.055) / 1.055, 2.4);
}

fn LinearToSRGBComp(value: f32) -> f32 {
	if (value <= 0.0031308) {
		return value * 12.92;
	}
	return 1.055 * (pow(value, 1.0/2.4)) + 0.055;
}

fn LinearToSRGB(lin: vec3<f32>) -> vec3<f32> {
    return vec3<f32>(LinearToSRGBComp(lin.x), LinearToSRGBComp(lin.y), LinearToSRGBComp(lin.z));
}

fn SRGBToLinear(srgb: vec3<f32>) -> vec3<f32> {
    return vec3<f32>(SRGBToLinearComp(srgb.x), SRGBToLinearComp(srgb.y), SRGBToLinearComp(srgb.z));
}

struct CameraUniform {
   view: mat4x4<f32>,
   prjn: mat4x4<f32>,
};

struct ObjectStorage {
	color: vec4<f32>,
	shinyBright: vec4<f32>,
	emissive: vec4<f32>,
	tiling: vec4<f32>,
	matrix: mat4x4<f32>,
	world: mat4x4<f32>,
};

@group(0) @binding(0)
var<uniform> camera: CameraUniform;

@group(1) @binding(0)
var<uniform> object: ObjectStorage;


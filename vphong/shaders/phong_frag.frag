// #include to implement Blinn-Phong lighting model

// note: all of these vec3 must be padded by an extra float on Go side

#define MAX_LIGHTS 8

layout(set = 3, binding = 0) uniform NLightsU {
	int NAmbient;
	int NDir;
	int NPoint;
	int NSpot;
};

struct Ambient {
	vec3 Color;
};

layout(set = 4, binding = 0) uniform AmbLightsU {
	Ambient AmbLights[MAX_LIGHTS];
};

struct Dir {
	vec3 Color;
	vec3 Pos;
};

layout(set = 4, binding = 1) uniform DirLightsU {
	Dir DirLights[MAX_LIGHTS];
};

struct Point {
	vec3 Color;
	vec3 Pos;
	vec3 Decay; // x = Lin, y = Quad
};

layout(set = 4, binding = 2) uniform PointLightsU {
	Point PointLights[MAX_LIGHTS];
};

struct Spot {
	vec3 Color;
	vec3 Pos;
	vec3 Dir;
	vec4 Decay; // x = Ang, y = CutAngle, z = Lin, w = Quad
};

layout(set = 4, binding = 3) uniform SpotLightsU {
	Spot SpotLights[MAX_LIGHTS];
};

// debugVec3 renders vector to color for debugging values
// void debugVec3(vec3 val, out vec4 clr) {
// 	clr = vec4(0.5 + 0.5 * val, 1.0);
// }

void PhongModel(vec4 pos, vec3 norm, vec3 camDir, vec3 matAmbient, vec3 matDiffuse, vec3 matSpecular, float shiny, out vec3 ambdiff, out vec3 spec) {

	vec3 ambientTotal  = vec3(0.0);
	vec3 diffuseTotal  = vec3(0.0);
	vec3 specularTotal = vec3(0.0);

	matSpecular = vec3(1,1,1);
	
	const float EPS = 0.00001;

    // Workaround for gl_FrontFacing (buggy on Intel integrated GPU's)
    vec3 fdx = dFdx(pos.xyz);
    vec3 fdy = dFdy(pos.xyz);
    vec3 faceNorm = normalize(cross(fdx,fdy));
    if (dot(norm, faceNorm) > 0.0) { // note: reversed from openGL due to vulkan
        norm = -norm;
    }
	// if (gl_FrontFacing) {
	// 	norm = -norm;
	// }

	for (int i = 0; i < NAmbient; i++) {
		ambientTotal += AmbLights[i].Color * matAmbient;
	}

	for (int i = 0; i < NDir; i++) {
		// LightDir is the position = - direction of the current light
		vec4 lp4 = vec4(DirLights[i].Pos, 0.0); // 0 = no offsets
 		vec3 lightDir = normalize((ViewMtx * lp4).xyz);
		// Calculates the dot product between the light direction and this vertex normal.
		float dotNormal = dot(lightDir, norm);
		if (dotNormal > EPS) {
			diffuseTotal += DirLights[i].Color * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			vec3 ref = reflect(-lightDir, norm);
			specularTotal += DirLights[i].Color * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
		}
	}

	for (int i = 0; i < NPoint; i++) {
		// Calculates the direction and distance from the current vertex to this point light.
		vec4 lp4 = vec4(PointLights[i].Pos, 1.0); // 1 = offset
 		vec3 lightPos = (ViewMtx * lp4).xyz;
		vec3 lightDir = lightPos - vec3(pos);
		float lightDist = length(lightDir);
		// Normalizes the lightDir
		lightDir = lightDir / lightDist;
		// Calculates the attenuation due to the distance of the light
		// Diffuse reflection
		float dotNormal = dot(lightDir, norm);
		if (dotNormal > EPS) {
			float linDecay = PointLights[i].Decay.x;
			float quadDecay = PointLights[i].Decay.y;
			float attenuation = 1.0 / (1.0 + lightDist * (linDecay +
				quadDecay * lightDist));
			vec3 attenColor = PointLights[i].Color * attenuation;
			diffuseTotal += attenColor * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			vec3 ref = reflect(-lightDir, norm);
			specularTotal += attenColor * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
		}
	}

	for (int i = 0; i < NSpot; i++) {
		// Calculates the direction and distance from the current vertex to this spot light.
		vec4 lp4 = vec4(SpotLights[i].Pos, 1.0); // 1 = offset
 		vec3 lightPos = (ViewMtx * lp4).xyz;
		vec3 lightDir = lightPos - vec3(pos);
		float lightDist = length(lightDir);
		lightDir = lightDir / lightDist;

		// Calculates the angle between the vertex direction and spot direction
		// If this angle is greater than the cutoff the spotlight will not contribute
		// to the final color.
		vec4 ld4 = vec4(SpotLights[i].Dir, 0.0); // 0 = no offset
 		vec3 lDir = (ViewMtx * ld4).xyz;

		float angle = acos(dot(-lightDir, lDir));
		float cutAng = SpotLights[i].Decay.y;
		float cutoff = radians(clamp(cutAng, 0.0, 90.0));

		if (angle < cutoff) {
			// Diffuse reflection
			float dotNormal = dot(lightDir, norm);
			if (dotNormal > EPS) {
				// Calculates the attenuation due to the distance of the light
				vec4 dk = SpotLights[i].Decay;
				float angDecay = dk.x;
				float linDecay = dk.z;
				float quadDecay = dk.w;
				float attenuation = 1.0 / (1.0 + lightDist * (linDecay +	quadDecay * lightDist));
				float spotFactor = pow(dot(-lightDir, SpotLights[i].Dir), angDecay);
				vec3 attenColor = SpotLights[i].Color * attenuation * spotFactor;
				diffuseTotal += attenColor * matDiffuse * dotNormal;
				// Specular reflection
				vec3 ref = reflect(-lightDir, norm);
				specularTotal += attenColor * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
			}
		}
	}

	ambdiff = ambientTotal + Emissive + diffuseTotal;
	spec = specularTotal;
}


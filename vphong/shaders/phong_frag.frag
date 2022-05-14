// #include to implement Blinn-Phong lighting model

// note: all of these vec3 must be padded by an extra float on Go side

#define MAX_LIGHTS 8

layout(set = 2, binding = 0) uniform NLightsU {
	int NAmbient;
	int NDir;
	int NPoint;
	int NSpot;
};

layout(set = 3, binding = 0) uniform AmbLightsU {
	vec3 Color;
} AmbLights[MAX_LIGHTS];

layout(set = 3, binding = 1) uniform DirLightsU {
	vec3 Color;
	vec3 Dir;
} DirLights[MAX_LIGHTS];

layout(set = 3, binding = 2) uniform PointLightsU {
	vec3 Color;
	vec3 Pos;
	vec3 Decay; // x = Lin, y = Quad
} PointLights[MAX_LIGHTS];

layout(set = 3, binding = 3) uniform SpotLightsU {
	vec3 Color;
	vec3 Pos;
	vec3 Dir;
	vec4 Decay; // x = Ang, y = CutAngle, z = Lin, w = Quad
} SpotLights[MAX_LIGHTS];

// debugVec3 renders vector to color for debugging values
// void debugVec3(vec3 val, out vec4 clr) {
// 	clr = vec4(0.5 + 0.5 * val, 1.0);
// }

void PhongModel(vec4 pos, vec3 norm, vec3 camDir, vec3 matAmbient, vec3 matDiffuse, vec3 matSpecular, float shiny, out vec3 ambdiff, out vec3 spec) {

	vec3 ambientTotal  = vec3(0.0);
	vec3 diffuseTotal  = vec3(0.0);
	vec3 specularTotal = vec3(0.0);

	const float EPS = 0.00001;

    // Workaround for gl_FrontFacing (buggy on Intel integrated GPU's)
    vec3 fdx = dFdx(pos.xyz);
    vec3 fdy = dFdy(pos.xyz);
    vec3 faceNorm = normalize(cross(fdx,fdy));
    if (dot(norm, faceNorm) < 0.0) { // Back-facing
        norm = -norm;
    }
	// if (!gl_FrontFacing) {
	// 	norm = -norm;
	// }

	for (int i = 0; i < NAmbient; i++) {
		ambientTotal += AmbLights[i].Color * matAmbient;
	}

	for (int i = 0; i < NDir; i++) {
		// DirLightDir is the position = direction of the current light
		vec3 lightDir = normalize(DirLights[i].Dir);
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
		vec3 lightDir = PointLights[i].Pos - vec3(pos);
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
			specularTotal += attenColor * matSpecular *	pow(max(dot(ref, camDir), 0.0), shiny);
		}
	}

	for (int i = 0; i < NSpot; i++) {
		// Calculates the direction and distance from the current vertex to this spot light.
		vec3 lightDir = SpotLights[i].Pos - vec3(pos);
		float lightDist = length(lightDir);
		lightDir = lightDir / lightDist;

		// Calculates the angle between the vertex direction and spot direction
		// If this angle is greater than the cutoff the spotlight will not contribute
		// to the final color.
		float angle = acos(dot(-lightDir, SpotLights[i].Dir));
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


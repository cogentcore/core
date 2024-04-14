// #include to implement Blinn-Phong lighting model

// note: all of these vector3 must be padded by an extra float on Go side

#define MAX_LIGHTS 8

layout(set = 1, binding = 0) uniform NLightsU {
	int NAmbient;
	int NDir;
	int NPoint;
	int NSpot;
};

struct Ambient {
	vector3 Color;
};

layout(set = 2, binding = 0) uniform AmbLightsU {
	Ambient AmbLights[MAX_LIGHTS];
};

struct Dir {
	vector3 Color;
	vector3 Pos;
};

layout(set = 2, binding = 1) uniform DirLightsU {
	Dir DirLights[MAX_LIGHTS];
};

struct Point {
	vector3 Color;
	vector3 Pos;
	vector3 Decay; // x = Lin, y = Quad
};

layout(set = 2, binding = 2) uniform PointLightsU {
	Point PointLights[MAX_LIGHTS];
};

struct Spot {
	vector3 Color;
	vector3 Pos;
	vector3 Dir;
	vector4 Decay; // x = Ang, y = CutAngle, z = Lin, w = Quad
};

layout(set = 2, binding = 3) uniform SpotLightsU {
	Spot SpotLights[MAX_LIGHTS];
};

// debugVector3 renders vector to color for debugging values
// void debugVector3(vector3 val, out vector4 clr) {
// 	clr = vector4(0.5 + 0.5 * val, 1.0);
// }

void PhongModel(vector4 pos, vector3 norm, vector3 camDir, vector3 matAmbient, vector3 matDiffuse, vector3 matSpecular, float shiny, float reflct, float bright, float opacity, out vector4 outColor) {

	vector3 ambientTotal  = vector3(0.0);
	vector3 diffuseTotal  = vector3(0.0);
	vector3 specularTotal = vector3(0.0);

	matSpecular =  vector3(reflct,reflct,reflct);
	
	const float EPS = 0.00001;

    // Workaround for gl_FrontFacing (buggy on Intel integrated GPU's)
    vector3 fdx = dFdx(pos.xyz);
    vector3 fdy = dFdy(pos.xyz);
    vector3 faceNorm = normalize(cross(fdx,fdy));
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
		vector4 lp4 = vector4(DirLights[i].Pos, 0.0); // 0 = no offsets
 		vector3 lightDir = normalize((ViewMtx * lp4).xyz);
		// Calculates the dot product between the light direction and this vertex normal.
		float dotNormal = dot(lightDir, norm);
		if (dotNormal > EPS) {
			diffuseTotal += DirLights[i].Color * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			vector3 ref = reflect(-lightDir, norm);
			specularTotal += DirLights[i].Color * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
		}
	}

	for (int i = 0; i < NPoint; i++) {
		// Calculates the direction and distance from the current vertex to this point light.
		vector4 lp4 = vector4(PointLights[i].Pos, 1.0); // 1 = offset
 		vector3 lightPos = (ViewMtx * lp4).xyz;
		vector3 lightDir = lightPos - vector3(pos);
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
			vector3 attenColor = PointLights[i].Color * attenuation;
			diffuseTotal += attenColor * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			vector3 ref = reflect(-lightDir, norm);
			specularTotal += attenColor * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
		}
	}

	for (int i = 0; i < NSpot; i++) {
		// Calculates the direction and distance from the current vertex to this spot light.
		vector4 lp4 = vector4(SpotLights[i].Pos, 1.0); // 1 = offset
 		vector3 lightPos = (ViewMtx * lp4).xyz;
		vector3 lightDir = lightPos - vector3(pos);
		float lightDist = length(lightDir);
		lightDir = lightDir / lightDist;

		// Calculates the angle between the vertex direction and spot direction
		// If this angle is greater than the cutoff the spotlight will not contribute
		// to the final color.
		vector4 ld4 = vector4(SpotLights[i].Dir, 0.0); // 0 = no offset
 		vector3 lDir = (ViewMtx * ld4).xyz;

		float angle = acos(dot(-lightDir, lDir));
		float cutAng = SpotLights[i].Decay.y;
		float cutoff = radians(clamp(cutAng, 0.0, 90.0));

		if (angle < cutoff) {
			// Diffuse reflection
			float dotNormal = dot(lightDir, norm);
			if (dotNormal > EPS) {
				// Calculates the attenuation due to the distance of the light
				vector4 dk = SpotLights[i].Decay;
				float angDecay = dk.x;
				float linDecay = dk.z;
				float quadDecay = dk.w;
				float attenuation = 1.0 / (1.0 + lightDist * (linDecay +	quadDecay * lightDist));
				float spotFactor = pow(dot(-lightDir, SpotLights[i].Dir), angDecay);
				vector3 attenColor = SpotLights[i].Color * attenuation * spotFactor;
				diffuseTotal += attenColor * matDiffuse * dotNormal;
				// Specular reflection
				vector3 ref = reflect(-lightDir, norm);
				specularTotal += attenColor * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
			}
		}
	}

	vector3 ambdiff = ambientTotal + Emissive.rgb + diffuseTotal;
	outColor = min(vector4((bright * ambdiff + specularTotal) * opacity, opacity), vector4(1.0));
}

float SRGBToLinearComp(float value) {
    const float inv_12_92 = 0.0773993808;
    return value <= 0.04045
       ? value * inv_12_92 
       : pow((value + 0.055) / 1.055, 2.4);
}

float LinearToSRGBComp(float value) {
    return value <= 0.0031308
       ? value * 12.92
       : 1.055 * (pow(value, 1.0/2.4)) + 0.055;
}

vector3 LinearToSRGB(vector3 lin) {
    return vector3(LinearToSRGBComp(lin.x), LinearToSRGBComp(lin.y), LinearToSRGBComp(lin.z));
}

vector3 SRGBToLinear(vector3 srgb) {
    return vector3(SRGBToLinearComp(srgb.x), SRGBToLinearComp(srgb.y), SRGBToLinearComp(srgb.z));
}


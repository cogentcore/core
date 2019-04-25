// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"errors"
	"image/color"
	"log"

	"github.com/goki/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/gpu"
)

// https://learnopengl.com/Lighting/Basic-Lighting
// https://en.wikipedia.org/wiki/Blinn%E2%80%93Phong_shading_model

// MatTypes is the global registry of material types.
// external packages can add to this list, after calling
// InitMatTypes after the GPU system has been properly initialized.
var MatTypes map[string]MatType

// MatTypeUnis are shared Uniforms (UBO's) that are used across
// multiple programs: "Lights" and "Camera"
var MatTypeUnis map[string]gpu.Uniforms

// InitMatTypes initializes the default MatTypes registry of core
// material types (and uniforms) that are built-in to the gi3d system.
// Returns true if wasn't already initialized, and error
// if there is some kind of error during initialization.
func InitMatTypes() (bool, error) {
	if MatTypes != nil {
		return false, nil
	}
	var err error
	oswin.TheApp.RunOnMain(func() {
		err = gpu.ActivateShared()
		if err != nil {
			log.Println(err)
			return
		}
		err = initMatTypeUnis()
		if err != nil {
			log.Println(err)
		}
		err = initMatTypesImpl()
		if err != nil {
			log.Println(err)
		}
	})
	return true, err
}

func initMatTypeUnis() error {
	MatTypeUnis = make(map[string]gpu.Uniforms)

	camera := gpu.TheGPU.NewUniforms("Camera")
	camera.AddUniform("CamViewMatrix", gpu.Mat4fUniType, false, 0)
	camera.AddUniform("NormMatrix", gpu.Mat3fUniType, false, 0)
	MatTypeUnis[camera.Name()] = camera

	lights := gpu.TheGPU.NewUniforms("Lights")
	lights.AddUniform("AmbLights", gpu.Vec3fUniType, true, 0)   // 1 per
	lights.AddUniform("DirLights", gpu.Vec3fUniType, true, 0)   // 2 per
	lights.AddUniform("PointLights", gpu.Vec3fUniType, true, 0) // 3 per
	lights.AddUniform("SpotLights", gpu.Vec3fUniType, true, 0)  // 5 per
	MatTypeUnis[lights.Name()] = lights
}

func initMatTypesImpl() error {
	MatTypes = make(map[string]MatType)
	var errs []error
	AddNewMatType(&ColorOpaqueVertexType{}, &errs)
	AddNewMatType(&ColorOpaqueUniformType{}, &errs)
	AddNewMatType(&ColorTransVertexType{}, &errs)
	AddNewMatType(&ColorTransUniformType{}, &errs)
	AddNewMatType(&TextureType{}, &errs)
	AddNewMatType(&TextureGi2DType{}, &errs)

	var erstr string
	for _, er := range errs {
		erstr += er.Error() + "\n"
	}
	if len(erstr) > 0 {
		return errors.New(erstr)
	}
	return nil
}

// AddNewMatType compiles the given MatType and adds any errors to error list
// and adds it to the global MatTypes map, by Name()
func AddNewMatType(mt MatType, errs *[]error) {
	err := mt.Compile()
	MatTypes[mt.Name()] = mt
	if err != nil {
		*errs = append(*errs, err)
	}
}

//////////////////////////////////////////////////////////////////////
//   Lights

// todo: some methods to set / add etc lights

// todo: some methods to set / add etc camera

//////////////////////////////////////////////////////////////////////
//   MatType

// MatType is the interface for material types, which each such type
// managing a GPU Pipeline that implements the shaders to render
// a given material.  MatTypes are initialized once and live in
// a global list of MatTypes, named after their type names.
// Material's use a specific MatType to achieve their rendering.
type MatType interface {
	// Name returns the material type's name, which is the same as
	// the Go type name of the MatType
	Name() string

	// TypeOrder represents the outer-loop material type ordering.
	// It is fixed and determined by the type of material (e.g., transparent
	// comes after opaque)
	TypeOrder() int

	// Pipeline returns the gpu.Pipeline that renders this material
	Pipeline() gpu.Pipeline

	// VtxFragProg returns the gpu.Program for Vertex and Fragment shaders
	// named "VtxFrag"
	VtxFragProg() gpu.Program

	// Compile compiles the gpu.Pipeline programs and shaders for
	// this material -- called during initialization.
	Compile()
}

// Base material type
type MatTypeBase struct {
	Nm    string
	Order int
	Pipe  gpu.Pipeline
}

func (mt *MatTypeBase) Name() string {
	return mb.Nm
}

func (mt *MatTypeBase) TypeOrder() int {
	return mb.Order
}

func (mt *MatTypeBase) Pipeline() gpu.Pipeline {
	return mb.Pipe
}

func (mt *MatTypeBase) VtxFragProg() gpu.Program {
	return mb.Pipe.ProgramByName("VtxFrag")
}

//////////////////////////////////////////////////////////////////////////
//    Types

// ColorOpaqueUniformType is a material with one set of opaque color parameters
// for entire object.  There is one of these per color.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type ColorOpaqueUniformType struct {
	MatTypeBase
}

func (mt *ColorOpaqueUniformType) Compile() error {
	mt.Nm = "ColorOpaqueUniformType"
	mt.Order = 1
	if mt.Pipe != nil {
		mt.Pipe = gpu.NewPipeline(mt.Nm)
		mt.Pipe.AddProgram("VtxFrag")
	}
	pl := mt.Pipe
	pr := pl.ProgramByName("VtxFrag")
	_, err := pr.AddShader(gpu.VertexShader, "Vtx",
		`
#version 330
`+MatTypeUniCamera+MatTypeVtxInPosNorm+MatTypeVtxInColor+
			`
out vec4 Pos;
out vec3 Norm;
out vec3 CamDir;

void main() {
	Pos = CamViewMatrix * vec4(VtxPos, 1.0);
	Norm = normalize(NormMatrix * VtxNorm);
	CamDir = normalize(-Pos.xyz);
	
	gl_Position = Pos;
}
`+"\x00")
	if err != nil {
		return err
	}
	_, err = pr.AddShader(gpu.FragmentShader, "Frag",
		`
#version 330
precision mediump float;
`+MatTypeUniLights+
			`
uniform vec3 Color;
uniform float Shininess;
in vec4 Pos;
in vec3 Norm;
in vec3 CamDir;
out vec4 outputColor;
`+MatTypePhong+
			`
void main() {
    // Inverts the fragment normal if not FrontFacing
    vec3 fragNormal = Normal;
    if (!gl_FrontFacing) {
        fragNormal = -fragNormal;
    }

    // Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
    vec3 Ambdiff, Spec;
    phongModel(Pos, fragNormal, CamDir, Color, Color, Ambdiff, Spec);

    // Final fragment color
    outputColor = min(vec4(Ambdiff + Spec, 1.0), vec4(1.0));
}
`+"\x00")
	if err != nil {
		return err
	}
	pr.AddUniforms(MatTypeUnis["Camera"])
	pr.AddUniforms(MatTypeUnis["Lights"])
	pr.AddUniform("Color", gpu.Vec3fUniType, false, 0)
	pr.AddUniform("Shininess", gpu.FUniType, false, 0)

	pr.AddInput("VtxPos", gpu.Vec3fVecType, gpu.VertexPosition)
	pr.AddInput("VtxNorm", gpu.Vec3fVecType, gpu.VertexNormal)

	pr.SetFragDataVar("outputColor")
	return nil
}

func (mt *ColorOpaqueUniformType) SetColor(color color.Color) error {
	pr := mt.VtxFragProg()
	clr := pr.UniformByName("Color")
	clr.SetValue()
}

func (mt *ColorOpaqueUniformType) SetColorF(color mat32.Color) error {
	pr := mt.VtxFragProg()
	clr := pr.UniformByName("Color")
	clr.SetValue(color)
}

// ColorTransUniformType is a material with one set of transparent color parameters
// for entire object. There is one of these per color.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type ColorTransUniformType struct {
	MatTypeBase
}

// ColorOpaqueVertexType is a material with opaque color parameters per vertex.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type ColorOpaqueVertexType struct {
	MatTypeBase
}

func (mt *ColorOpaqueVertexType) Compile() error {
	mt.Nm = "ColorOpaqueVertexType"
	mt.Order = 1
	pl := gpu.NewPipeline(mt.Nm)
	pr := pl.AddProgram("MainVertFrag")
	mt.Pipe = pl
}

// ColorTransVertexType is a material with transparent color parameters per vertex.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
// Verticies are automatically depth-sorted using GPU-computed depth map.
type ColorTransVertexType struct {
	MatTypeBase
}

// Texture is a texture material -- any objects using the same texture can be rendered
// at the same time.  This is a static texture.
type Texture struct {
	MatTypeBase
	TextureFile string
}

// TextureGi2D is a dynamic texture material driven by a gi.Viewport2D viewport
// anything rendered to the viewport will be projected onto the surface of any
// object using this texture.
type TextureGi2D struct {
	MatTypeBase
	Viewport *gi.Viewport2D
}

//////////////////////////////////////////////////////////////////////
//  Shader code elements

var MatTypeVtxInPosNorm = `in vec3 VtxPos;
in vec3 VtxNorm;
`

var MatTypeVtxInColor = `in vec3 VtxColor;
`

var MatTypeVtxInTex = `in vec2 VtxTex;
`

var MatTypeUniCamera = `layout (std140) uniform Camera
{
    mat4 CamViewMatrix;
    mat3 NormMatrix;
};
`

var MatTypeUniLights = `layout (std140) uniform Lights
{
#if AmbLights_LEN>0
    vec3 AmbLights[AmbLights_LEN];
#endif
#if DirLights_LEN>0
    vec3 DirLights[DirLights_LEN];
    #define DirLightColor(a) DirLights[2*a]
    #define DirLightPos(a) DirLights[2*a+1]
#endif
#if PointLights_LEN>0
    vec3 PointLights[PointLights_LEN];
    #define PointLightColor(a)     PointLights[3*a]
    #define PointLightPos(a)       PointLights[3*a+1]
    #define PointLightLinDecay(a)	  PointLights[3*a+2].x
    #define PointLightQuadDecay(a)	 PointLights[3*a+2].y
#endif
#if SpotLights_LEN>0
    vec3 SpotLights[SpotLights_LEN];
    #define SpotLightColor(a)     SpotLights[5*a]
    #define SpotLightPos(a)       	SpotLights[5*a+1]
    #define SpotLightDir(a)		       SpotLights[5*a+2]
    #define SpotLightAngDecay(a)  	SpotLights[5*a+3].x
    #define SpotLightCutAngle(a)  SpotLights[5*a+3].y
    #define SpotLightLinDecay(a)  SpotLights[5*a+3].z
    #define SpotLightQuadDecay(a) 	SpotLights[5*a+4].x
#endif
};
`

var MatTypePhong = `
/***
 phong lighting model
 Parameters:
    pos:        input vertex position in camera coordinates
    normal:     input vertex normal in camera coordinates
    camDir:     input camera directions
    matAmbient: input material ambient color
    matDiffuse: input material diffuse color
    ambdiff:    output ambient+diffuse color
    spec:       output specular color
 Uniforms:
    Lights
    Shininess
*****/
void phongModel(vec4 pos, vec3 normal, vec3 camDir, vec3 matAmbient, vec3 matDiffuse, out vec3 ambdiff, out vec3 spec) {

    vec3 specularColor = vec3(1.0); // always white anyway
    vec3 ambientTotal  = vec3(0.0);
    vec3 diffuseTotal  = vec3(0.0);
    vec3 specularTotal = vec3(0.0);

#if AmbLights_LEN>0
    for (int i = 0; i < AmbLights_LEN; i++) {
        ambientTotal += AmbLights[i] * matAmbient;
    }
#endif

#if DirLights_LEN>0
    int ndir = DirLights_LEN / 2;
    for (int i = 0; i < ndir; i++) {
        // Diffuse reflection
        // DirLightPos is the direction of the current light
        vec3 lightDir = normalize(DirLightPos(i));
        // Calculates the dot product between the light direction and this vertex normal.
        float dotNormal = max(dot(lightDir, normal), 0.0);
        diffuseTotal += DirLightColor(i) * matDiffuse * dotNormal;
        // Specular reflection
        // Calculates the light reflection vector
        vec3 ref = reflect(-lightDir, normal);
        if (dotNormal > 0.0) {
            specularTotal += DirLightColor(i) * specularColor * pow(max(dot(ref, camDir), 0.0), Shininess);
        }
    }
#endif

#if PointLights_LEN>0
    int npoint = PointLights_LEN / 3;
    for (int i = 0; i < npoint; i++) {
        // Common calculations
        // Calculates the direction and distance from the current vertex to this point light.
        vec3 lightDir = PointLightPos(i) - vec3(pos);
        float lightDist = length(lightDir);
        // Normalizes the lightDir
        lightDir = lightDir / lightDist;
        // Calculates the attenuation due to the distance of the light
        float attenuation = 1.0 / (1.0 + PointLightLinDecay(i) * lightDist +
            PointLightQuadDecay(i) * lightDist * lightDist);
        // Diffuse reflection
        float dotNormal = max(dot(lightDir, normal), 0.0);
        diffuseTotal += PointLightColor(i) * matDiffuse * dotNormal * attenuation;
        // Specular reflection
        // Calculates the light reflection vector
        vec3 ref = reflect(-lightDir, normal);
        if (dotNormal > 0.0) {
            specularTotal += PointLightColor(i) * specularColor *
                pow(max(dot(ref, camDir), 0.0), Shininess) * attenuation;
        }
    }
#endif

#if SpotLights_LEN>0
    int nspot = Spotights_LEN / 5;
    for (int i = 0; i < nspot; i++) {
        // Calculates the direction and distance from the current vertex to this spot light.
        vec3 lightDir = SpotLightPos(i) - vec3(pos);
        float lightDist = length(lightDir);
        lightDir = lightDir / lightDist;

        // Calculates the attenuation due to the distance of the light
        float attenuation = 1.0 / (1.0 + SpotLightLinDecay(i) * lightDist +
            SpotLightQuadDecay(i) * lightDist * lightDist);

        // Calculates the angle between the vertex direction and spot direction
        // If this angle is greater than the cutoff the spotlight will not contribute
        // to the final color.
        float angle = acos(dot(-lightDir, SpotLightDir(i)));
        float cutoff = radians(clamp(SpotLightCutAngle(i), 0.0, 90.0));

        if (angle < cutoff) {
            float spotFactor = pow(dot(-lightDir, SpotLightDir(i)), SpotLightAngDecay(i));

            // Diffuse reflection
            float dotNormal = max(dot(lightDir, normal), 0.0);
            diffuseTotal += SpotLightColor(i) * matDiffuse * dotNormal * attenuation * spotFactor;

            // Specular reflection
            vec3 ref = reflect(-lightDir, normal);
            if (dotNormal > 0.0) {
                specularTotal += SpotLightColor(i) * specularColor * pow(max(dot(ref, camDir), 0.0), Shininess) * attenuation * spotFactor;
            }
        }
    }
#endif

    // Sets output colors
    ambdiff = ambientTotal + diffuseTotal; // note: missing emissive color
    spec = specularTotal;
}
`

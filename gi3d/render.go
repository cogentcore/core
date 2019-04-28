// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"errors"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/gpu"
)

// https://learnopengl.com/Lighting/Basic-Lighting
// https://en.wikipedia.org/wiki/Blinn%E2%80%93Phong_shading_model

// RenderInputs define the locations of the Vectors inputs to the rendering programs
// All Vectors must use these locations so that Mesh data does not depend on which
// program is being used to render it.
type RenderInputs int32

const (
	InVtxPos RenderInputs = iota
	InVtxNorm
	InVtxTexUV
	InVtxColor
	RenderInputsN
)

// RenderClasses define the different classes of rendering
type RenderClasses int32

const (
	RClassOpaqueUniform RenderClasses = iota
	RClassOpaqueVertex
	RClassTexture
	RClassTransUniform
	RClassTransVertex
	RenderClassesN
)

// Renderers is the container for all GPU rendering Programs
// Each scene requires its own version of these because
// the programs need to be recompiled for each specific set
// of lights.
type Renderers struct {
	Unis    map[string]gpu.Uniforms `desc:"uniforms shared across code"`
	Vectors []gpu.Vectors           `desc:"input vectors shared across code, indexed by RenderInputs"`
	Renders map[string]Render       `desc:"collection of Render items"`
}

// SetLights sets the lights and recompiles the programs accordingly
// Must be called with proper context activated
func (rn *Renderers) SetLights(sc *Scene) {
	oswin.TheApp.RunOnMain(func() {
		rn.SetLightsUnis(sc)
		for _, rd := range rn.Renders {
			rd.Compile(rn)
		}
	})
}

// SetMatrix sets the view etc matrix uniforms
// Must be called with appropriate context (window) activated and already on main.
func (rn *Renderers) SetMatrix(pose *Pose) {
	cu := rn.Unis["Camera"]
	mvu := cu.UniformByName("MVMatrix")
	mvu.SetValue(pose.MVMatrix)
	mvpu := cu.UniformByName("MVPMatrix")
	mvpu.SetValue(pose.MVPMatrix)
	nu := cu.UniformByName("NormMatrix")
	nu.SetValue(pose.NormMatrix)
}

// Init initializes the Render programs.
// Must be called with appropriate context (window) activated.
// Returns true if wasn't already initialized, and error
// if there is some kind of error during initialization.
func (rn *Renderers) Init() (bool, error) {
	if rn.Renders != nil {
		return false, nil
	}
	var err error
	oswin.TheApp.RunOnMain(func() {
		rn.InitVectors()
		err = rn.InitUnis()
		if err != nil {
			log.Println(err)
		}
		err = rn.InitRenders()
		if err != nil {
			log.Println(err)
		}
	})
	return true, err
}

func (rn *Renderers) InitVectors() {
	rn.Vectors = make([]gpu.Vectors, RenderInputsN)
	rn.Vectors[InVtxPos] = gpu.TheGPU.NewInputVectors("InVtxPos", int(InVtxPos), gpu.Vec3fVecType, gpu.VertexPosition)
	rn.Vectors[InVtxNorm] = gpu.TheGPU.NewInputVectors("InVtxNorm", int(InVtxNorm), gpu.Vec3fVecType, gpu.VertexNormal)
	rn.Vectors[InVtxTexUV] = gpu.TheGPU.NewInputVectors("InVtxTexUV", int(InVtxTexUV), gpu.Vec2fVecType, gpu.VertexTexcoord)
	rn.Vectors[InVtxColor] = gpu.TheGPU.NewInputVectors("InVtxColor", int(InVtxColor), gpu.Vec4fVecType, gpu.VertexColor)
}

func (rn *Renderers) InitUnis() error {
	rn.Unis = make(map[string]gpu.Uniforms)

	camera := gpu.TheGPU.NewUniforms("Camera")
	camera.AddUniform("MVMatrix", gpu.Mat4fUniType, false, 0)
	camera.AddUniform("NormMatrix", gpu.Mat3fUniType, false, 0)
	camera.AddUniform("MVPMatrix", gpu.Mat4fUniType, false, 0)
	rn.Unis[camera.Name()] = camera

	lights := gpu.TheGPU.NewUniforms("Lights")
	lights.AddUniform("AmbLights", gpu.Vec3fUniType, true, 0)   // 1 per
	lights.AddUniform("DirLights", gpu.Vec3fUniType, true, 0)   // 2 per
	lights.AddUniform("PointLights", gpu.Vec3fUniType, true, 0) // 3 per
	lights.AddUniform("SpotLights", gpu.Vec3fUniType, true, 0)  // 5 per
	rn.Unis[lights.Name()] = lights
	return nil
}

func (rn *Renderers) InitRenders() error {
	rn.Renders = make(map[string]Render)
	var errs []error
	rn.AddNewRender(&RenderUniformColor{}, &errs)
	rn.AddNewRender(&RenderVertexColor{}, &errs)
	rn.AddNewRender(&RenderTexture{}, &errs)

	var erstr string
	for _, er := range errs {
		erstr += er.Error() + "\n"
	}
	if len(erstr) > 0 {
		return errors.New(erstr)
	}
	return nil
}

// AddNewRender compiles the given Render and adds any errors to error list
// and adds it to the global Renders map, by Name()
func (rn *Renderers) AddNewRender(mt Render, errs *[]error) {
	err := mt.Compile(rn)
	rn.Renders[mt.Name()] = mt
	if err != nil {
		*errs = append(*errs, err)
	}
}

// todo: delete mat32.color, mat32.color4

// ColorToVec4f converts given gi.Color to mat32.Vec4 float32's
func ColorToVec4f(clr gi.Color) mat32.Vec4 {
	v := mat32.Vec4{}
	v.X, v.Y, v.Z, v.W = clr.ToFloat32()
	return v
}

// ColorToVec3f converts given gi.Color to mat32.Vec3 float32's
func ColorToVec3f(clr gi.Color) mat32.Vec3 {
	v := mat32.Vec3{}
	v.X, v.Y, v.Z, _ = clr.ToFloat32()
	return v
}

//////////////////////////////////////////////////////////////////////
//   Render

// Render is the interface for a render program, with each managing a
// GPU Pipeline that implements the shaders to render a given material.
// Material's use a specific Render to achieve their rendering.
type Render interface {
	// Name returns the render name, which is the same as the Go type name
	Name() string

	// Pipeline returns the gpu.Pipeline for rendering
	Pipeline() gpu.Pipeline

	// VtxFragProg returns the gpu.Program for Vertex and Fragment shaders
	// named "VtxFrag"
	VtxFragProg() gpu.Program

	// Compile compiles the gpu.Pipeline programs and shaders for
	// this material -- called during initialization.
	Compile(rn *Renderers) error
}

// Base render type
type RenderBase struct {
	Nm   string
	Pipe gpu.Pipeline
}

func (rb *RenderBase) Name() string {
	return rb.Nm
}

func (rb *RenderBase) Pipeline() gpu.Pipeline {
	return rb.Pipe
}

func (rb *RenderBase) VtxFragProg() gpu.Program {
	return rb.Pipe.ProgramByName("VtxFrag")
}

//////////////////////////////////////////////////////////////////////////
//    RenderUniformColor

// RenderUniformColor renders a material with one color for entire object.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type RenderUniformColor struct {
	RenderBase
}

func (rb *RenderUniformColor) Compile(rn *Renderers) error {
	rb.Nm = "RenderUniformColor"
	if rb.Pipe == nil {
		rb.Pipe = gpu.TheGPU.NewPipeline(rb.Nm)
		rb.Pipe.AddProgram("VtxFrag")
	}
	pl := rb.Pipe
	pr := pl.ProgramByName("VtxFrag")
	_, err := pr.AddShader(gpu.VertexShader, "Vtx",
		`
#version 330
`+RenderUniCamera+
			`
layout(location = 0) in vec3 VtxPos;
layout(location = 1) in vec3 VtxNorm;
out vec4 Pos;
out vec3 Norm;
out vec3 CamDir;

void main() {
	vPos = vec4(VtxPos, 1.0);
	Pos = MVMatrix * vPos;
	Norm = normalize(NormMatrix * VtxNorm);
	CamDir = normalize(-Pos.xyz);
	
	gl_Position = MVPMatrix * vPos;
}
`+"\x00")
	if err != nil {
		return err
	}

	_, err = pr.AddShader(gpu.FragmentShader, "Frag",
		`
#version 330
precision mediump float;
`+RenderUniLights+
			`
uniform vec4 Color;
uniform vec3 EmissiveColor;
uniform float Shininess;
in vec4 Pos;
in vec3 Norm;
in vec3 CamDir;
out vec4 outputColor;
`+RenderPhong+
			`
void main() {
    // Inverts the fragment normal if not FrontFacing
    vec3 fragNormal = Norm;
    if (!gl_FrontFacing) {
        fragNormal = -fragNormal;
    }
    float opacity = Color.a;
    vec3 clr = Color.rgb;	
	
    // Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
    vec3 Ambdiff, Spec;
    phongModel(Pos, fragNormal, CamDir, clr, clr, Ambdiff, Spec);

    // Final fragment color
    outputColor = min(vec4(Ambdiff + Spec, opacity), vec4(1.0));
}
`+"\x00")
	if err != nil {
		return err
	}

	pr.AddUniforms(rn.Unis["Camera"])
	pr.AddUniforms(rn.Unis["Lights"])
	pr.AddUniform("Color", gpu.Vec3fUniType, false, 0)
	pr.AddUniform("EmissiveColor", gpu.Vec4fUniType, false, 0)
	pr.AddUniform("Shininess", gpu.FUniType, false, 0)

	pr.SetFragDataVar("outputColor")
	return nil
}

func (rb *RenderUniformColor) SetColors(clr, emmis gi.Color) error {
	pr := rb.VtxFragProg()
	clru := pr.UniformByName("Color")
	clrv := ColorToVec4f(clr)
	clru.SetValue(clrv)
	emsu := pr.UniformByName("EmissiveColor")
	emsv := ColorToVec3f(emmis)
	emsu.SetValue(emsv)
	return nil
}

//////////////////////////////////////////////////////////////////////////
//    RenderVertexColor

// todo: how to do per-face color?

// RenderVertexColor renders color parameters per vertex.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type RenderVertexColor struct {
	RenderBase
}

func (rb *RenderVertexColor) Compile(rn *Renderers) error {
	rb.Nm = "RenderVertexColor"
	pl := gpu.TheGPU.NewPipeline(rb.Nm)
	pl.AddProgram("VtxFrag")
	rb.Pipe = pl
	return nil
}

//////////////////////////////////////////////////////////////////////////
//    RenderTexture

// RenderTexture renders a texture material.
type RenderTexture struct {
	RenderBase
}

func (rb *RenderTexture) Compile(rn *Renderers) error {
	rb.Nm = "RenderTexture"
	pl := gpu.TheGPU.NewPipeline(rb.Nm)
	pl.AddProgram("VtxFrag")
	rb.Pipe = pl
	return nil
}

//////////////////////////////////////////////////////////////////////
//  Shader code elements

var RenderUniCamera = `layout (std140) uniform Camera
{
    mat4 MVMatrix;
    mat3 NormMatrix;
    mat4 MVPMatrix;
};
`

var RenderUniLights = `layout (std140) uniform Lights
{
#if AmbLights_LEN>0
    vec3 AmbLights[AmbLights_LEN];
#endif
#if DirLights_LEN>0
    vec3 DirLights[DirLights_LEN];
    #define DirLightColor(a) DirLights[2*a]
    #define DirLightDir(a) DirLights[2*a+1]
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

var RenderPhong = `
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
        // DirLightDir is the negated position = direction of the current light
        vec3 lightDir = normalize(DirLightDir(i));
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
    ambdiff = ambientTotal + EmissiveColor + diffuseTotal;
    spec = specularTotal;
}
`

// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"errors"
	"log"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/mat32"
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
	InVtxTex
	InVtxColor
	RenderInputsN
)

// RenderClasses define the different classes of rendering
type RenderClasses int32

const (
	RClassNone          RenderClasses = iota
	RClassOpaqueTexture               // textures tend to be in background
	RClassOpaqueUniform
	RClassOpaqueVertex
	RClassTransTexture
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
	NLights int                     `view:"-" desc:"the number of lights when the rendering programs were last compiled -- need to recompile when number of lights change"`
}

// SetLights sets the lights and recompiles the programs accordingly
// Must be called with proper context activated
func (rn *Renderers) SetLights(sc *Scene) {
	if rn.NLights == len(sc.Lights) {
		return
	}
	oswin.TheApp.RunOnMain(func() {
		rn.SetLightsUnis(sc)
		for _, rd := range rn.Renders {
			rd.Compile(rn)
		}
	})
	rn.NLights = len(sc.Lights)
}

// SetMatrix sets the view etc matrix uniforms
// Must be called with appropriate context (window) activated and already on main.
func (rn *Renderers) SetMatrix(pose *Pose) {
	cu := rn.Unis["Camera"]
	cu.Activate()
	mvu := cu.UniformByName("MVMatrix")
	mvu.SetValue(pose.MVMatrix)
	mvpu := cu.UniformByName("MVPMatrix")
	mvpu.SetValue(pose.MVPMatrix)
	nu := cu.UniformByName("NormMatrix")
	nu.SetValue(pose.NormMatrix)
	// fmt.Printf("mv matrix:\n%v\nnorm matrix:\n%v\n", pose.MVMatrix, pose.NormMatrix)
}

// Init initializes the Render programs.
// Must be called with appropriate context (window) activated.
// Returns true if wasn't already initialized, and error
// if there is some kind of error during initialization.
func (rn *Renderers) Init(sc *Scene) (bool, error) {
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
	rn.SetLights(sc) // compiles the shaders assuming lights exist
	return true, err
}

func (rn *Renderers) InitVectors() {
	rn.Vectors = make([]gpu.Vectors, RenderInputsN)
	rn.Vectors[InVtxPos] = gpu.TheGPU.NewInputVectors("InVtxPos", int(InVtxPos), gpu.Vec3fVecType, gpu.VertexPosition)
	rn.Vectors[InVtxNorm] = gpu.TheGPU.NewInputVectors("InVtxNorm", int(InVtxNorm), gpu.Vec3fVecType, gpu.VertexNormal)
	rn.Vectors[InVtxTex] = gpu.TheGPU.NewInputVectors("InVtxTex", int(InVtxTex), gpu.Vec2fVecType, gpu.VertexTexcoord)
	rn.Vectors[InVtxColor] = gpu.TheGPU.NewInputVectors("InVtxColor", int(InVtxColor), gpu.Vec4fVecType, gpu.VertexColor)
}

func (rn *Renderers) InitUnis() error {
	rn.Unis = make(map[string]gpu.Uniforms)

	camera := gpu.TheGPU.NewUniforms("Camera")
	camera.AddUniform("MVMatrix", gpu.Mat4fUniType, false, 0)
	camera.AddUniform("MVPMatrix", gpu.Mat4fUniType, false, 0)
	camera.AddUniform("NormMatrix", gpu.Mat3fUniType, false, 0)
	camera.Activate()
	err := gpu.TheGPU.ErrCheck("camera unis activate")
	rn.Unis[camera.Name()] = camera
	if err != nil {
		return err
	}

	lights := gpu.TheGPU.NewUniforms("Lights")
	lights.AddUniform("AmbLights", gpu.Vec3fUniType, true, 0)   // 1 per
	lights.AddUniform("DirLights", gpu.Vec3fUniType, true, 0)   // 2 per
	lights.AddUniform("PointLights", gpu.Vec3fUniType, true, 0) // 3 per
	lights.AddUniform("SpotLights", gpu.Vec3fUniType, true, 0)  // 5 per
	lights.Activate()
	err = gpu.TheGPU.ErrCheck("lights unis activate")
	rn.Unis[lights.Name()] = lights
	return err
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
func (rn *Renderers) AddNewRender(rb Render, errs *[]error) {
	err := rb.Init(rn)
	rn.Renders[rb.Name()] = rb
	if err != nil {
		*errs = append(*errs, err)
	}
}

// DrawState configures the draw state for rendering -- call when first starting rendering
func (rn *Renderers) DrawState() {
	gpu.Draw.DepthTest(true)
	gpu.Draw.Multisample(true)
}

// Delete deletes GPU resources for renderers
// must be called in context on main
func (rn *Renderers) Delete() {
	for _, rd := range rn.Renders {
		rd.Delete(rn)
	}
	// note: Vectors, Unis don't have deletable resources beyond programs?
}

// ColorToVec4f converts given gist.Color to mat32.Vec4 float32's
func ColorToVec4f(clr gist.Color) mat32.Vec4 {
	v := mat32.Vec4{}
	v.X, v.Y, v.Z, v.W = clr.ToFloat32()
	return v
}

// ColorToVec3f converts given gist.Color to mat32.Vec3 float32's
func ColorToVec3f(clr gist.Color) mat32.Vec3 {
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

	// Init initializes the gpu.Pipeline programs and shaders.
	Init(rn *Renderers) error

	// Compile compiles the gpu.Pipeline programs and shaders.
	Compile(rn *Renderers) error

	// Activate activates this renderer for rendering
	Activate(rn *Renderers)

	// Delete deletes this renderer
	Delete(rn *Renderers)
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

func (rb *RenderBase) Compile(rn *Renderers) error {
	pr := rb.VtxFragProg()
	err := pr.Compile(false) // showSrc -- good for debugging
	if err != nil {
		return err
	}
	return nil
}

func (rb *RenderBase) Activate(rn *Renderers) {
	// fmt.Printf("activating program: %v\n", rb.Nm)
	pr := rb.VtxFragProg()
	pr.Activate()
	gpu.TheGPU.ErrCheck("vtx frag prog activate")
	cmu := rn.Unis["Camera"]
	cmu.Activate()
	cmu.Bind(pr)
	gpu.TheGPU.ErrCheck("camera bind")
	ltu := rn.Unis["Lights"]
	ltu.Activate()
	ltu.Bind(pr)
	gpu.TheGPU.ErrCheck("lights bind")
	pr.Activate()
}

func (rb *RenderBase) Delete(rn *Renderers) {
	rb.Pipe.Delete()
}

//////////////////////////////////////////////////////////////////////////
//    RenderUniformColor

// RenderUniformColor renders a material with one color for entire solid.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type RenderUniformColor struct {
	RenderBase
}

func (rb *RenderUniformColor) Init(rn *Renderers) error {
	rb.Nm = "RenderUniformColor"
	if rb.Pipe == nil {
		rb.Pipe = gpu.TheGPU.NewPipeline(rb.Nm)
		rb.Pipe.AddProgram("VtxFrag")
	}
	pl := rb.Pipe
	pr := pl.ProgramByName("VtxFrag")
	_, err := pr.AddShader(gpu.VertexShader, "Vtx", RenderUniCamera+
		`
layout(location = 0) in vec3 VtxPos;
layout(location = 1) in vec3 VtxNorm;
// layout(location = 2) in vec2 VtxTex;
// layout(location = 3) in vec4 VtxColor;
out vec4 Pos;
out vec3 Norm;
out vec3 CamDir;

void main() {
	vec4 vPos = vec4(VtxPos, 1.0);
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
// precision mediump float;
`+RenderUniLights+
			`
uniform vec4 Color;
uniform vec3 Emissive;
uniform vec3 Specular;
uniform float Shiny;
uniform float Bright;
in vec4 Pos;
in vec3 Norm;
in vec3 CamDir;
out vec4 outputColor;
`+RenderPhong+
			`
			
void main() {
	float opacity = Color.a;
	vec3 clr = Color.rgb;	
	
	// Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
	vec3 Ambdiff, Spec;
	phongModel(Pos, Norm, CamDir, clr, clr, Specular, Shiny, Ambdiff, Spec);

	// Final fragment color -- premultiplied alpha
	outputColor = min(vec4((Bright * Ambdiff + Spec) * opacity, opacity), vec4(1.0));
	// debugVec3(Norm, outputColor);
}
`+"\x00")
	if err != nil {
		return err
	}

	pr.AddUniforms(rn.Unis["Camera"])
	pr.AddUniforms(rn.Unis["Lights"])
	pr.AddUniform("Color", gpu.Vec4fUniType, false, 0)
	pr.AddUniform("Emissive", gpu.Vec3fUniType, false, 0)
	pr.AddUniform("Specular", gpu.Vec3fUniType, false, 0)
	pr.AddUniform("Shiny", gpu.FUniType, false, 0)
	pr.AddUniform("Bright", gpu.FUniType, false, 0)

	pr.SetFragDataVar("outputColor")

	return nil
}

func (rb *RenderUniformColor) SetMat(mat *Material, sc *Scene) error {
	pr := rb.VtxFragProg()
	clru := pr.UniformByName("Color")
	clrv := ColorToVec4f(mat.Color)
	clru.SetValue(clrv)
	emsu := pr.UniformByName("Emissive")
	emsv := ColorToVec3f(mat.Emissive)
	emsu.SetValue(emsv)
	spcu := pr.UniformByName("Specular")
	spcv := ColorToVec3f(mat.Specular)
	spcu.SetValue(spcv)
	shu := pr.UniformByName("Shiny")
	shu.SetValue(mat.Shiny)
	btu := pr.UniformByName("Bright")
	btu.SetValue(mat.Bright)
	gpu.Draw.CullFace(mat.CullFront, mat.CullBack, true) // back face culling, std CCW ordering
	return nil
}

//////////////////////////////////////////////////////////////////////////
//    RenderVertexColor

// RenderVertexColor renders color parameters per vertex.
// This uses the standard Phong color model, with color computed in the
// fragment shader (more accurate, more expensive).
type RenderVertexColor struct {
	RenderBase
}

func (rb *RenderVertexColor) Init(rn *Renderers) error {
	rb.Nm = "RenderVertexColor"
	if rb.Pipe == nil {
		rb.Pipe = gpu.TheGPU.NewPipeline(rb.Nm)
		rb.Pipe.AddProgram("VtxFrag")
	}
	pl := rb.Pipe
	pr := pl.ProgramByName("VtxFrag")
	_, err := pr.AddShader(gpu.VertexShader, "Vtx", RenderUniCamera+
		`
layout(location = 0) in vec3 VtxPos;
layout(location = 1) in vec3 VtxNorm;
// layout(location = 2) in vec2 VtxTex;
layout(location = 3) in vec4 VtxColor;
out vec4 Pos;
out vec3 Norm;
out vec3 CamDir;
out vec4 Color;

void main() {
	vec4 vPos = vec4(VtxPos, 1.0);
	Pos = MVMatrix * vPos;
	Norm = normalize(NormMatrix * VtxNorm);
	CamDir = normalize(-Pos.xyz);
	Color = VtxColor;
	
	gl_Position = MVPMatrix * vPos;
}
`+"\x00")
	if err != nil {
		return err
	}

	_, err = pr.AddShader(gpu.FragmentShader, "Frag",
		`
// precision mediump float;
`+RenderUniLights+
			`
uniform vec3 Emissive;
uniform vec3 Specular;
uniform float Shiny;
uniform float Bright;
in vec4 Pos;
in vec3 Norm;
in vec3 CamDir;
in vec4 Color;
out vec4 outputColor;
`+RenderPhong+
			`
			
void main() {
	float opacity = Color.a;
	vec3 clr = Color.rgb;	
	
	// Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
	vec3 Ambdiff, Spec;
	phongModel(Pos, Norm, CamDir, clr, clr, Specular, Shiny, Ambdiff, Spec);

	// Final fragment color -- premultiplied alpha
	outputColor = min(vec4((Bright * Ambdiff + Spec) * opacity, opacity), vec4(1.0));
}
`+"\x00")
	if err != nil {
		return err
	}

	pr.AddUniforms(rn.Unis["Camera"])
	pr.AddUniforms(rn.Unis["Lights"])
	pr.AddUniform("Emissive", gpu.Vec3fUniType, false, 0)
	pr.AddUniform("Specular", gpu.Vec3fUniType, false, 0)
	pr.AddUniform("Shiny", gpu.FUniType, false, 0)
	pr.AddUniform("Bright", gpu.FUniType, false, 0)

	pr.SetFragDataVar("outputColor")

	return nil
}

func (rb *RenderVertexColor) SetMat(mat *Material, sc *Scene) error {
	gpu.Draw.CullFace(mat.CullFront, mat.CullBack, true) // back face culling, std CCW ordering
	pr := rb.VtxFragProg()
	emsu := pr.UniformByName("Emissive")
	emsv := ColorToVec3f(mat.Emissive)
	emsu.SetValue(emsv)
	spcu := pr.UniformByName("Specular")
	spcv := ColorToVec3f(mat.Specular)
	spcu.SetValue(spcv)
	shu := pr.UniformByName("Shiny")
	shu.SetValue(mat.Shiny)
	btu := pr.UniformByName("Bright")
	btu.SetValue(mat.Bright)
	return nil
}

//////////////////////////////////////////////////////////////////////////
//    RenderTexture

// RenderTexture renders a texture material.
type RenderTexture struct {
	RenderBase
}

func (rb *RenderTexture) Init(rn *Renderers) error {
	rb.Nm = "RenderTexture"
	if rb.Pipe == nil {
		rb.Pipe = gpu.TheGPU.NewPipeline(rb.Nm)
		rb.Pipe.AddProgram("VtxFrag")
	}
	pl := rb.Pipe
	pr := pl.ProgramByName("VtxFrag")
	_, err := pr.AddShader(gpu.VertexShader, "Vtx", RenderUniCamera+
		`
layout(location = 0) in vec3 VtxPos;
layout(location = 1) in vec3 VtxNorm;
layout(location = 2) in vec2 VtxTex;
// layout(location = 3) in vec4 VtxColor;
uniform bool FlipY;
out vec4 Pos;
out vec3 Norm;
out vec3 CamDir;
out vec2 TexCoord;

void main() {
	vec4 vPos = vec4(VtxPos, 1.0);
	Pos = MVMatrix * vPos;
	Norm = normalize(NormMatrix * VtxNorm);
	CamDir = normalize(-Pos.xyz);
	TexCoord = VtxTex;
	if(FlipY) {
		TexCoord.y = 1 - TexCoord.y;
	}
	
	gl_Position = MVPMatrix * vPos;
}
`+"\x00")
	if err != nil {
		return err
	}

	_, err = pr.AddShader(gpu.FragmentShader, "Frag",
		`
// precision mediump float;
`+RenderUniLights+
			`
uniform vec3 Emissive;
uniform vec3 Specular;
uniform float Shiny;
uniform float Bright;
uniform sampler2D Tex;
uniform vec2 TexRepeat;
uniform vec2 TexOff;
in vec4 Pos;
in vec3 Norm;
in vec3 CamDir;
in vec2 TexCoord;
out vec4 outputColor;
`+RenderPhong+
			`
			
void main() {
	vec4 Color = texture(Tex, TexCoord * TexRepeat + TexOff);
	float opacity = Color.a;
	vec3 clr = Color.rgb;	
	
	// Calculates the Ambient+Diffuse and Specular colors for this fragment using the Phong model.
	vec3 Ambdiff, Spec;
	phongModel(Pos, Norm, CamDir, clr, clr, Specular, Shiny, Ambdiff, Spec);

	// Final fragment color -- premultiplied alpha
	outputColor = min(vec4((Bright * Ambdiff + Spec) * opacity, opacity), vec4(1.0));
}
`+"\x00")
	if err != nil {
		return err
	}

	pr.AddUniforms(rn.Unis["Camera"])
	pr.AddUniforms(rn.Unis["Lights"])
	pr.AddUniform("Emissive", gpu.Vec3fUniType, false, 0)
	pr.AddUniform("Specular", gpu.Vec3fUniType, false, 0)
	pr.AddUniform("Shiny", gpu.FUniType, false, 0)
	pr.AddUniform("Bright", gpu.FUniType, false, 0)
	pr.AddUniform("FlipY", gpu.BUniType, false, 0)
	pr.AddUniform("Tex", gpu.IUniType, false, 0)
	pr.AddUniform("TexRepeat", gpu.Vec2fUniType, false, 0)
	pr.AddUniform("TexOff", gpu.Vec2fUniType, false, 0)

	pr.SetFragDataVar("outputColor")

	return nil
}

func (rb *RenderTexture) SetMat(mat *Material, sc *Scene) error {
	if mat.TexPtr != nil {
		mat.TexPtr.Activate(sc, 0)
	}
	pr := rb.VtxFragProg()
	emsu := pr.UniformByName("Emissive")
	emsv := ColorToVec3f(mat.Emissive)
	emsu.SetValue(emsv)
	spcu := pr.UniformByName("Specular")
	spcv := ColorToVec3f(mat.Specular)
	spcu.SetValue(spcv)
	shu := pr.UniformByName("Shiny")
	shu.SetValue(mat.Shiny)
	btu := pr.UniformByName("Bright")
	btu.SetValue(mat.Bright)
	flu := pr.UniformByName("FlipY")
	flu.SetValue(!mat.TexPtr.BotZero()) // flip if not botzero..
	txu := pr.UniformByName("Tex")
	txu.SetValue(0)
	rpu := pr.UniformByName("TexRepeat")
	rpu.SetValue(mat.Tiling.Repeat)
	ofu := pr.UniformByName("TexOff")
	ofu.SetValue(mat.Tiling.Off)
	gpu.Draw.CullFace(mat.CullFront, mat.CullBack, true) // back face culling, std CCW ordering
	return nil
}

//////////////////////////////////////////////////////////////////////
//  Shader code elements

var RenderUniCamera = `
layout (std140) uniform Camera
{
    mat4 MVMatrix;
    mat4 MVPMatrix;
    mat3 NormMatrix;
};
`

var RenderUniLights = `
layout (std140) uniform Lights
{
#if AMBLIGHTS_LEN>0
	vec3 AmbLights[AMBLIGHTS_LEN];
#endif
#if DIRLIGHTS_LEN>0
	vec3 DirLights[DIRLIGHTS_LEN];
	#define DirLightColor(a) DirLights[2*a]
	#define DirLightDir(a) DirLights[2*a+1]
#endif
#if POINTLIGHTS_LEN>0
	vec3 PointLights[POINTLIGHTS_LEN];
	#define PointLightColor(a)     PointLights[3*a]
	#define PointLightPos(a)       PointLights[3*a+1]
	#define PointLightLinDecay(a)	  PointLights[3*a+2].x
	#define PointLightQuadDecay(a)	 PointLights[3*a+2].y
#endif
#if SPOTLIGHTS_LEN>0
	vec3 SpotLights[SPOTLIGHTS_LEN];
	#define SpotLightColor(a)     SpotLights[5*a]
	#define SpotLightPos(a)       	SpotLights[5*a+1]
	#define SpotLightDir(a)       SpotLights[5*a+2]
	#define SpotLightAngDecay(a)  	SpotLights[5*a+3].x
	#define SpotLightCutAngle(a)  SpotLights[5*a+3].y
	#define SpotLightLinDecay(a)  SpotLights[5*a+3].z
	#define SpotLightQuadDecay(a) 	SpotLights[5*a+4].x
#endif
};
`

var RenderPhong = `
// debugVec3 renders vector to color for debugging values
// void debugVec3(vec3 val, out vec4 clr) {
// 	clr = vec4(0.5 + 0.5 * val, 1.0);
// }


void phongModel(vec4 pos, vec3 norm, vec3 camDir, vec3 matAmbient, vec3 matDiffuse, vec3 matSpecular, float shiny, out vec3 ambdiff, out vec3 spec) {

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

	
#if AMBLIGHTS_LEN>0
	for (int i = 0; i < AMBLIGHTS_LEN; i++) {
		ambientTotal += AmbLights[i] * matAmbient;
	}
#endif

#if DIRLIGHTS_LEN>0
	int ndir = DIRLIGHTS_LEN / 2;
	for (int i = 0; i < ndir; i++) {
		// DirLightDir is the position = direction of the current light
		vec3 lightDir = normalize(DirLightDir(i));
		// Calculates the dot product between the light direction and this vertex normal.
		float dotNormal = dot(lightDir, norm);
		if (dotNormal > EPS) {
			diffuseTotal += DirLightColor(i) * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			vec3 ref = reflect(-lightDir, norm);
			specularTotal += DirLightColor(i) * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
		}
	}
#endif

#if POINTLIGHTS_LEN>0
	int npoint = POINTLIGHTS_LEN / 3;
	for (int i = 0; i < npoint; i++) {
		// Calculates the direction and distance from the current vertex to this point light.
		vec3 lightDir = PointLightPos(i) - vec3(pos);
		float lightDist = length(lightDir);
		// Normalizes the lightDir
		lightDir = lightDir / lightDist;
		// Calculates the attenuation due to the distance of the light
		// Diffuse reflection
		float dotNormal = dot(lightDir, norm);
		if (dotNormal > EPS) {
			float attenuation = 1.0 / (1.0 + lightDist * (PointLightLinDecay(i) +
				PointLightQuadDecay(i) * lightDist));
			vec3 attenColor = PointLightColor(i) * attenuation;
			diffuseTotal += attenColor * matDiffuse * dotNormal;
			// Specular reflection -- calculates the light reflection vector
			vec3 ref = reflect(-lightDir, norm);
			specularTotal += attenColor * matSpecular *			pow(max(dot(ref, camDir), 0.0), shiny);
		}
	}
#endif

#if SPOTLIGHTS_LEN>0
	int nspot = SPOTLIGHTS_LEN / 5;
	for (int i = 0; i < nspot; i++) {
		// Calculates the direction and distance from the current vertex to this spot light.
		vec3 lightDir = SpotLightPos(i) - vec3(pos);
		float lightDist = length(lightDir);
		lightDir = lightDir / lightDist;

		// Calculates the angle between the vertex direction and spot direction
		// If this angle is greater than the cutoff the spotlight will not contribute
		// to the final color.
		float angle = acos(dot(-lightDir, SpotLightDir(i)));
		float cutoff = radians(clamp(SpotLightCutAngle(i), 0.0, 90.0));

		if (angle < cutoff) {
			// Diffuse reflection
			float dotNormal = dot(lightDir, norm);
			if (dotNormal > EPS) {
				// Calculates the attenuation due to the distance of the light
				float attenuation = 1.0 / (1.0 + lightDist * (SpotLightLinDecay(i) +
					SpotLightQuadDecay(i) * lightDist));
				float spotFactor = pow(dot(-lightDir, SpotLightDir(i)), SpotLightAngDecay(i));
				vec3 attenColor = SpotLightColor(i) * attenuation * spotFactor;
				diffuseTotal += attenColor * matDiffuse * dotNormal;
				// Specular reflection
				vec3 ref = reflect(-lightDir, norm);
				specularTotal += attenColor * matSpecular * pow(max(dot(ref, camDir), 0.0), shiny);
			}
		}
	}
#endif

	ambdiff = ambientTotal + Emissive + diffuseTotal;
	spec = specularTotal;
}
`

var debugDepth = `
float near = 0.1; 
float far  = 100.0; 
  
float LinearizeDepth(float depth) 
{
    float z = depth * 2.0 - 1.0; // back to NDC 
    return (2.0 * near * far) / (far + near - z * (far - near));	
}

// add to main:
 //    float depth = LinearizeDepth(gl_FragCoord.z) / far; // divide by far for demonstration
 //    outputColor = vec4(vec3(depth), 1.0);
`

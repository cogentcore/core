# Phong is a Blinn-Phong rendering system in vGPU

Blinn-Phong is a standard lighting model that used to be built into OpenGL, and is widely used in 3D graphics: [wikipedia Blinn-Phong](https://en.wikipedia.org/wiki/Blinn%E2%80%93Phong_shading_model),  [learnopengl.com](https://learnopengl.com/Lighting/Basic-Lighting).


Supports 4 different types of lights, with a max of 8 instances of each light type:

* Ambient: light emitted from everywhere -- provides a background of diffuse light bouncing around in all directions.

* Directional: light directed along a given vector (specified as a position of a light shining toward the origin), with no attenuation.  This is like the sun.

* Point: an omnidirectional light with a position and associated decay factors, which divide the light intensity as a function of linear and quadratic distance.  The quadratic factor dominates at longer distances.light radiating out in all directdions from a specific point, 

* Spot: a light with a position and direction and associated decay factors and angles, which divide the light intensity as a function of linear and quadratic distance. The quadratic factor dominates at longer distances.

Meshes are indexed triangles.

There are 3 rendering pipelines:
* Texture: color comes from texture image
* OneColor: a single color for the entire mesh.
* PerVertex: color is provided per vertex by the mesh.

The color model has the following factors:
* `Color` = main color of surface arising from one of the 3 sources listed above, used for both ambient and diffuse color in standard Phong model -- alpha component determines transparency -- note that transparent objects require more complex rendering, to sort objects as a function of depth.
* `Emissive` = color that surface emits independent of any lighting -- i.e., glow -- can be used for marking lights with an object.
* `Shiny` = specular shininess factor -- how focally the surface shines back directional light -- this is an exponential factor, with 0 = very broad diffuse reflection, and higher values (typically max of 128) having a smaller more focal specular reflection. Default is 30.
* `Reflect` = reflectiveness of the surface in the region where specular reflected light is emitted -- 1 for full shiny white reflection (specular) color, 0 = no shine reflection.  The specular color is always set to white, which will reflect the light color accurately.
* `Bright` = overall multiplier on final computed color value -- can be used to tune the overall brightness of various surfaces relative to each other for a given set of lighting parameters.

# Known Issues

There must be at least one texture image, otherwise the Mac VKMolten system triggers this error.  The system automatically adds a dummy image to deal with this.

```
[mvk-error] VK_ERROR_INVALID_SHADER_NV: Unable to convert SPIR-V to MSL:
MSL conversion error: Unsized array of images is not supported in MSL.
```

# Layout of Vars

```
Set: -2
    Role: Vertex
        Var: 0:	Pos	Float32Vector3	(size: 12)	Values: 6
        Var: 1:	Norm	Float32Vector3	(size: 12)	Values: 6
        Var: 2:	Tex	Float32Vector2	(size: 8)	Values: 6
        Var: 3:	Color	Float32Vector4	(size: 16)	Values: 6
    Role: Index
        Var: 4:	Index	Uint32	(size: 4)	Values: 6
Set: -1
    Role: Push
        Var: 0:	PushU	Struct	(size: 128)	Values: 1
Set: 0
    Role: Uniform
        Var: 0:	Mtxs	Struct	(size: 128)	Values: 1
Set: 1
    Role: Uniform
        Var: 0:	NLights	Struct	(size: 16)	Values: 1
Set: 2
    Role: Uniform
        Var: 0:	AmbLights	Struct[8]	(size: 16)	Values: 1
        Var: 1:	DirLights	Struct[8]	(size: 32)	Values: 1
        Var: 2:	PointLights	Struct[8]	(size: 48)	Values: 1
        Var: 3:	SpotLights	Struct[8]	(size: 64)	Values: 1
Set: 3
    Role: SampledTexture
        Var: 0:	Tex	TextureRGBA32	(size: 4)	Values: 3
```

# WebGPU specific considerations

WebGPU does not (yet) support either:
* push constants: https://github.com/gpuweb/gpuweb/issues/75
* arrays of samplers / textures: https://github.com/gpuweb/gpuweb/issues/822

Vulkan-based vGPU took full advantage of push constants, to send up the model matrix and texture index for the current object being rendered.

The general problem here is called "instanced rendering", and there are various ideas about how best to accomplish it.

* https://www.reddit.com/r/vulkan/comments/1bp3tw3/how_to_do_instanced_rendering/
* https://stackoverflow.com/questions/40309914/how-to-send-my-model-matrix-only-once-per-model-to-shaders

Without push constants, there are 3 different options:

1. push it through the vertex buffer, as _per instance_ vec4 values that are then assembled into a 4x4 matrix.  this is what the `go-webgpu-examples/learn-gpu/beginner/tutorial7-instances` does.

2. store it in a storage buffer per object, and use the magic `gl_InstanceID` index to index into that.

3. use different BindGroup configs that reference different Uniform or Storage Values, one per object, and update the bindings before each object.

All options require allocating data on a _per object_ basis, whereas otherwise phong is designed to only require knowing the _meshes_ which drive the vertex data, and are generally much smaller in number than the objects (this is the instancing concept).

Probably the vertex buffer is the fastest of the options -- will investigate.

Next, we have to figure out how to deal with the textures..





# vPhong is a Blinn-Phong rendering system in vGPU

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

# Layout of Vars

```
Set: -2
    Role: Vertex
        Var: 0:	Pos	Float32Vec3	(size: 12)	Vals: 6
        Var: 1:	Norm	Float32Vec3	(size: 12)	Vals: 6
        Var: 2:	Tex	Float32Vec2	(size: 8)	Vals: 6
        Var: 3:	Color	Float32Vec4	(size: 16)	Vals: 6
    Role: Index
        Var: 4:	Index	Uint32	(size: 4)	Vals: 6
Set: -1
    Role: Push
        Var: 0:	PushU	Struct	(size: 128)	Vals: 1
Set: 0
    Role: Uniform
        Var: 0:	Mtxs	Struct	(size: 128)	Vals: 1
Set: 1
    Role: Uniform
        Var: 0:	NLights	Struct	(size: 16)	Vals: 1
Set: 2
    Role: Uniform
        Var: 0:	AmbLights	Struct[8]	(size: 16)	Vals: 1
        Var: 1:	DirLights	Struct[8]	(size: 32)	Vals: 1
        Var: 2:	PointLights	Struct[8]	(size: 48)	Vals: 1
        Var: 3:	SpotLights	Struct[8]	(size: 64)	Vals: 1
Set: 3
    Role: TextureRole
        Var: 0:	Tex	ImageRGBA32	(size: 4)	Vals: 3
```


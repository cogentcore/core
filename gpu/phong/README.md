# Phong is a Blinn-Phong rendering system in vGPU

Blinn-Phong is a standard lighting model that used to be built into OpenGL, and is widely used in 3D graphics: [wikipedia Blinn-Phong](https://en.wikipedia.org/wiki/Blinn%E2%80%93Phong_shading_model),  [learnopengl.com](https://learnopengl.com/Lighting/Basic-Lighting).

# Usage

See [examples/phong] for a working example.

```Go
    // needs a gpu.GPU, gpu.Device, and a gpu.TextureFormat for the render target
    ph := phong.NewPhong(sf.GPU, &sf.Device, &sf.Format)
    
    // add lights
    ph.AddDirectional(math32.NewVector3Color(color.White), math32.Vec3(0, 1, 1))
    ...
    
    // add meshes
    ph.AddMeshFromShape("sphere", shape.NewSphere(.5, 64))
    ...
    
    // add textures
    ph.AddTexture(fn, phong.NewTexture(imgs[i])) // go image.Image, ideally RGBA
    ...

    // add objects
    ph.AddObject(nm, phong.NewObject(&ob.Matrix, ob.Color, color.Black, 30, 1, 1))
    ...
    
    // set the camera matricies
    ph.SetCamera(view, projection)
    
    ph.Config() // configures everything for current settings -- only call once
    
    // can call ph.ConfigMeshes(), ph.ConfigTextures(), ph.ConfigLights()
    // to update any of those elements if they change

    // render, to a surface:
    cmd, rp := ph.RenderStart(view) // uploads updated object data too
    for i, ob := range objects {
        ph.UseObjectIndex(i)
        ph.UseMeshName(ob.Mesh)
        if ob.Texture != "" {
            ph.UseTextureName(ob.Texture)
        } else {
            ph.UseNoTexture()
        }
    }
    rp.End()
    sf.SubmitRender(cmd)
    sf.Present()
```

# Features

Supports 4 different types of lights, with a max of 8 instances of each light type:

* `Ambient`: light emitted from everywhere -- provides a background of diffuse light bouncing around in all directions.

* `Directional`: light directed along a given vector (specified as a position of a light shining toward the origin), with no attenuation.  This is like the sun.

* `Point`: an omnidirectional light with a position and associated decay factors, which divide the light intensity as a function of linear and quadratic distance.  The quadratic factor dominates at longer distances.light radiating out in all directdions from a specific point, 

* `Spot`: a light with a position and direction and associated decay factors and angles, which divide the light intensity as a function of linear and quadratic distance. The quadratic factor dominates at longer distances.

Meshes are indexed triangles.

There are 3 rendering pipelines:
* OneColor: a single color for the entire mesh.
* Texture: color comes from texture image
* PerVertex: color is provided per vertex by the mesh.

The color model has the following factors:
* `Color` = main color of surface arising from one of the 3 sources listed above, used for both ambient and diffuse color in standard Phong model.  The alpha component determines transparency.  Note that transparent objects require more complex rendering, to sort objects as a function of depth.
* `Emissive` = color that surface emits independent of any lighting, i.e., glow, can be used for marking lights with an object.
* `Shiny` = specular shininess factor, which determines how focally the surface shines back directional light. This is an exponential factor, where 0 = very broad diffuse reflection, and higher values (typically max of 128) have a smaller more focal specular reflection. Default is 30.
* `Reflect` = reflectiveness of the surface in the region where specular reflected light is emitted, where 1 = full shiny white reflection (specular) color and 0 = no shine reflection.  The specular color is always set to white, which will reflect the light color accurately.
* `Bright` = overall multiplier on final computed color value. Can be used to tune the overall brightness of various surfaces relative to each other for a given set of lighting parameters.

# Layout of Vars

Without push constants in WebGPU, we maintain an `Objects` group with per-object dynamic offset data, that is selected with the `UseObject*` function on each render step.  This means that the phong system must know about all the objects in advance.

```
Group: -2 Vertex
    Role: Vertex
        Var: 0:	Pos	Float32Vector3	(size: 12)	Values: 8
        Var: 1:	Normal	Float32Vector3	(size: 12)	Values: 8
        Var: 2:	TexCoord	Float32Vector2	(size: 8)	Values: 8
        Var: 3:	VertexColor	Float32Vector4	(size: 16)	Values: 8
    Role: Index
        Var: 0:	Index	Uint32	(size: 4)	Values: 8
Group: 0 Camera
    Role: Uniform
        Var: 0:	Camera	Struct	(size: 128)	Values: 1
Group: 1 Objects
    Role: Uniform
        Var: 0:	Object	Struct	(size: 192)	Values: 1
Group: 2 Lights
    Role: Uniform
        Var: 0:	NLights	Struct	(size: 16)	Values: 1
        Var: 1:	Ambient	Struct[8]	(size: 16)	Values: 1
        Var: 2:	Directional	Struct[8]	(size: 32)	Values: 1
        Var: 3:	Point	Struct[8]	(size: 48)	Values: 1
        Var: 4:	Spot	Struct[8]	(size: 64)	Values: 1
Group: 3 Texture
    Role: SampledTexture
        Var: 0:	TexSampler	TextureRGBA32	(size: 4)	Values: 1
```



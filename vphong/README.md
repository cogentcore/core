# vPhong is a Blinn-Phong rendering system in vGPU

Supports 4 different types of lights:
* Ambient: light emitted from everywhere
* Directional: 
* Point
* Spot

Meshes are indexed triangles.

There are 3 rendering pipelines:
* Texture: color comes from texture image
* OneColor: a single color for the entire mesh.
* PerVertex: color is provided per vertex by the mesh.


# Layout of Vars

```
Set: -2 - VertexSet
    Role: Vertex
        Var: 0:	Pos	Float32Vec3	(size: 12)	Vals: 6
        Var: 1:	Norm	Float32Vec3	(size: 12)	Vals: 6
        Var: 2:	Tex	Float32Vec2	(size: 8)	Vals: 6
        Var: 3:	Color	Float32Vec4	(size: 16)	Vals: 6
    Role: Index
        Var: 4:	Index	Uint32	(size: 4)	Vals: 6
Set: -1 -- PushSet
    Role: Push
        Var: 0:	TexPush	Struct	(size: 24)	Vals: 1
Set: 0
    Role: Uniform
        Var: 0:	Mtxs	Struct	(size: 192)	Vals: 6
Set: 1
    Role: Uniform
        Var: 0:	Color	Struct	(size: 64)	Vals: 5
Set: 2
    Role: Uniform
        Var: 0:	ViewMtx	Struct	(size: 64)	Vals: 1
Set: 3
    Role: Uniform
        Var: 0:	NLights	Struct	(size: 16)	Vals: 1
Set: 4
    Role: Uniform
        Var: 0:	AmbLights	Struct[8]	(size: 16)	Vals: 1
        Var: 1:	DirLights	Struct[8]	(size: 32)	Vals: 1
        Var: 2:	PointLights	Struct[8]	(size: 48)	Vals: 1
        Var: 3:	SpotLights	Struct[8]	(size: 64)	Vals: 1
Set: 5
    Role: TextureRole
        Var: 0:	Tex	ImageRGBA32	(size: 4)	Vals: 3
```


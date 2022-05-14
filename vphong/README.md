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


# vshape: Compositional 3D shape library

`vshape` provides a library of 3D shapes, built from indexed triangle meshes, which can be added together in `ShapeGroup` lists.  Each `Shape` can report the number of points and indexes based on configured parameters, and keeps track of its offset within an overall `mat32.ArrayF32` allocated based on total numbers.  In this way, separate Allocate then Configure phases are supported, as required by the vgpu Memory allocation system.

Basic building blocks like a Plane have standalone methods, in addition to Shape elements.



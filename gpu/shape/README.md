# shape: Compositional 3D shape library

`shape` provides a library of 3D shapes, built from indexed triangle meshes, which can be added together in `ShapeGroup` lists.  Each `Shape` can report the number of points and indexes based on configured parameters, and keeps track of its offset within an overall `math32.ArrayF32` allocated based on total numbers.

The `Mesh` interface defines the set of functions that render shapes to arrays of vertex, normal, and texture coordinate points, which each different specific shape type implements.  This interface is used in the [phong](../phong) system for setting meshes.

It only has a dependency on the [math32](https://cogentcore.org/core/math32) package and could be used for anything.

Basic building blocks (e.g., Plane, SphereSector) have standalone methods, in addition to Shape elements.

Here are the shapes:

* Plane
* Box
* Sphere (including various partial segments thereof)
* Cylinder, Cone (including different size top and bottom radii, e.g., Cones)
* Capsule: a cylinder with half-sphere caps -- good for simple body segments
* Torus
* Lines: uses a box to draw 3D lines
* Group: combines multiple shapes.


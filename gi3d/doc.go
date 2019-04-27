// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gi3d provides a 3D scenegraph for the GoGi GUI framework.

The scenegraph is rooted at a gi3d.Scene node which is a gi.Viewport2D where the scene
is rendered, similar to the svg.SVG node which is also a gi.Viewport2D where SVG drawings
are rendered.

Children of the Scene are either Group or Object -- Group applies a transform (position
size, rotation etc) to everything under it, while Object has its own transform
and a Material and Mesh which define the color / texture and shape of the object.

Solid shapes that are have uniform Surface (color) properties on all surfaces can
be a single Object, but if you need e.g., different textures for each side of a box
then that must be represented as a Group of Objects using Plane Mesh's, each of
which can then bind to a different texture.

All Meshes are indexed triangles.  Objects have computed bounding boxes and can be
selected etc, and maintain their own verticies and normals that reflect their specific
transform relative to the standard Mesh object (which maintains the rest of the relevant
index and texture coords, and, optionally, shared per-vertex colors, which can also
be set individually on the Object if that makes more sense).

The Scene maintains a map of uniquely-named Material and Mesh elements, and Objects
refer to those by name.  This allows for efficient re-use of materials and meshes
across multiple objects -- the object basically instantiates a unique combination
of these properties, and has a unique location / orientation in the scene.

The Scene also holds the Camera and Lights for rendering.

Rendering is performed over an optimized ordering of the Materials, as each
material requires its own specific Shader programs, organized as a gpu.Pipeline,
using the oswin/gpu interface to the underlying rendering system (OpenGL or,
later, Vulkan).

Updating of individual nodes is optimized using the ki.Node update signals
to only update Objects that have changed.  Compute shader programs are used
to update verticies based on geometry changes.
*/
package gi3d

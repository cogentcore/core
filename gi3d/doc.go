// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gi3d provides a 3D scenegraph for the GoGi GUI framework.

The scenegraph is rooted at a gi3d.Scene node which is like gi.Viewport2D where the scene
is rendered, similar to the svg.SVG node for SVG drawings.

Children of the Scene are either Group or Object -- Group applies a transform (position
size, rotation etc) to everything under it, while Object has its own transform
and a Material and Mesh which define the color / texture and shape of the object.

Solid shapes that are have uniform Material color properties on all surfaces can
be a single Object, but if you need e.g., different textures for each side of a box
then that must be represented as a Group of Objects using Plane Mesh's, each of
which can then bind to a different Texture via their Material settings.

All Meshes are indexed triangles, and there are standard shapes such as Box,
Sphere, Cylinder, Capsule, and Line (rendered as a thin Box with end points specified).
Objects have computed bounding boxes and can be selected etc.

The Scene maintains a map of uniquely-named Mesh elements, and Objects
refer to those by name.  The same goes for Textures.
This allows for efficient re-use of meshes and textures across multiple objects.
The object basically instantiates a unique combination
of these properties, and has a unique location / orientation in the scene.

The Scene also holds the Camera and Lights for rendering, and is fully in
charge of the rendering process by iterating over the scene elements and
culling out-of-view elements, ordering opaque then transparent elements, etc.

There are standard Render types that manage the relevant GPU programs /
Pipelines to do the actual rendering, depending on Material and Mesh properties
(e.g., uniform vs per-vertex color vs. texture).
*/
package gi3d

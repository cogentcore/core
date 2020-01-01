// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gi3d provides a 3D scenegraph for the GoGi GUI framework.

The scenegraph is rooted at a gi3d.Scene node which is like gi.Viewport2D,
where the scene is rendered, similar to the svg.SVG node for SVG drawings.

Children of the Scene are either Group or Solid -- Group applies a transform
(position size, rotation etc) to everything under it, while Solid has its
own transform and a Material and points to a Mesh which define the color /
texture and shape of the solid.

Objects that have uniform Material color properties on all surfaces can
be a single Solid, but if you need e.g., different textures for each side of a box
then that must be represented as a Group of Solids using Plane Mesh's, each of
which can then bind to a different Texture via their Material settings.

Thus, in most cases, a discrete "object" is a Group, often with multiple levels of
sub-Group's containing various Solids.

Groups and Solids have computed bounding boxes, in both local and World reference
frames, and can be selected etc.  Bounding boxes are used for visibility and event
selection.

All Meshes are stored directly on the Scene, and must have unique names, as they
are referenced from Solids by name.  The Mesh contains all the verticies, etc
that define a shape, and are the major memory-consuming elements of the scene
(along with textures).  Thus, the Solid is very lightweight and just points to
the Mesh, so Meshes can be reused across multiple Solids for efficiency.

Meshes are only indexed triangles, and there are standard shapes such as Box,
Sphere, Cylinder, Capsule, and Line (rendered as a thin Box with end
points specified).

Textures are also stored by unique names on the Scene, and the Material can
optionally refer to a texture -- likewise allowing efficient re-use across
different Solids.

The Scene also contains a Library of uniquely-named "objects" (Groups)
which can be loaded from 3D object files, and then added into the scenegraph as
needed.  Thus, a typical, efficient workflow is to initialize a Library of such
objects, and then configure the specific scene from these objects.  The library
objects are Cloned into the scenegraph -- because the Group and Solid nodes
are lightweight, this is all very efficient.

The Scene also holds the Camera and Lights for rendering -- there is no point in
putting these out in the scenegraph -- if you want to add a Solid representing
one of these elements, you can easily do so.

The Scene is fully in charge of the rendering process by iterating over the
scene elements and culling out-of-view elements, ordering opaque then
transparent elements, etc.

There are standard Render types that manage the relevant GPU programs /
Pipelines to do the actual rendering, depending on Material and Mesh properties
(e.g., uniform vs per-vertex color vs. texture).
*/
package gi3d

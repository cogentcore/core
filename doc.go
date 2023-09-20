// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gi3d provides a 3D scenegraph for the GoGi GUI framework.

The scenegraph is rooted at a gi3d.Scene node which is like gi.Viewport2D,
where the scene is rendered, similar to the svg.SVG node for SVG drawings.

Children of the Scene are Node3D nodes, with Group and Solid as the main
subtypes.  Node3DBase is the base implementation, which has a Pose for
the full matrix transform of relative position, scale, rotation, and
bounding boxes at multiple levels.

* Group is a container -- most discrete objects should be organized
into a Group, with Groups of Solids underneath.
For maximum efficiency it is important to organize large scenegraphs
into hierarchical groups by location, so that regions can be
pruned for rendering.  The Pose on the Group is inherited by everything
under it, so things can be transformed at different levels as well.

* Solid has a Material to define the color / texture of the solid,
and the name of a Mesh that defines the shape.

Objects that have uniform Material color properties on all surfaces can
be a single Solid, but if you need e.g., different textures for each side of a box
then that must be represented as a Group of Solids using Plane Mesh's, each of
which can then bind to a different Texture via their Material settings.

Node bounding boxes are in both local and World reference frames, and are
used for visibility and event selection.

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

Any change to the Mesh after first initialization (Init3D) must be activated
by calling Scene.InitMesh(nm) or Scene.InitMeshes() to redo all.  The Update
method on the Scene does Init3D and re-renders.

Mouse events are handled by the standard GoGi Window event dispatching
methods, based on bounding boxes which are always updated -- this greatly
simplifies gui interactions.  There is default support for selection and
Pose manipulation handling -- see manip.go code and Node3DBase's
ConnectEvents3D which responds to mouse clicks.
*/
package gi3d

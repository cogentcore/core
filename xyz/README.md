# XYZ

[![Go Reference](https://pkg.go.dev/badge/goki.dev/xyz/xyz.svg)](https://pkg.go.dev/goki.dev/xyz)

`xyz` is a 3D graphics framework written in Go.  It was originally developed as `gi3d`, included in the [GoGi](https;//goki.dev/GoGi) GUI framework, but is now a separate standalone package that renders to an offscreen Vulkan framebuffer, which can easily be converted into a Go `image.RGBA`.  The `xyzv` package in GoGi provides an integration of xyz in GoGi, for dynamic and efficient 3D rendering within 2D GUI windows.

`xyz` is built on the [vgpu](https://github.com/goki/vgpu) Vulkan GPU framework, and uses the [goki](https://github.com/goki/ki) tree structure for the scenegraph.  It currently supports standard Phong-based rendering with different types of lights and materials.  It is designed for scientific and other non-game 3D applications, and lacks almost all of the advanced features one would expect in a modern 3D graphics framework.  Thus, its primary advantage is its simplicity and support for directly programming 3D visualizations in Go, its broad cross-platform support across all major desktop and mobile platforms, and the use of Vulkan which is highly efficient.  See [eve](https://github.com/emer/eve) for a virtual environment framework built on top of xyz, which provides a physics engine for simulating 3D worlds (for use in training neural network models).

# Basic elements

The 3D scenegraph is rooted at a `xyz.Scene` node, which contains all of the shared `Mesh` and `Texture` elements, along with `Lights`, `Camera` and a `Library` of loaded 3D objects that can be referenced by name.

Children of the Scene are `Node` nodes, with `Group` and `Solid` as the main subtypes.  `NodeBase` is the base implementation, which has a `Pose `for the full matrix transform of relative position, scale, rotation, and bounding boxes at multiple levels.

* `Group` is a container -- most discrete objects should be organized into a Group, with Groups of Solids underneath.  For maximum efficiency it is important to organize large scenegraphs into hierarchical groups by location, so that regions can be pruned for rendering.  The Pose on the Group is inherited by everything under it, so things can be transformed at different levels as well.

* `Solid` has a `Material` to define the color / texture of the solid, and the name of a `Mesh` that defines the shape.  Thus, the scenegraph elements are very lightweight and basically point (by name) to shared larger resources on the Scene, where they can be efficiently shared among any number of Solid elements.

Objects that have uniform Material color properties on all surfaces can be a single Solid, but if you need e.g., different textures for each side of a box then that must be represented as a Group of Solids using Plane Meshes, each of which can then bind to a different Texture via their Material settings.

Node bounding boxes are in both local and World reference frames, and are used for visibility (render pruning) and event selection.  Very large scenegraphs can be efficiently rendered by organizing elements into larger and larger Group collections: pruning automatically happens at the highest level where everything below is invisible.  This same principle is used in `eve` for efficient collision detection.

Meshes are *exclusively* defined by indexed triangles, and there are standard shapes such as `Box`, `Sphere`, `Cylinder`, `Capsule`, and `Lines` (rendered as thin boxes with end points specified).

`Texture`s are loaded from Go image files, stored by unique names on the Scene, and the Node-specific Material can optionally refer to a texture -- likewise allowing efficient re-use across different Solids.

The Scene also contains a `Library` of uniquely-named "objects" (Groups) which can be loaded from 3D object files, and then added into the scenegraph as needed.  Thus, a typical, efficient workflow is to initialize a Library of such objects, and then configure the specific scene from these objects.  The library objects are Cloned into the scenegraph so they can be independently configured and manipulated there.  Because the Group and Solid nodes are lightweight, this is all very efficient.

The Scene also holds the Camera and Lights for rendering -- there is no point in putting these out in the scenegraph -- if you want to add a Solid representing one of these elements, you can easily do so.

The Scene is fully in charge of the rendering process by iterating over the scene elements and culling out-of-view elements, ordering opaque then transparent elements, etc.

There are standard Render types that manage the relevant GPU programs / Pipelines to do the actual rendering, depending on Material and Mesh properties (e.g., uniform vs per-vertex color vs. texture).

# Scenegraph Structure

* `Scene` is the root node of the 3D scenegraph.

    + `Camera` is a field on the Scene that has all the current camera settings.  By default the camera does a naturalistic Perspective projection, but you can enable Orthographic by ticking the `Ortho` button -- you will generally need to reduce the Far plane value to be able to see anything -- the Ortho projection shows you the entire scene within those two planes, and it scales accordingly to be able to fit everything.

    + `Lights` contain the lighting parameters for the scene -- if you don't have any lights, everything will be dark!
        + `Ambient` lights contribute generic lighting to every surface uniformly -- usually have this at a low level to represent scattered light that has bounced around everywhere.
        + `Dir` ectional lights represent a distant light-source like the sun, with effectively parallel rays -- the position of the light determines its direction by pointing back from there to the origin -- think of it as the location of the sun.  Only the *normal* direction value is used so the magnitude of the values doesn't matter.
        + `Point` lights have a specific position and radiate light uniformly in all directions from that point, with both a linear and quadratic decay term.
        + `Spot` lights are the most sophisticated lights, with both a position and direction, and an angular cutoff so light only spreads out in a cone, with appropriate decay factors.

    + `Meshes` are the library of `Mesh` shapes that can be used in the scene.  These provide the triangle-based surfaces used to define shapes.  The `shape.go` code provides the basic geometric primitives such as `Box`, `Sphere`, `Cylinder`, etc, and you can load mesh shapes from standard `.obj` files as exported by almost all 3D rendering programs.  You can also write code to generate your own custom / dynamic shapes, as we do with the `NetView` in the [emergent](https://github.com/emer/emergent) neural network simulation system.
    
    + `Textures` are the library of `Texture` files that define more complex colored surfaces for objects.  These can be loaded from standard image files.
    
    + `Solid`s are the Children of the Scene, and actually determine the content of the 3D scene.  Each Solid has a `Mesh` field with the name of the mesh that defines its shape, and a `Mat` field that determines its material properties (Color, Texture, etc).  In addition, each Solid has its own `Pose` field that determines its position, scale and orientation within the scene.  Because each `Solid` is a `ki.Ki` tree node, it can contain other scene elements as its Children -- they will inherit the `Pose` settings of the parent (and so-on up the tree -- all poses are cumulative) but *not* automatically any material settings.  You can call `CopyMatToChildren` if you want to apply the current materials to the children nodes.  And use Style parameters to set these according to node name or Class name.

    + `Group`s can be used to apply `Pose` settings to a set of Children that are all grouped together (e.g., a multi-part complex object can be moved together etc by putting a set of `Solid`s into a Group)

# Events, Selection, Manipulation

Mouse events are handled by the standard GoGi Window event dispatching methods, based on bounding boxes which are always updated -- this greatly simplifies gui interactions.  There is default support for selection and `Pose` manipulation handling -- see `manip.go` code and `NodeBase`'s `ConnectEvents3D` which responds to mouse clicks.

# Embedded 2D Viewport

A full 2D GoGi GUI can be embedded within a 3D scene using the `xyzv.Embed2D` Node type, which renders a `gi.Scene` onto a Texture projected onto a Plane.  It captures events within its own bounding box, and translates them into coordinates for the 2D embedded gui. This allows full 2D interactive control within whatever perspective is present in the 3D scene.  However, things like cursors and popups render in the flat 2D screen and are only approximately located.

In addition to interactive guis, the embedded 2D node can be used for rendering full SVG graphics to a texture.



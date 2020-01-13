# GoGi 3D Demo

This demo shows the basic functionality of the 3D rendering framework in GoGi.  

The gopher object that is imported is borrowed from: https://github.com/g3n/engine, which also served as the source / reference for a lot of the 3D code, including especially the mat32 math library, the file importing code, and the opengl shader programs.

## Installation

The usual Go install procedure will work -- see [Install](https://github.com/goki/gi/wiki/Install) for details.

## Animation and embedded 2D GUI

As you can see, a 2D gui can be embedded within the 3D scene, in this case to control the animation.  This affords unique opportunities for embedding active gui elements within scenes.  Technically, it is achieved by updating a texture projected on a flat plane surface, which is rendered by the 2D gui.  The 2D framework is sufficiently encapsulated that it can be embedded like this with just a bit of surrounding code to manage events within the altered geometry (which nevertheless causes some remaining challenges for elements like popups and the cursor which operate strictly in the overlaying 2D).

You can see the code for how the animation works -- it just updates positions and re-renders the scene, using a `time.Ticker` to trigger movements and updates.

The updating signaling is robust to multiple sources of update signals, so you can do all the other interaction with the scene while the animation is ongoing.

## Camera Navigation

The default camera navigation controls allow you to move around the scene.  To get the keyboard versions to work, you need to set the focus on the 3D scene, which can be done by clicking into it or tabbing or using the arrow keys as usual in GoGi.  These are the different modes of movement:

* **Orbit** moves the camera around in a sphere, with it always pointing at the same **Target** location (the origin by default).

* **Pan** moves the camera *and* the target together along the horizontal / vertical direction defined by the current viewing plane (the plane of the screen).

* **AxisPan** moves the camera and the target along the *world* horizontal / vertical (X, Y) axes.

* **TargetPan** moves the target in *world* horizontal / vertical / depth (X, Y, Z) axes, and tells the camera to LookAt that new target location.

Here are the default bindings (you can see the `gi3d.Scene NavEvents` method in `gi3d/scene.go` for how it works -- set the `NoNav` flag to true to disable it and you can write a different mapping by either making a new version of the Scene type or wrapping the Scene in a custom outer type that handles all the keyboard mappings.

* Mouse or keyboard arrows with no modifiers does *Orbit* rotation
* `Shift+Mouse` / `Shift+Arrow` = *Pan*
* `Ctrl+Mouse` / `Alt+Arrow` = *AxisPan*
* `Alt+Mouse` / `Ctrl+Arrow` = *TargetPan*
* `+ / -` = Zoom in / out along current view axis.
* `Alt++` / `Alt+-` = *TargetPan* along depth Z axis
* `Space` = reset to the Defaults initial camera location (+10 in Z, Up is +Y, and looking at origin)

These controls are also present at the bottom of the `gi3d.SceneView` used here, which provides a basic gui around the 3D Scene.

## Selecting and Manipulating

There is a selector at the bottom of the SceneView where you can change the selection behavior.  It defaults to `NotSelectable`, but if you change it to `SelectionBox` or `Manipulable`, then you can click on different objects (and parts of objects) and see a selection box or manipulation box around them.

Selected objects can be edited using the `Edit` button, and the overall `Scene` can be edited using the `Edit Scene` button.

In `Manipulable` mode, you can change the Pose of objects by dragging on the different control spheres, using the following keyboard modifiers:
* none = Move -- moves in dominant plane -- orbit camera to enable moving in other planes.
* `Ctrl` = scale -- scales along the relevant dimensions.
* `Alt` = rotate -- rotates in current "depth" plane (again move camera to rotate in other planes).

The code for all of this is in `gi3d/manip.go` -- it is relatively straightforward, leveraging the GoGi event system, based on bounding boxes for directing events, and having builtin support for dragging etc.  The `ManipPt` manipulation points receive mouse events directly and translate movements into the respective transformations.

You can generate Go code for the Pose produced by manipulating, by clicking on the Pose field in the Edit dialog of a Node, and then clicking on the `Go Code` button in the toolbar.

## Inspect and Edit the Scene

You can also use the standard GoGi `Ctrl+Alt+I` shortcut to invoke the `GoGi Editor` and you can click on the `scene` and other elements of the scenegraph and edit / inspect them.  Toolbar actions have been enabled on everything to call useful methods, so you can pretty much configure the entire scene dynamically on the fly.

For example, you can click on `Meshes`, and edit the parameters of any of the various mesh objects, cutting the sphere into a sliver, experimenting with different numbers of segments which determines how smooth the curved shapes are, etc.

Changing colors in the materials, and turning off `Mat.CullBack` are also useful and informative, especially in combination with various mesh manipulations that result in partial shape rendering (e.g., sphere slivers, the cylinder without a top or bottom, etc) -- being able to see the back sides of those shapes makes them look less strange.

## Scenegraph Structure

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
   

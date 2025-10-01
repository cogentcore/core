+++
Name = "XYZ"
Categories = ["Architecture"]
+++

**xyz** is a package that supports interactive 3D viewing and editing, building on the [[GPU]] framework, which in turn is based on [WebGPU](https://www.w3.org/TR/webgpu/), which is available on all non-web platforms and is gaining wider browser support: [WebGPU](https://caniuse.com/webgpu). The full API documentation is at [[doc:xyz]], and the [xyz example](https://github.com/cogentcore/core/tree/main/examples/xyz) provides a good working example to start from.

Here is a screenshot of the xyz example:

![Screenshot of the xyz example](media/xyz.jpg)

## Elements

### Solid

The [[doc:xyz.Solid]] type is the main 3D object that renders a visible shape. Its shape comes from a [[doc:xyz.Mesh]], which is a potentially shared element stored in a library and accessed by unique names. The other visible properties (e.g., `Color`, how `Shiny` or `Reflective` it is, etc) are defined by the [[doc:xyz.Material]] properties. It can also have a [[doc:xyz.Texture]] (also referenced by unique name from a shared library) that is an image providing a richer more realistic appearance.

The [[doc:xyz.Pose]] values on a Solid (and all other `xyz` nodes) specify the 3D position and orientation of the object.

### Scene

The 3D scenegraph and the shared `Mesh` and `Texture` libraries are managed by the [[doc:xyz.Scene]] node, along with the `Lights`, `Camera` and a `Library` of loaded 3D objects that can be referenced by name. The scene of objects is rooted in the `Children` of the Scene.

`Mesh` and `Texture` elements use the "Set / Reset" approach, where `Set` does add or update, based on unique name id, and if there are large changes and unused elements, a `Reset` can be used to start over. After the GPU is up and running (e.g., after the main app window is opened), changes take effect immediately, but everything can be configured prior to that, and they will all be applied when the GPU is activated.

### Group

The [[doc:xyz.Group]] is a container of other elements, and typically visible objects are actually Groups with multiple different subgroups and finally Solid elements. Groups are also used for optimizing the rendering, such that any Group that is fully out of view will be skipped over entirely: therefore, it is a good idea to create spatially-organized groups at different scales, depending on the overall scene size and complexity.

### Resources

Meshes are _exclusively_ defined by indexed triangles, and there are standard shapes such as `Box`, `Sphere`, `Cylinder`, `Capsule`, and `Lines` (rendered as thin boxes with end points specified), e.g.,:

```go
	sphere := xyz.NewSphere(sc, "sphere", .75, 32)
	tree.AddChild(sc, func(n *xyz.Solid) {
		n.SetMesh(sphere).SetColor(colors.Orange).SetPos(0, -2, 0)
	})
```

`Texture`s are loaded from Go image files, e.g.,:

```go
    grtx := xyz.NewTextureFileFS(assets.Content, sc, "ground", "ground.png")
	floorp := xyz.NewPlane(sc, "floor-plane", 100, 100)
	tree.AddChild(sc, func(n *xyz.Solid) {
		n.SetMesh(floorp).SetTexture(grtx).SetPos(0, -5, 0)
		n.Material.Tiling.Repeat.Set(40, 40)
    })
```

The Scene also contains a `Library` of uniquely named "objects" (Groups) which can be loaded from 3D object files in the [Wavefront .obj format](https://en.wikipedia.org/wiki/Wavefront_.obj_file), and then added into the scenegraph as needed, e.g.:

```go
	lgo := errors.Log1(sc.OpenToLibraryFS(assets.Content, "gopher.obj", ""))
	tree.Add(p, func(n *xyz.Object) {
		n.SetObjectName("gopher").SetScale(.5, .5, .5).SetPos(1.4, -2.5, 0)
		n.SetAxisRotation(0, 1, 0, -60)
	})
```

The library objects are Cloned into the scenegraph so they can be independently configured and manipulated there. Because the Group and Solid nodes are lightweight, this is all very efficient.

## Lights

At least one light must be added to make everything in the scene visible. Four different types are supported.

The lights use [standard light types](http://planetpixelemporium.com/tutorialpages/light.html) via the [[doc:xyz.LightColors]] to specify the light color, e.g., `xyz.DirectSun` is the brightest pure white color.

The [[doc:xyz.Ambient]] light is the simplest, providing a diffuse uniform light that doesn't come from any given direction:

```go
	xyz.NewAmbient(sc, "ambient", 0.3, xyz.DirectSun)
```

The [[doc:xyz.Directional]] light shines toward the origin (position 0,0,0) from wherever it is placed, and only the vector from this position to the origin matters: distance is not a factor:

```go
	xyz.NewDirectional(sc, "directional", 1, xyz.DirectSun).Pos.Set(0, 2, 1)
```

The [[doc:xyz.Point]] light is like `Directional` except that it has decay factors as a function of distance:

```go
	xyz.NewPoint(sc, "point", 1, xyz.DirectSun).Pos.Set(-5, 0, 2)
```

The [[doc:xyz.Spot]] light is the most complex, with an angular decay and a angular cutoff, light a classic desk lamp.

```go
	xyz.NewSpot(sc, "spot", 1, xyz.DirectSun).Pos.Set(-5, 0, 2)
```

If you want a visible representation of the light in the Scene, you can add that using whatever Solid you want.

## Camera

The [[doc:xyz.Camera]] determines what view into the 3D scene is rendered, via its Pose parameters, e.g.,:

```go
	sc.Camera.Pose.Pos.Set(0, 2, 10)
	sc.Camera.LookAt(math32.Vector3{}, math32.Vec3(0, 1, 0)) // look at origin
```

## xyzcore

The [[doc:xyz/xyzcore]] package provides two `core.Widget` wrappers around the `xyz.Scene`:

* [[doc:xyz/xyzcore.SceneEditor]] provides full object selection and manipulation functionality, and a toolbar of controls. 

* [[doc:xyz/xyzcore.Scene]] just displays the scene, and supports mouse-based zooming and panning of the camera.

### Events

The GUI interactions are managed by functions such as [[doc:xyz.Scene.MouseScrollEvent]], [[doc:xyz.Scene.SlideMoveEvent]], and [[doc:xyz.Scene.NavKeyEvent]], which are connected to the core event management system in `xyzcore.Scene`.

There are also other helper functions in `xyz/events.go`.

## Updating, Making, Rendering, etc

xyz is based on the [[doc:tree]] infrastructure, documented in [[Plan]], for flexible ways of building and updating 3D scenes.

The `Update()` method on `Scene` (or any node) will update everything based on Maker and Updater functions that have been installed.

The `Render()` method on the Scene will render to a [[doc:gpu.RenderTexture]] that is an offscreen rendering target. Use `RenderGrabImage()` to get the resulting image as a Go image. The `xyzcore.Scene` automatically manages the updating and rendering consistent with the [[core]] standard mechanisms, using an optimized direct rendering logic so the resulting rendered image stays on the GPU and is used directly.

## Lines, arrows, etc

There are handy functions in `xyz/lines.go` for creating complex line-based shapes, including those with arrow heads. These include `Mesh` functions for making the meshes, and `Init` functions that initialize a `Group` or `Solid` with line shapes.

To make a set of lines:

```go
	lines := NewLines(sc, "Lines", []math32.Vector3{{-3, -1, 0}, {-2, 1, 0}, {2, 1, 0}, {3, -1, 0}}, math32.Vec2(.2, .1), CloseLines)
	tree.AddChild(sc, func(n *Solid) {
		n.SetMesh(lines).SetColor(color.RGBA{255, 255, 0, 128}).SetPos(0, 0, 1)
	})
```

To make a line with arrow heads:

```go
	tree.AddChild(sc, func(g *xyz.Group) {
		InitArrow(g, math32.Vec3(-1.5, -.5, .5), math32.Vec3(2, 1, 1), .05, colors.Cyan, StartArrow, EndArrow, 4, .5, 8)
	})
```


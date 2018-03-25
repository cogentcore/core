# gi

GoGi is part of the GoKi Go language (golang) full strength tree structure system (ki = tree in Japanese)

`package gi` -- scenegraph based 2D and 3D GUI / graphics interface (Gi) in Go

GoDoc documentation: https://godoc.org/github.com/rcoreilly/goki/gi

Building: cd to OS-specific dir (`cocoa` for mac, `win` for windows, etc) and type "make" to build the core OS-specific GUI C interface for windows and events -- speeds up subsequent `go build` steps significantly.

# Code Map

* `node*.go` -- `NodeBase`, `Node2DBase`, `3D` structs and interfaces -- all Gi nodes are of this type
* `geom2d.go` -- All 2D geometry: Point2D, Size2D, etc
* `paint.go` -- `Paint` struct that does all the direct rendering, based on `gg`
	+ `stroke.go`, `fill.go` -- `StrokeStyle` and `FillStyle` structs for stroke, fill settings
* `style.go` -- `Style` and associated structs for CSS-based `Widget` styling
* `viewport2d.go` -- `Viewport2D` that has an `Image.RGBA` that `Paint` renders onto
* `window.go` -- `Window` is the top-level window that uses (our own version of) `go.wde` to open a gui window and send events to nodes
* `shapes2d.go` -- All the basic 2D SVG-based shapes: `Rect`, `Circle` etc
* `font.go`, `text.go` -- `FontStyle`, `TextStyle`, `Text2D` node
* `path.go` -- TBD: path-rendering nodes

# Design notes

* Excellent rendering code on top of freetype rasterizer, does almost everything we need: https://github.com/fogleman/gg -- borrowed heavily from that!

* Also incorporated this framework for getting windows and events: https://github.com/skelterjohn/go.wde

The 2D Gi is based entirely on the SVG2 spec: https://www.w3.org/TR/SVG2/Overview.html, and renders directly to an Image struct (`Viewport2D`)

The 3D Gi is based on TBD (will be impl later) and renders directly into a `Viewport3D` offscreen image buffer (OpenGL for now, but with generalization to Vulkan etc).

Any number of such (nested or otherwise) Viewport nodes can be created and they are all then composted into the final underlying bitmap of the Window.

Within a given rendering parent (Viewport2D or Viewport3D), only nodes of the appropriate type (`GiNode2D` or `GiNode3D`) may be used -- each has a pointer to their immediate parent viewport (indirectly through a ViewBox in 2D)

There are nodes to embed a Viewport2D bitmap within a Viewport3D scene, and vice-versa.  For a 2D viewport in a 3D scene, it acts like a texture and can be mapped onto a plane or anything else.  For a 3D viewport in a 2D scene, it just composts into the bitmap directly.

The overall parent Window can either provide a 2D or 3D viewport, which map directly into the underlying pixels of the window, and provide the "native" mode of the window, for efficiency.

## 2D Design

* There are two main types of 2D nodes, which can be intermingled, but generally are segregated:
	+ SVG rendering nodes that directly set properties on the Paint object and typically have their own geometry etc -- generally not put within a Layout etc -- convenient to put in an SVGBox or SVGViewport -- their geom units are determined entirely by the transforms etc and we do not support any further unit specification -- just raw float64 values
	+ Widget nodes that use the full CSS-based styling (e.g., the Box model etc), are typically placed within a Layout

* Using the basic 64bit geom from fogleman/gg -- the `github.com/go-gl/mathgl/mgl32/` math elements (vectors, matricies) which build on the basic `golang.org/x/image/math/f32` did not have appropriate 2D rendering transforms etc.

* Using 64bit floats for coordinates etc because the spec says you need those for the "high quality" transforms, and Go defaults to them, and it just makes life easier -- won't have so crazy many coords in 2D space as we might have in 3D, where 32bit makes more sense and optimizes GPU hardware.
	+ shiny uses highly optimized rendering with either 32bit floats or ints -- later could look into it

* The SVG default coordinate system has 0,0 at the upper-left.  The default 3D coordinate system flips the Y axis so 0,0 is at the lower left effectively (actually it uses center-based coordinates so 0,0 is in the center of the image, effectively -- everything is defined by the camera anyway)

* Widget-based layout is simple x,y offsets, and All 2D nodes obey that -- typically you want to add an SVGBox or SVGViewport node to encapsulate pure SVG-based rendering within an overall simple x,y framework

* Basic CSS styling is based on the Box model: https://www.w3schools.com/css/css_boxmodel.asp -- see also the Box shadow model https://www.w3schools.com/css/css3_shadows.asp -- general html spec: https://www.w3.org/TR/html5/index.html#contents -- better ref section of w3schools for css spec: https://www.w3schools.com/cssref/default.asp

* Every non-terminal Widget must either be a Layout or take full responsibility for everything under it -- i.e., all arbitrary collections of widgets must be Layouts -- only the layout has all the logic necessary for organizing the geometry of its children.  There is only one Layout type that supports all forms of Layout -- and it is a proper Widget -- not a side class like in Qt Widgets.  The Frame is a type of Layout that draws a frame around itself.

### TODO

* Rendering with a clip mask in place is DRAMATICALLY slower.  Need to use a diff solution for keeping stuff from rendering outside the box. 

In paintserver.go, this seems to be it:

			ma := s.Alpha
			if r.mask != nil {
				ma = ma * uint32(r.mask.AlphaAt(x, y).A) / 255
				if ma == 0 {
					continue
				}
			}

really not clear how that ends up being so slow.  another soln is to just add extra bounds.


* need disconnection code in ki 

* fix scrolling issues per below, and look into scroll gestures, scrollwheel, etc.
* tree view should work quite well -- put in a layout and test out..

Next:
* why do we have units context on style and paint?  one is not getting initialized -- 
doesn't make sense to have two.
* check for Updating count > 0 somewhere -- going to be a common error
* Layout flow types
* Layout grid
* WidgetBase-- has Controls = Layout -- add to NodeWidget. -- Qt calls them "subcontrols" -- e.g., http://doc.qt.io/qt-5/stylesheet-examples.html#customizing-qspinbox

* double-click!

* style parsing crash on font-family

* color generates linear interpolations, lighter, darker -- then add a painter guy based on that to generate gradients, and then we're in the shadow business, etc -- key innovation over css: relative color transforms: lighterX darkerX that transform existing color -- great for styling widgets etc.

Soon:

* Reminder: grep all todo: in code -- lots!
* svg box, viewport
* keyboard shortcuts -- need to register with window / event manager on a signal list..

* Missing Widgets, in rough order of importance / ease -- see http://doc.qt.io/qt-5/qtquickcontrols2-differences.html for ref
	+ SplitView
	+ Menu / MenuBar / MenuItem
	+ ComboBox
	+ SpinBox
	+ RadioButton, CheckBox
	+ Dialog -- either overlay or additional window -- platform dependent
	+ Toolbar / ToolButton / Action
	+ ProgressBar  (based on slider?)
	+ ToolTip
	+ TextArea

### TO-DONE (ish)

* Widgets
	+ Button -- needs alt styling through children?
	+ Slider -- pretty done
	+ TextField -- needs selection / clipboard, constraints
	+ TreeView (NodeWidget) -- needs controls, menu, updating, dnd, clip, -- see about LI, UL lists..
	+ TabWidget -- needs updating
	+ Label -- done -- could make lots of H1, etc alts
	+ ScrollBar -- ScrollArea must just be a layout, as Layout is already in the right position to know about all of its children's sizes, and to control the display thereof -- it just changes the child positions based on scroll position, and uses WinBBox to exclude rendering of any objects fully outside of view, and clipping for those partially within view -- very efficient!  Except clipping seems a bit slow.

* update increment threshold for scrollbar -- less frequent updates.


## 3D Design

* keep all the elements separate: geometry, material, transform, etc.  Including shader programs.  Maximum combinatorial flexibility.  not clear if Qt3D really obeys this principle, but Inventor does, and probably other systems do to.


# Links

## SVG

* SVG *text* generator in go: https://github.com/ajstarks/svgo
* cairo wrapper in go: https://github.com/ungerik/go-cairo -- maybe needed for PDF generation from SVG?
* https://github.com/jyotiska/go-webcolors -- need this for parsing colors
* highly optimized vector rasterizer -- not clear about full scope but could potentially impl that later https://godoc.org/golang.org/x/image/vector

## GUI

* Shiny (not much progress recently, only works on android?):  https://github.com/golang/go/issues/11818 https://github.com/golang/exp/tree/master/shiny
* Current plans for GUI based on OpenGL: https://docs.google.com/document/d/1mXev7TyEnvM4t33lnqoji-x7EqGByzh4RpE4OqEZck4
* Window events: https://github.com/skelterjohn/go.wde
* Mobile: https://github.com/golang/mobile/  https://github.com/golang/go/wiki/Mobile

### Material design

* https://github.com/dskinner/material -- uses simplex layout -- seems like a complicated area: https://arxiv.org/pdf/1401.1031.pdf

* https://doc.qt.io/qt-5.10/qtquickcontrols2-material.html


## Go graphics

* https://golang.org/pkg/image/
* https://godoc.org/golang.org/x/image/vector
* https://godoc.org/github.com/golang/freetype/raster
* https://github.com/fogleman/gg -- key lib using above -- 2D rendering!

## 3D

* https://github.com/g3n/engine
* https://github.com/oakmound/oak
* https://github.com/walesey/go-engine

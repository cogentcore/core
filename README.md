# gi

GoGi is part of the GoKi Go language (golang) full strength tree structure system (ki = tree in Japanese)

`package gi` -- scenegraph based 2D and 3D GUI / graphics interface (Gi) in Go

[![Go Report Card](https://goreportcard.com/badge/github.com/rcoreilly/goki/gi)](https://goreportcard.com/report/github.com/rcoreilly/goki/gi)
[![GoDoc](https://godoc.org/github.com/rcoreilly/goki/gi?status.svg)](http://godoc.org/github.com/rcoreilly/goki/gi)

Building: cd to OS-specific dir (`cocoa` for mac, `win` for windows, etc) and type "make" to build the core OS-specific GUI C interface for windows and events -- speeds up subsequent `go build` steps significantly.

# Code Map

* `examples/widgets` -- main example widget gallery -- `go build ...` in there to give it a try
* `node*.go` -- `NodeBase`, `Node2DBase`, `3D` structs and interfaces -- all Gi nodes are of this type
* `geom2d.go` -- `Vec2D` is main geom type used for 2D, plus transform matrix
* `paint.go` -- `Paint` struct that does all the direct rendering, based on `gg`
	+ `stroke.go`, `fill.go` -- `StrokeStyle` and `FillStyle` structs for stroke, fill settings
* `style.go` -- `Style` and associated structs for CSS-based `Widget` styling
* `viewport2d.go` -- `Viewport2D` that has an `Image.RGBA` that `Paint` renders onto
* `window.go` -- `Window` is the top-level window that uses (our own version of) `go.wde` to open a gui window and send events to nodes
* `shapes2d.go` -- All the basic 2D SVG-based shapes: `Rect`, `Circle` etc
* `font.go`, `text.go` -- `FontStyle`, `TextStyle`, `Text2D` node
* `layout.go` -- main `Layout` object with various ways of arranging widget elements
* `widget.go` -- `WidgetBase` for all widgets
* `buttons.go` -- `ButtonBase`, `Button` and other basic command button types
* `sliders.go` -- `SliderBase`, `Slider`, `ScrollBar`
* `action.go` -- `Action` is a Button-type used in menus and toolbars, with a simplified `ActionTriggered` signal
* `textwidgets.go` -- `Label`, `TextField`, `ComboBox` -- also defines the `gi.Labeler` interface and `ToLabel` converter method (which falls back on kit.ToString using Stringer), which is used for generating a gui-appropriate label for an object -- e.g., for reflect.Type it just presents the raw type name without prefix.
* `*view.go` -- `TreeView` widget shows a graphical view of a tree, `TabView` widget for tabbed panels.  Todo: `StructView` for editing structs
* `oswin` is a modified version of the back-end OS-specific code from Shiny: https://github.com/golang/exp/tree/master/shiny -- originally used https://github.com/skelterjohn/go.wde but shiny is much faster for updating the window because it is gl-based, and doesn't have any other dependencies (removed dependencies on mobile, changed the event structure to better fit needs here).

# Design notes

The 2D Gi is based entirely on the SVG2 spec: https://www.w3.org/TR/SVG2/Overview.html, and renders directly to an Image struct (`Viewport2D`)

The 3D Gi is based on TBD (will be impl later) and renders directly into a `Viewport3D` offscreen image buffer (OpenGL for now, but with generalization to Vulkan etc).

Any number of such (nested or otherwise) Viewport nodes can be created and they are all then composted into the final underlying bitmap of the Window.

Within a given rendering parent (Viewport2D or Viewport3D), only nodes of the appropriate type (`GiNode2D` or `GiNode3D`) may be used -- each has a pointer to their immediate parent viewport (indirectly through a ViewBox in 2D)

There are nodes to embed a Viewport2D bitmap within a Viewport3D scene, and vice-versa.  For a 2D viewport in a 3D scene, it acts like a texture and can be mapped onto a plane or anything else.  For a 3D viewport in a 2D scene, it just composts into the bitmap directly.

The overall parent Window can either provide a 2D or 3D viewport, which map directly into the underlying pixels of the window, and provide the "native" mode of the window, for efficiency.

## 2D Design

* There are three main types of 2D nodes:
	+ `Viewport2D` nodes that manage their own `oswin.Image` bitmap and can upload that directly to the `oswin.Texture` that then uploads directly to the `oswin.Window`.  The parent `Window` has a master `Viewport2D` that backs the entire window, and is what most `Widget`'s render into.
		+ Popup `Dialog` and `Menu`'s have their own viewports that are layered on top of the main window viewport.
		+ `SVG` and its subclass `Icon` are containers for SVG-rendering nodes.
	+ `Widget` nodes that use the full CSS-based styling (e.g., the Box model etc), are typically placed within a `Layout` -- they use `units` system with arbitrary DPI to transform sizes into actual rendered `dots` (term for actual raw resolution-dependent pixels -- "pixel" has been effectively co-opted as a 96dpi display-independent unit at this point).  Widgets have non-overlapping bounding boxes (`BBox`).
	+ `SVG` rendering nodes that directly set properties on the `Paint` object and typically have their own geometry etc -- they should be within a parent `SVG` viewport, and their geom units are determined entirely by the transforms etc and we do not support any further unit specification -- just raw float values.

* Rendering: there are 2 major render frameworks:
	+ https://godoc.org/github.com/golang/freetype/raster
	+ https://godoc.org/golang.org/x/image/vector
	+ This code: https://github.com/fogleman/gg uses freetype and handles the majority of SVG.  Freetype has a `Painter` interface that is key for supporting the more flexible types of patterns, images, etc that can be used for the final render step.  It also directly supports line joins (round, bevel) and caps: square, butt, round.  It uses fixed.Int26_6 values.  The `image/vector` code is highly optimized based on this rust-based rasterizer: https://medium.com/@raphlinus/inside-the-fastest-font-renderer-in-the-world-75ae5270c445 and uses SIMD instructions.  It switches between float32 and fixed.Int22_10 values depending on size.  Presumably the optimal case would be a merge of these different technologies for the best-of-all but I'm not sure how the Painter flexibility could be incorporated.  Also, the freetype system is already supported for fonts -- would need to integrate that.  This is clearly a job for nigeltao.. :)
	+ Converted the gg system to float32 instead of 64, using the `geom.go Vec2D` core element.  Note that the `github.com/go-gl/mathgl/mgl32/` math elements (vectors, matricies) which build on the basic `golang.org/x/image/math/f32` do not have appropriate 2D rendering transforms etc.

* The SVG and most 2D default coordinate systems have 0,0 at the upper-left.  The default 3D coordinate system flips the Y axis so 0,0 is at the lower left effectively (actually it uses center-based coordinates so 0,0 is in the center of the image, effectively -- everything is defined by the camera anyway)

* Basic CSS styling is based on the Box model: https://www.w3schools.com/css/css_boxmodel.asp -- see also the Box shadow model https://www.w3schools.com/css/css3_shadows.asp -- general html spec: https://www.w3.org/TR/html5/index.html#contents -- better ref section of w3schools for css spec: https://www.w3schools.com/cssref/default.asp

* Naming conventions for scenegraph / html / css objects: it seems conventional in HTML to use lowercase with hyphen separator for id naming.  And states such as :hover :active etc: https://stackoverflow.com/questions/1696864/naming-class-and-id-html-attributes-dashes-vs-underlines https://stackoverflow.com/questions/70579/what-are-valid-values-for-the-id-attribute-in-html -- so we use that convention, which then provides a clear contrast to the UpperCase Go code (see ki/README.md for Go conventions).  Specific defined selectors: https://www.w3schools.com/cssref/css_selectors.asp -- note that read-only is used

* Every non-terminal Widget must either be a Layout or take full responsibility for everything under it -- i.e., all arbitrary collections of widgets must be Layouts -- only the layout has all the logic necessary for organizing the geometry of its children.  There is only one Layout type that supports all forms of Layout -- and it is a proper Widget -- not a side class like in Qt Widgets.  The Frame is a type of Layout that draws a frame around itself.

* General Widget method conventions:
	+ SetValue kinds of methods are wrapped in updates, but do NOT emit a signal
	+ SetValueAction calls SetValue and emits the signal
	+ this allows other users of the widget that also recv the signal to not trigger themselves, but typically you want the update, so it makes sense to have that in the basic version.  ValueView in particular requires this kind of behavior.  todo: go back and make this more consistent.

### TODO

* TEST: CSS on everything .class #name -- just need to call
  additional style passes with whatever matches..

* Style one field -- use for fields in Node objs -- e.g., for indent in TreeView
* also thumb in slider -- make it em sized -- will fix weird sizing issues -- generalize "StyleFields" struct -- just needs an Init interface.

* Nonmodal dialogs -> windows

* highlight, lowlight versions of lighter-darker that are relative to current
  lightness for dark-style themes.

* override ki.Props json to save type names

* get all json save / load working

* Button parts not updating when I add a label, even with full re-render.

* Auto-apply when valueview TextField is destroyed?

* scroll wheel events for scrolling! -- frame intercepts.  also sliders when in focus?

* color generates linear interpolations, lighter, darker -- then add a painter guy based on that to generate gradients, and then we're in the shadow business, etc 

* saving non-string properties not working -- doesn't know the type for
  loading.. converts to a map.

* double-click interval not working at all -- calling everything double-click

* arg view / dialog and button tags

* DND for slices, trees: need the restore under vp, draw vp sequence to work right -- maybe after new rendering.

* consolidate dialog popup code between menu and dialog

* fix issue with tiny window and dialog not scrolling and blocking interface

* Structview: condshow / edit
	
* test SVG path rendering 
* property-based xforms for svg

* native UnmarshalXML is not going to be flexible enough to support effective
  parsing from SVG into corresponding nodes -- going to have to use pi parsing system.. 
  
* which means finishing graphical elements using simple hand-coded Icon's instead of parsing from file, until the tree view and property editor are usable, to then make the parsing workable.

* first pass of parser retains a full static []byte string and creates pointers into it as indicies -- don't have to duplicate all that -- actually the go slice system does this sharing already so not a big deal..

Next:
* Layout flow types

* style parsing crash on font-family

* all widgets need read-only and disabled states

Soon:

* Reminder: grep all todo: in code -- lots!
* keyboard shortcuts -- need to register with window / event manager on a signal list..

* all builtin defaults should use units.Value and other raw values instead of strings

* maybe not: need general Node2D Apply, Revert methods, for e.g., text fields when the user presses Ok

* Missing Widgets, in rough order of importance / ease -- see http://doc.qt.io/qt-5/qtquickcontrols2-differences.html for ref
	+ sub-menus -- should just work??
	+ RadioButton -- checkbox + mutex logic -- everyone within same parent is mutex -- easy
	+ Toolbar / ToolButton -- just a layout really, with some styling?
	+ ProgressBar -- very simple
	+ ToolTip
	+ TextArea

### TO-DONE (ish)

* Widgets
	+ Menu / MenuBar / MenuItem -- needs sub-menu support
	+ Button -- needs alt styling through children?
	+ Slider -- pretty done
	+ TextField -- needs selection / clipboard, constraints
	+ TreeView (NodeWidget) -- needs controls, menu, updating, dnd, clip, -- see about LI, UL lists..
	+ TabWidget -- needs updating
	+ Label -- done -- could make lots of H1, etc alts
	+ ScrollBar -- ScrollArea must just be a layout, as Layout is already in the right position to know about all of its children's sizes, and to control the display thereof -- it just changes the child positions based on scroll position, and uses WinBBox to exclude rendering of any objects fully outside of view, and clipping for those partially within view -- very efficient! 
	+ ComboBox
	+ Dialog -- either overlay or additional window -- platform dependent
	+ CheckBox
	+ SpinBox
	+ SplitView -- almost..

* not needed now: update increment threshold for scrollbar -- less frequent updates.


## 3D Design

* keep all the elements separate: geometry, material, transform, etc.  Including shader programs.  Maximum combinatorial flexibility.  not clear if Qt3D really obeys this principle, but Inventor does, and probably other systems do to.


# Links

## SVG

* SVG *text* generator in go: https://github.com/ajstarks/svgo
* cairo wrapper in go: https://github.com/ungerik/go-cairo -- maybe needed for PDF generation from SVG?
* https://github.com/jyotiska/go-webcolors -- need this for parsing colors
* highly optimized vector rasterizer -- not clear about full scope but could potentially impl that later https://godoc.org/golang.org/x/image/vector

## GUI

* qt quick controls https://doc.qt.io/qt-5.10/qtquickcontrols2-differences.html
* Shiny https://github.com/golang/go/issues/11818 https://github.com/golang/exp/tree/master/shiny
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

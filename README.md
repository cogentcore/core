![alt tag](logo/gogi_logo.png)

GoGi is part of the GoKi Go language (golang) full strength tree structure system (ki = 木 = tree in Japanese)

`package gi` is a scenegraph-based 2D and 3D GUI / graphics interface (Gi) in Go.

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/gi)](https://goreportcard.com/report/github.com/goki/gi)
[![GoDoc](https://godoc.org/github.com/goki/gi?status.svg)](https://godoc.org/github.com/goki/gi)

NOTE: Requires Go version `1.10+` due to use of `math.Round`.

See the [Wiki](https://github.com/goki/gi/wiki) for more docs,   [Install](https://github.com/goki/gi/wiki/Install) instructions (just standard `go get ...`), and [Google Groups goki-gi](https://groups.google.com/forum/#!forum/goki-gi) emailing list.

GoGi uses the [GoKi](https://github.com/goki/ki) tree infrastructure to implement a simple, elegant GUI framework in full native idiomatic Go (with minimal OS-specific backend interfaces based on the Shiny drivers, supporting MacOS, Linux, and Windows).  The overall design is an attempt to integrate existing standards and conventions from widely-used frameworks, including Qt (overall widget design), HTML / CSS (styling), and SVG (rendering).  This 2D framework also integrates with a (in progress) 3D scenegraph, supporting interesting combinations of these frameworks.  Currently GoGi is focused on desktop systems, but nothing prevents adaptation to mobile. 

See [Gide](https://github.com/goki/gide) for a complete, complex application written in GoGi (an IDE), and the various examples in this repository for lots of useful demonstrations -- start with the  [Widgets](https://github.com/goki/gi/tree/master/examples/widgets) example which has a bit of a tutorial introduction.

# Main Features

* Has all the standard widgets: `Button`, `Menu`, `Slider`, `TextField`, `SpinBox`, `ComboBox` etc, with tooltips, hover, focus, copy / paste (full native clipboard support), drag-n-drop -- the full set of standard GUI functionality.  See `gi/examples/widgets` for a demo of all the widgets.

* Powerful `Layout` logic auto-sizes everything -- very easy to configure interfaces that just work across different scales, resolutions, platforms.  Automatically remembers and reinstates window positions and sizes across sessions.

* CSS-based styling allows easy customization of everything -- native style properties are fully HTML compatible (with all standard `em`, `px`, `pct` etc units), including full HTML "rich text" styling for all text rendering (e.g., in `Label` widget) -- can decorate any text with inline tags (`<strong>`, `<em>` etc), and even include links.

* Compiles in seconds, compared to many minutes to hours for comparable alternatives such as Qt, and with  minimal cgo dependency.  As of April 2019 we now depend on the [glfw](https://github.com/go-gl/glfw) cross-platform GUI infrastructure system, and the [go-gl/gl](https://github.com/go-gl/gl) OpenGL bindings, to support the 3D (`gi3d`) aspect of the framework.

* Fully self-contained -- does *not* use OS-specific native widgets -- results in simple, elegant, consistent code across platforms, and is fully `HiDPI` capable and scalable using standard `Ctrl/Cmd+Plus or Minus` key, and in `Preferences`.

* `SVG` element (in `svg` sub-package) supports full SVG rendering -- used for Icons internally and available for advanced graphics displays -- see `gi/examples/svg` for viewer and start on editor, along with a number of test .svg files.

* Advanced **Model / View** paradigm with `reflect`ion-based view elements that display and manipulate all the standard Go types (in `giv` sub-package), from individual types (e.g., int, float display in a `SpinBox`, "enum" const int types in a `ComboBox` chooser) to composite data structures, including `StructView` editor of `struct` fields, `MapView` and `SliceView` displays of `map` and `slice` elements (including full editing / adding / deleting of elements), and full-featured `TableView` for a `slice`-of-`struct` and `TreeView` for GoKi trees.
    + `TreeView` enables a built-in GUI editor / inspector for designing gui elements themselves.  Just press `Control+Alt+I` in any window to pull up this editor / inspector.  Scene graphs can be automatically saved / loaded from JSON files, to provide a basic GUI designer framework -- just load and add appropriate connections..

![Screenshot of Widgets demo](screenshot.png?raw=true "Screenshot of Widgets demo")

![Screenshot of Widgets demo, Darker](screenshot_dark.png?raw=true "Screenshot of Widgets demo, Darker Colors")

# Code Overview

There are three main types of 2D nodes:

* `Viewport2D` nodes that manage their own `oswin.Image` bitmap and can upload that directly to the `oswin.Texture` that then uploads directly to the `oswin.Window`.  The parent `Window` has a master `Viewport2D` that backs the entire window, and is what most `Widget`'s render into.
    + Popup `Dialog` and `Menu`'s have their own viewports that are layered on top of the main window viewport.
    + `SVG` and its subclass `Icon` are containers for SVG-rendering nodes.

* `Widget` nodes that use the full CSS-based styling (e.g., the Box model etc), are typically placed within a `Layout` -- they use `units` system with arbitrary DPI to transform sizes into actual rendered `dots` (term for actual raw resolution-dependent pixels -- "pixel" has been effectively co-opted as a 96dpi display-independent unit at this point).  Widgets have non-overlapping bounding boxes (`BBox` -- cached for all relevant reference frames).

* `SVG` rendering nodes that directly set properties on the `Paint` object and typically have their own geometry etc -- they should be within a parent `SVG` viewport, and their geom units are determined entirely by the transforms etc and we do not support any further unit specification -- just raw float values.

General Widget method conventions:
* `SetValue` kinds of methods are wrapped in `UpdateStart` / `End`, but do NOT emit a signal
* `SetValueAction` calls `SetValue` and emits the signal
This allows other users of the widget that also recv the signal to not trigger themselves, but typically you want the update, so it makes sense to have that in the basic version.  `ValueView` in particular requires this kind of behavior.

The best way to see how the system works are in the `examples` directory, and by interactively modifying any existing gui using the interactive reflective editor via `Control+Alt+I`.

# Status

As of 4/22/2019 there will be a period of relative instability during the development of the 3D `gi3d` component of the framework, and switch to using `glfw`.  Active users should join [Google Groups goki-gi](https://groups.google.com/forum/#!forum/goki-gi) emailing list to receive more detailed status updates.

For the 2D framework:

* All major functionality is in place, and API is stable and only minor changes are anticipated going forward.  The system is now ready for wider adoption.

* Everything has been thoroughly tested, but generally only by a small number of core developers / users.

* Please file [Issues](https://github.com/goki/gi/issues) for anything that does not work.

* 3/2019: `python` wrapper is now available!  you can do most of GoGi from python now.  See [README.md](https://github.com/goki/gi/tree/master/python/README.md) file there for more details.


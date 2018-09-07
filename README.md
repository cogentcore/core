# gi

![alt tag](logo/gogi_logo.png)

GoGi is part of the GoKi Go language (golang) full strength tree structure system (ki = tree in Japanese)

`package gi` is a scenegraph-based 2D and 3D GUI / graphics interface (Gi) in Go

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/gi)](https://goreportcard.com/report/github.com/goki/gi)
[![GoDoc](https://godoc.org/github.com/goki/gi?status.svg)](http://godoc.org/github.com/goki/gi)

NOTE: Requires Go version `1.10+` due to use of `math.Round`.

See the [Wiki](https://github.com/goki/gi/wiki) for more docs, discussion, etc.

GoGi uses the GoKi tree infrastructure to implement a simple, elegant GUI framework in full native idiomatic Go (with minimal OS-specific backend interfaces based on the Shiny drivers).  The overall design is an attempt to integrate existing standards and conventions from widely-used frameworks, including Qt (overall widget design), HTML / CSS (styling), and SVG (rendering).  This 2D framework also integrates with a (planned) 3D scenegraph, supporting interesting combinations of these frameworks.  Currently GoGi is focused on desktop systems, but nothing prevents adaptation to mobile.

# Main Features

* Has all the standard widgets: `Button`, `Menu`, `Slider`, `TextField`, `SpinBox`, `ComboBox` etc, with tooltips, hover, focus, copy / paste (full native clipboard support), drag-n-drop -- the full set of standard GUI functionality.  See `gi/examples/widgets` for a demo.

* Powerful `Layout` logic auto-sizes everything -- very easy to configure interfaces that just work across different scales, resolutions, platforms.  Automatically remembers and reinstates window positions and sizes across sessions.

* CSS-based styling allows easy customization of everything -- native style properties are fully HTML compatible (with all standard `em`, `px`, `pct` etc units), including full HTML "rich text" styling for all text rendering (e.g., in `Label` widget) -- can decorate any text with inline tags (`<strong>`, `<em>` etc), and even include links.

* Compiles in second(s), compared to hour(s) for Qt, and is fully native with no cgo dependency on Linux and Windows, and minimal cgo (necessary) on MacOS.

* Fully self-contained -- does *not* use OS-specific native widgets -- results in simple, elegant, consistent code across platforms, and is fully `HiDPI` capable and scalable using standard `Shift+Ctrl/Cmd+Plus or Minus` key, and in `Preferences` (press `Ctrl+Alt+P` in any window to get Prefs editor).

* `SVG` element (in `svg` sub-package) supports full SVG rendering -- used for Icons internally and available for advanced graphics displays -- see `gi/examples/svg` for viewer and start on editor, along with a number of test .svg files.

* Advanced **Model / View** paradigm with `reflect`ion-based view elements that display and manipulate all the standard Go types (in `giv` sub-package), from individual types (e.g., int, float display in a `SpinBox`, "enum" const int types in a `ComboBox` chooser) to composite data structures, including `StructView` editor of `struct` fields, `MapView` and `SliceView` displays of `map` and `slice` elements (including full editing / adding / deleting of elements), and full-featured `TableView` for a `slice`-of-`struct` and `TreeView` for GoKi trees.
	+ `TreeView` enables a built-in GUI editor / inspector for designing gui elements themselves.  Just press `Control+Alt+I` in any window to pull up this editor / inspector.  Scene graphs can be automatically saved / loaded from JSON files, to provide a basic GUI designer framework -- just load and add appropriate connections..

![Screenshot of Widgets demo](screenshot.png?raw=true "Screenshot of Widgets demo")

# Code Map

* `examples/widgets` -- main example widget gallery -- `go get ...` `go build` in there to give it a try -- see README there for more info.  Many other demos / tests in `examples/*`.
* `node*.go` -- `NodeBase`, `Node2DBase`, `3D` structs and interfaces -- all Gi nodes are of this type.
* `geom2d.go` -- `Vec2D` is main geom type used for 2D, plus transform `Matrix2D`.
* `paint.go` -- `Paint` struct that does all the direct rendering, uses `gg`-based API but now uses the `srwiley/renderx` rendering system which supports SVG 2.0, and is very fast.
	+ `stroke.go`, `fill.go` -- `StrokeStyle` and `FillStyle` structs for stroke, fill settings
	+ `color.go` -- `ColorSpec` for full gradient support, `Color` is basic `color.Color` compatible RGBA type with many additional useful methods, including support for `HSL` colorspace -- see [Wiki Color](https://github.com/goki/gi/wikiColor) for more info.
* `style.go` -- `Style` and associated structs for CSS-based `Widget` styling.
* `viewport2d.go` -- `Viewport2D` that has an `Image.RGBA` that `Paint` renders onto.
* `window.go` -- `Window` is the top-level window that manages an OS-specific `oswin.Window` and handles the event loop.
	+ `oswin` is a modified version of the back-end OS-specific code from Shiny: https://github.com/golang/exp/tree/master/shiny -- originally used https://github.com/skelterjohn/go.wde but shiny is much faster for updating the window because it is gl-based, and doesn't have any other dependencies (removed dependencies on mobile, changed the event structure to better fit needs here).
* `font.go`, `text.go` -- `FontStyle`, `TextStyle`, and `TextRender` that manages rich-text rendering in a powerful, efficient manner (layered on `RuneRender` and `SpanRender`).  `FontStyle` contains the global `color`,  `background-color`, and `opacity` values, to make these easily avail to the `TextRender` logic.
* `layout.go` -- main `Layout` object with various ways of arranging widget elements, and `Frame` which does layout and renders a surrounding frame.
* `widget.go` -- `WidgetBase` for all widgets.
* `buttons.go` -- `ButtonBase`, `Button` and other basic button types.
* `action.go` -- `Action` is a Button-type used in menus and toolbars, with a simplified `ActionSig` signal.
* `bars.go` -- `MenuBar` and `ToolBar`
* `sliders.go` -- `SliderBase`, `Slider`, `ScrollBar`.
* `textfield.go` for `TextField`, `label.go` for `Label`, etc -- also defines the `gi.Labeler` interface and `ToLabel` converter method (which falls back on kit.ToString using Stringer), which is used for generating a gui-appropriate label for an object -- e.g., for reflect.Type it just presents the raw type name without prefix.
* `icon.go` for `Icon` wrapper around svg icons (in `svg` sub-package)
* Sub-packages:
	+ `svg` -- has all the SVG nodes (`Path`, `Rect` etc) plus `io.go` parser
	+ `giv` -- has all the `*View` elements
	+ `gimain` -- provides a meta-package wrapper to simplify imports for `main` apps -- also does relevant final platform-specific customization
	+ `units` -- CSS unit representation

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

The best way to see how the system works are in the `examples` directory, and by interactively modifying any existing gui using the interactive reflective editor via `Control+Alt+E`.

# Status

Currently at a **pre-beta** level (**DON'T RECOMMEND USING RIGHT NOW** -- come back in a few weeks after announcement on go-nuts email list).

* Major push underway to get to the following target for a **beta** level release:

* All major functionality is in place, and API is stable and only very minor changes will be allowed going forward.  The system is now ready for wider adoption.

* Everything has been thoroughly tested, but generally only by a small number of core developers / users.

* Please file Issues for anything that does not work (except as noted below under TODO)

# TODO

* TextView:
	+ losing line numbers when typing new text, in some select cases
	+ word-level functions: forward, back, delete etc.  ctrl+ backspace is back.
	+ optimize wraparound multi-span rendering -- if no change in start of line, don't re render!
	+ cache keys if typing faster than can be processed -- need to check lag on new key?  later..
	+ can lose focusactive and not get it back despite being able to type -- reactivate on key input?  should be happening already but maybe missing.
	+ command keys in keyfuns (C-x, C-c, M-x), sequences, etc
	+ clipboard history
	+ clipboard "registers" (C-x x <label>, C-x g <label>)

* Splitview: 
	+ some lowpri keyfuns for collapsing and expanding  

* CSS class = x should bring in properties for that class into top-level CSS
  for all below -- not sure it does that - nested classes.  need to figure that out really.

* slice of ki: do proper add so it works directly by editing children

* tom's checkbox bug!

* that changed flag on prefs would be reassuring to know that save actually worked..

* textfield should scroll layout so that *cursor* is always in view, when editing..

* fileview should add extension to filename if only one extension provided, if
  user types in a new filename.. 

* fix text scrolling-off-top rendering finally

* fix the style-props context -- need an overall prop on objects -- in type presumably.  completer needs to know about this too.

* update gi/doc.go with final readme notes etc!  add docs to this README about
  "what can you do with demos?" kind of thing..

* tab widget basic fix, and integrate with tree view editor? Popups show up in
  a separate tab? ultimately want multi-row super-tabs -- flow layout..  with
  dnd..

* add context menus for delete, rename to fileview -- would be a good test to
  make sure context menu api is sufficiently flexible. (use new ContextMenu type properties?)

* update widgets README

* Reminder: grep all todo: in code

	



# gi

GoGi is part of the GoKi Go language (golang) full strength tree structure system (ki = tree in Japanese)

`package gi` -- scenegraph based 2D and 3D GUI / graphics interface (Gi) in Go

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/gi)](https://goreportcard.com/report/github.com/goki/gi)
[![GoDoc](https://godoc.org/github.com/goki/gi?status.svg)](http://godoc.org/github.com/goki/gi)

NOTE: Requires Go version `1.10+` due to use of `math.Round`.

See the [Wiki](https://github.com/goki/gi/wiki) for more docs, discussion, etc.

GoGi uses the GoKi tree infrastructure to implement a simple, elegant, GUI framework in full native idiomatic Go (with minimal OS-specific backend interfaces based on the Shiny drivers).  The overall design is an attempt to integrate existing standards and conventions from widely-used frameworks, including Qt (overall widget design), HTML / CSS (styling), and SVG (rendering).  Rendering in SVG is directly supported by the GoGi 2D scenegraph, with enhanced functionality for interactive GUI's, and the layout etc should be able to support at least a decent subset of HTML.  This 2D framework also integrates with a (planned) 3D scenegraph, supporting interesting combinations of these frameworks.  Currently GoGi is focused on desktop systems, but nothing prevents adaptation to mobile.

GoGi also incorporates reflection-based View elements that enable automatic representation and editing of all native Go data structures, providing a built-in native first-order GUI framework with no additional coding.  This enables built-in GUI editor / inspector for designing gui elements themselves.  Just press `Control+Alt+E` (or `I`) on any window to pull up this editor / inspector.  Scene graphs can be automatically saved / loaded from JSON files, to provide a basic GUI designer framework -- just load and add appropriate connections..

**IMPORTANT for Linux users:** You need to install the Arial TTF font to get decent-looking rendering: https://askubuntu.com/questions/651441/how-to-install-arial-font-in-ubuntu

![Screenshot of Widgets demo](screenshot.png?raw=true "Screenshot of Widgets demo")

# Code Map

* `examples/widgets` -- main example widget gallery -- `go get ...` `go build` in there to give it a try -- see README there for more info
* `node*.go` -- `NodeBase`, `Node2DBase`, `3D` structs and interfaces -- all Gi nodes are of this type
* `geom2d.go` -- `Vec2D` is main geom type used for 2D, plus transform matrix
* `paint.go` -- `Paint` struct that does all the direct rendering, uses `gg`-based API but now uses the `srwiley/renderx` rendering system which supports SVG 2.0, and is very fast
	+ `stroke.go`, `fill.go` -- `StrokeStyle` and `FillStyle` structs for stroke, fill settings
* `style.go` -- `Style` and associated structs for CSS-based `Widget` styling
* `viewport2d.go` -- `Viewport2D` that has an `Image.RGBA` that `Paint` renders onto
* `window.go` -- `Window` is the top-level window that manages an OS-specific `oswin.Window` and handles the event loop.
	+ `oswin` is a modified version of the back-end OS-specific code from Shiny: https://github.com/golang/exp/tree/master/shiny -- originally used https://github.com/skelterjohn/go.wde but shiny is much faster for updating the window because it is gl-based, and doesn't have any other dependencies (removed dependencies on mobile, changed the event structure to better fit needs here).
* `shapes2d.go` -- All the basic 2D SVG-based shapes: `Rect`, `Circle` etc
* `font.go`, `text.go` -- `FontStyle`, `TextStyle`, `Text2D` node
* `layout.go` -- main `Layout` object with various ways of arranging widget elements, and `Frame` which does layout and renders a surrounding frame
* `widget.go` -- `WidgetBase` for all widgets
* `buttons.go` -- `ButtonBase`, `Button` and other basic command button types
* `action.go` -- `Action` is a Button-type used in menus and toolbars, with a simplified `ActionTriggered` signal
* `sliders.go` -- `SliderBase`, `Slider`, `ScrollBar`
* `textwidgets.go` -- `Label`, `TextField`, `ComboBox` -- also defines the `gi.Labeler` interface and `ToLabel` converter method (which falls back on kit.ToString using Stringer), which is used for generating a gui-appropriate label for an object -- e.g., for reflect.Type it just presents the raw type name without prefix.
* `*view.go` -- `TreeView` widget shows a graphical view of a tree, `StructView` for editing structs, `MapView`, `SliceView`, etc.  `ValueView` framework for managing mapping between `reflect.Value`'s and gui widgets for displaying them.

# Overview

There are three main types of 2D nodes:

* `Viewport2D` nodes that manage their own `oswin.Image` bitmap and can upload that directly to the `oswin.Texture` that then uploads directly to the `oswin.Window`.  The parent `Window` has a master `Viewport2D` that backs the entire window, and is what most `Widget`'s render into.
		+ Popup `Dialog` and `Menu`'s have their own viewports that are layered on top of the main window viewport.
		+ `SVG` and its subclass `Icon` are containers for SVG-rendering nodes.

* `Widget` nodes that use the full CSS-based styling (e.g., the Box model etc), are typically placed within a `Layout` -- they use `units` system with arbitrary DPI to transform sizes into actual rendered `dots` (term for actual raw resolution-dependent pixels -- "pixel" has been effectively co-opted as a 96dpi display-independent unit at this point).  Widgets have non-overlapping bounding boxes (`BBox`).

* `SVG` rendering nodes that directly set properties on the `Paint` object and typically have their own geometry etc -- they should be within a parent `SVG` viewport, and their geom units are determined entirely by the transforms etc and we do not support any further unit specification -- just raw float values.

General Widget method conventions:
* `SetValue` kinds of methods are wrapped in `UpdateStart` / `End`, but do NOT emit a signal
* `SetValueAction` calls `SetValue` and emits the signal
This allows other users of the widget that also recv the signal to not trigger themselves, but typically you want the update, so it makes sense to have that in the basic version.  ValueView in particular requires this kind of behavior.

The best way to see how the system works are in the `examples` directory, and by interactively modifying any existing gui using the interactive reflective editor via `Control+Alt+E`.

# Status

Currently at an **alpha** level release:

* Core code is all functional, and on the Mac (main dev) platform, everything should generally always work smoothly -- linux and windows tend to lag a bit, but the OS-specific changes are getting less and less as things mature, so they should generally be more stable and similarly functional.

* Many things are missing and only skeletally present -- the initial release goal was to get the full set of interdependent parts up and running, and obtain any input about overall design issues.  Will be fleshing out all this stuff in the next couple of months, and then transition to a more standard issue-tracker based management of tasks.

# TODO

## Platforms / oswin

* clip.Board: windows & mac converted over to mimedata multipart encoding for more complex cases (e.g., treeview) -- update linux to use the same so everything is consistent, and much simpler!
  
* enable dnd to use OS DND when moves outside window

* windows:
	+ support SetPos window method (and probably need to track movement)

* linux:
	+ moving window isn't updating pos of new windows (now fixed? test)
	+ do similar font name updating as done on Windows now

* mac:
	+ impl setPos
	+ win pos getting incremented by title.. so annoying
	+ positioning not working for laptop display -- use non-dpi coords?

* lifecycle not really being used, and closing last window doesn't kill app -- need to clarify that logic vis-a-vis main app window, main app menu / toolbar etc.

## General / Widgets

* add margin for para in text

* tooltip prevents button from opening dialog, causes hang sometimes -- close tooltip right away?

* #icon props not getting used for resizing icons in button.

* CSSAgg needs a tag for no-inline -- don't inline its map ever
* also, inline choice could be smarter if we knew what was in the map, but...

* selector for different standard keyfun maps..

* bitflag elements, e.g., TypeDecoration in FontStyle -- field should in
  general be a uint32 or uint64, but bitflag uses int32, int64 which is fine,
  but key problem is how to associate the enum with the field then?  bit-set
  values don't match the defined ints.. but who cares?  simplest to just use
  type. but for bitflag never want consts to be int64, but often do want flags
  field to be int64..  for 32bit case, not that big a deal, and for most
  user-facing cases, int32 is sufficient, so focus on that case??

* Use same technique as IconName for FontName and that can be used to trigger chooser for font_family.

* search for structtableview, treeview
* DND and copy/paste for structtableview

* scroll should go to the sub-widget first before going to the layout: add a First and Last event signal in addition to the regular one, plus registering for each.

* tab widget basic fix, and integrate with tree view editor? Popups show up in a separate tab?

* menu separator not rendering..

* add MenuBar / Toolbar -- just a layout really, with some styling?

* main menu (mac, other platforms?)

* arg view / dialog and button tags

* Structview: condshow / edit
	
* keyboard shortcuts -- need to register with window / event manager on a signal list..

* button repeat settings when button is held down -- esp for spinner buttons -- probably off by default

* text translation functionality -- just do it automatically for everything, or require user to specifically request it for each string??  prefer a Stringer kind of method?  or a big map of translations?  send it to google??

* DND needs enter / exit events so nodes can signal their ability to accept drop..  later..

* Reminder: grep all todo: in code -- lots!

## layout

* really want an additional spacing parameter on layout -- needs to be
  separate from margin / padding which just apply to the frame-like property
  -- easy

* add new TableGrid widget that combines a Frame Grid Layout with a top row of
  header action labels that just grab the sizes from the grid, and also supports clicking to
  select sort order

* grid not using spans

* Layout flow types


## Rendering / SVG

* icons/actions/adjusthsl.svg has a rectangle with an opacity gradient -- not
  rendering -- unclear if any gradients are rendering properly

* default join not looking right for some test cases..

* clip-path and ClipPath element..

* impl ViewBox options

* path: re-render data string after parsing to be more human friendly.

## Missing Widgets

see http://doc.qt.io/qt-5/qtquickcontrols2-differences.html for ref

+ RadioButton -- checkbox + mutex logic -- everyone within same parent is mutex -- easy
+ ProgressBar -- very simple
+ TextArea -- go full editor..

## Remaining features for widgets

+ FileView view and dialog -- various, see todo in fileview.go
+ TextField -- undo, constraints
+ TreeView -- see about LI, UL lists..
+ TabWidget -- needs updating

## Performance issues

* Styling and ToDots
	+ currently compiling default of main style, but derived state / sub styles MUST be styled dynamically otherwise css and props changes don't propagate -- this doesn't add much -- was previously caching those but then they were not responsive to overall changes.
	+ Lots of redundant ToDots is happening, but it is difficult to figure out exactly when minimal recompute is necessary.  right now only for nil props.  computing prop diffs might be more expensive and complex than just redoing everything.
	+ switched to map!  old: 4.6sec on FindConnectionIndex when making new Connections -- this is most of the time in Init2D
	

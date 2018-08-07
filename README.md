# gi

GoGi is part of the GoKi Go language (golang) full strength tree structure system (ki = tree in Japanese)

`package gi` is a scenegraph-based 2D and 3D GUI / graphics interface (Gi) in Go

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/gi)](https://goreportcard.com/report/github.com/goki/gi)
[![GoDoc](https://godoc.org/github.com/goki/gi?status.svg)](http://godoc.org/github.com/goki/gi)

NOTE: Requires Go version `1.10+` due to use of `math.Round`.

**IMPORTANT for Linux users:** You need to install the standard Windows TTF fonts (e.g., Arial, etc) to get decent-looking rendering: https://askubuntu.com/questions/651441/how-to-install-arial-font-in-ubuntu

See the [Wiki](https://github.com/goki/gi/wiki) for more docs, discussion, etc.

GoGi uses the GoKi tree infrastructure to implement a simple, elegant, GUI framework in full native idiomatic Go (with minimal OS-specific backend interfaces based on the Shiny drivers).  The overall design is an attempt to integrate existing standards and conventions from widely-used frameworks, including Qt (overall widget design), HTML / CSS (styling), and SVG (rendering).  This 2D framework also integrates with a (planned) 3D scenegraph, supporting interesting combinations of these frameworks.  Currently GoGi is focused on desktop systems, but nothing prevents adaptation to mobile.

# Main Features

* All standard widgets: `Button`, `Menu`, `Slider`, `TextField`, `SpinBox`, `ComboBox` etc, with tooltips, hover, focus, copy / paste (full native clipboard support), drag-n-drop -- the full set of standard GUI functionality.  See `gi/examples/widgets` for a demo.

* Powerful `Layout` logic auto-sizes everything -- very easy to configure interfaces that just work across different scales, resolutions, platforms.  Automatically remembers and reinstates window positions and sizes across sessions.

* CSS-based styling allows easy customization of everything -- native style properties are fully HTML compatible (with all standard `em`, `px`, `pct` etc units), including full HTML "rich text" styling for all text rendering (e.g., in `Label` widget) -- can decorate any text with inline tags (`<strong>`, `<em>` etc).

* Compiles in second(s), compared to hour(s) for Qt, and is fully native with no cgo dependency on Linux and Windows, and minimal cgo (necessary) on MacOS.

* Fully self-contained -- does *not* use OS-specific native widgets -- results in simple, elegant, consistent code across platforms, and is fully `HiDPI` capable and scalable using standard `Shift+Ctrl/Cmd+Plus or Minus` key, and in `Preferences` (press `Ctrl+Alt+P` in any window to get Prefs editor).

* `SVG` element (in `svg` sub-package) supports full SVG rendering -- used for Icons internally and available for advanced graphics displays -- see `gi/examples/svg` for viewer and start on editor, along with a number of test .svg files.

* Advanced **Model / View** paradigm with `reflect`ion-based view elements that display and manipulate all the standard Go types (in `giv` sub-package), from individual types (e.g., int, float display in a `SpinBox`, "enum" const int types in a `ComboBox` chooser) to composite data structures, including `StructView` editor of `struct` fields, `MapView` and `SliceView` displays of `map` and `slice` elements (including full editing / adding / deleting of elements), and full-featured `TableView` for a `slice`-of-`struct` and `TreeView` for GoKi trees.
	+ `TreeView` enables a built-in GUI editor / inspector for designing gui elements themselves.  Just press `Control+Alt+E` (or `I`) in any window to pull up this editor / inspector.  Scene graphs can be automatically saved / loaded from JSON files, to provide a basic GUI designer framework -- just load and add appropriate connections..

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

* lifecycle not really being used, and closing last window doesn't kill app -- need to clarify that logic vis-a-vis main app window, main app menu / toolbar etc.

## General / Widgets

* update gi/doc.go with final readme notes etc!

* MenuBar has gradient but actions don't -- add gradient to actions that show
  up in menubar -- can't not have a background..
  
* key.ChordShortForm transforms chord into short-form suitable for menu
  shortcuts -- not clear if mac command rune will render -- seems to have broken arial.

* add Toolbar based on MenuBar

* event injection -- Edit/Cut/Copy/Paste convenience method, and these just
  inject the corresponding key press -- problem: they do so when the menu is
  still there..  would work ok in buttons probably.

* main menu (mac, other platforms?)

* Label renders links, uses HandPointing for links, delivers link clicked signal

* drag should be stateful -- only drag current item -- logic is in there but not working properly

* use screenprefs

* search for tableview, treeview
* DND and copy/paste for tableview

* combobox getting text cutoff -- descenders and []

* Use same technique as IconName for FontName and that can be used to trigger chooser for font_family.

* add margin for para in text

* tooltip prevents button from opening dialog, causes hang sometimes -- close tooltip right away?

* bitflag elements, e.g., TypeDecoration in FontStyle -- field should in
  general be a uint32 or uint64, but bitflag uses int32, int64 which is fine,
  but key problem is how to associate the enum with the field then?  bit-set
  values don't match the defined ints.. but who cares?  simplest to just use
  type. but for bitflag never want consts to be int64, but often do want flags
  field to be int64..  for 32bit case, not that big a deal, and for most
  user-facing cases, int32 is sufficient, so focus on that case??

* tab widget basic fix, and integrate with tree view editor? Popups show up in a separate tab? ultimately want multi-row super-tabs -- flow layout..  with dnd..

* arg view / dialog and button tags

* button repeat settings when button is held down -- esp for spinner buttons -- probably off by default

* text language translation functionality -- just do it automatically for everything, or require user to specifically request it for each string??  prefer a Stringer kind of method?  or a big map of translations?  send it to google??

* Reminder: grep all todo: in code -- lots!

### After Beta Release

* undo -- sub-package, use diff package (prob this: https://github.com/sergi/go-diff) on top of json outputs, as in emergent diffmgr
	+ map of diff records for each top-level entity -- can be many of these in parallel (e.g., textfield vs. ki tree etc)
	+ records themselves are sequential slices of diff records and commands, with same logic as emergent
	+ diffing happens in separate routine..

* DND needs enter / exit events so nodes can signal their ability to accept drop..  later..

* Cursors for various systems need extra custom ones to fill in standard set,
  and support general custom cursors as well

## layout

* really want an additional spacing parameter on layout -- needs to be
  separate from margin / padding which just apply to the frame-like property
  -- easy

* grid not using spans

* Layout flow types


## Rendering / SVG

* TestShapes5.svg and all the new icons added to svg have issues -- possibly clip-path?
* radial gradient is off still, at least in userspace units

* flowRoot in fig_vm_as_tug_of_war creates big black box
* default join not looking right for some test cases -- getting white holes
* clip-path and ClipPath element..

* impl ViewBox options

* path: re-render data string after parsing to be more human friendly.

## Missing Widgets

see http://doc.qt.io/qt-5/qtquickcontrols2-differences.html for ref

+ RadioButton -- checkbox + mutex logic -- everyone within same parent is mutex -- easy
+ ProgressBar -- very simple
+ TextArea -- go full editor..

## Performance issues

* Styling and ToDots
	+ currently compiling default of main style, but derived state / sub styles MUST be styled dynamically otherwise css and props changes don't propagate -- this doesn't add much -- was previously caching those but then they were not responsive to overall changes.
	+ Lots of redundant ToDots is happening, but it is difficult to figure out exactly when minimal recompute is necessary.  right now only for nil props.  computing prop diffs might be more expensive and complex than just redoing everything.
	+ switched to map!  old: 4.6sec on FindConnectionIndex when making new Connections -- this is most of the time in Init2D
	

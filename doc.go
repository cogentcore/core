// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package Gi (GoGi) provides a Graphical Interface based on GoKi Tree Node structs

2D and 3D scenegraphs supported, each rendering to respective Viewport2D or 3D
which in turn can be integrated within the other type of scenegraph.

Within 2D scenegraph, the following are supported

	* Widget nodes for GUI actions (Buttons, Views etc) -- render directly via Paint
	* Layouts for placing widgets
	* CSS-based styling, directly on Node Props (properties), and CSS StyleSheet
	* SVG container for SVG elements: shapes, paths, etc (in svg package)
    * Icons wrappers around an SVG container

Layout Logic

All 2D scenegraphs are controlled by the Layout, which provides the logic for
organizing widgets / elements within the constraints of the display.
Typically start with a vertical LayoutCol in a viewport, with LayoutCol's
within that, or a LayoutGrid for more complex layouts:

	win := gi.NewWindow2D("test window", width, height)
	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	vlay := vpfill.AddNewChildNamed(gi.KiT_Layout, "vlay").(*gi.Layout)
	vlay.Lay = gi.LayoutCol

	row1 := vlay.AddNewChildNamed(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutRow

	...

    vp.UpdateEnd(updt)

Controlling the layout involves the following style properties:

	* width / height: sets the preferred size of item -- layout tries to give
      this amount of space unless it can't in which case it falls back on:

	* min-width / min-height: minimum size -- will not scale below this size.
      if not specified, it defaults to 1 em (the size of 1 character)

	* max-width / max-height: maximum size -- will not exceed this size if
      specified, otherwise if 0 it is ignored and preferred size is used.  If
      a negative number is specified, then the item stretches to take up
      available room.  The Stretch node is a builtin type that has this
      property set automatically, and can be added to any layout to fill up
      any available space.  The preferred size of the item is used to
      determine how much of the space is used by each stretchable element, so
      you can set that to achieve different proportional spacing.  By default
      the Stretch is just the minumum 1em in preferred size.

	* align-horiz / align-vert: for the other dimension in a Layout (e.g., for
      LayoutRow, the vertical dimension) this specifies how the items are
      aligned within the available space (e.g., tops, bottoms, centers).  In
      the dimension of the Layout (horiz for LayoutRow) it determines how
      extra space is allocated (only if there aren't any infinitely stretchy
      elements), e.g., right / left / center or justified.

	* SetFixedWidth / Height method can be used to set all size params to the
      same value, causing that item to be definitively sized.  This is
      convenient for sizing the Space node which adds a fixed amount of space
      (1em by default).

    * See the wiki for more detailed documentation.

Signals

All widgets send appropriate signals about user actions -- Connect to those
and check the signal type to determine the type of event.

Views

Views are Widgets that automatically display and interact with structured
data, providing powerful GUI elements, with extensive property-based
customization options.  They can easily provide the foundation for entire
apps.

ValueView

The ValueView provides a common API for representing values (int, string, etc)
in the GUI, and are used by more complex views (StructView, MapView,
SliceView, etc) to represents the elements of those data structures.

The ValueViewer interface provides a standard Go way of customizing the GUI
display for any particular type -- just define a ValueView() ValueView method
for any type and it will use that type of ValueView to display and interact
with that type.

Do Ctrl+Alt+E in any window to pull up the GiEditor which will show you ample
examples of the ValueView interface in action, and also allow you to customize
your GUI.

TreeView

The TreeView displays GoKi Node Trees, using a standard tree-browser with
collapse / open widgets and a menu for typical actions such as adding and
deleting child nodes, along with full drag-n-drop and clipboard Copy/Cut/Paste
functionality.  You can connect to the selection signal to e.g., display a
StructView field / property editor of the selected node.

SVG for Icons, Displays, etc

SVG (Structured Vector Graphics) is used icons, and for rendering any kind of
graphical output (drawing a graph, dial, etc).

SVGNodeBase is the base type for all SVG elements -- unlike Widget nodes, SVG
nodes do not use layout logic, and just draw directly into a parent SVG
viewport, with cumulative transforms determining drawing position, etc.  The
BBox values are only valid after rendering for these nodes.

Overlay

The gi.Window contains an OverlayVp viewport with nodes that are rendered on
top of the regular scenegraph -- this is used for drag-n-drop and other kinds
of transient control / manipulation functionality.  Overlay elements are not
subject to the standard layout constraints (via having the Overlay NodeFlag
set)

*/
package gi

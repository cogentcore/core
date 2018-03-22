// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package Gi (GoGi) provides a Graphical Interface based on GoKi Tree Node structs

2D and 3D scenegraphs supported, each rendering to respective Viewport2D or 3D
which in turn can be integrated within the other type of scenegraph.

Within 2D scenegraph, the following are supported

	* SVG-based rendering nodes for basic shapes, paths, curves, arcs
	* Widget nodes for GUI actions (Buttons, Views etc)
	* Layouts for placing widgets
	* CSS-based styling, directly on Node Props (properties), and css sheets
	* HTML elements -- the 2D scenegraph can render html documents

Layout Logic

For Widget-based displays, *everything* should be contained in a Layout,
because a layout provides the primary logic for organizing widgets within the
constraints of the display.  Typically start with a vertical LayoutCol in a
viewport, with LayoutCol's within that, or a LayoutGrid for more complex
layouts:

	win := gi.NewWindow2D("test window", width, height)
	win.UpdateStart()
	vp := win.WinViewport2D()

	vpfill := vp.AddNewChildNamed(gi.KiT_Viewport2DFill, "vpfill").(*gi.Viewport2DFill)
	vpfill.SetProp("fill", "#FFF") // white background

	vlay := vpfill.AddNewChildNamed(gi.KiT_Layout, "vlay").(*gi.Layout)
	vlay.Lay = gi.LayoutCol

	row1 := vlay.AddNewChildNamed(gi.KiT_Layout, "row1").(*gi.Layout)
	row1.Lay = gi.LayoutRow

	...

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

Signals

All widgets send appropriate signals about user actions -- Connect to those
and check the signal type to determine the type of event.

Views

Views are Widgets that automatically display and interact with structured
data, providing powerful GUI elements, with extensive property-based
customization options.  They can easily provide the foundation for entire
apps.

NodeWidget

The NodeWidget displays GoKi Node Trees, using a standard tree-browser with
collapse / open widgets and a menu for typical actions such as adding and
deleting child nodes, along with full drag-n-drop and clipboard Copy/Cut/Paste
functionality.  You can connect to the selection signal to e.g., display a
StructWidget field / property editor of the selected node.

The properties controlling the NodeWidget include:

	* "collapsed" -- node starts out collapsed (default is open)
    * "background-color" -- color of the background of node box
	* "color" -- font color in rendering node label
	* "read-only" -- do not display the editing menu actions

StructWidget

The StructWidget displays an arbitrary struct object, showing its fields and
values, in an editable form, with type-appropriate widgets.

*/
package gi

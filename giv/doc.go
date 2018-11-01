// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package GiV (GoGi Views) provides a model / view framework to view Go data using reflection

Views are Widgets that automatically display and interact with standard Go
data, including structs, maps, slices, and the primitive data elements
(string, int, etc).  This implements a form of model / view separation between data
and GUI representation thereof, where the models are the Go data elements themselves.

This provides automatic, powerful GUI access to essentially any data in any
other Go package.  Furthermore, the ValueView framework allows for easy
customization and extension of the GUI representation, based on the classic Go
"Stringer"-like interface paradigm -- simply define a ValueView() method on
any type, returning giv.ValueView that manages the interface between data
structures and GUI representations.

See the wiki at: https://github.com/goki/gi/wiki/Views for more extensive docs.

Some of the most important view elements are:

ValueView

The ValueView provides a common API for representing values (int, string, etc)
in the GUI, and are used by more complex views (StructView, MapView,
SliceView, etc) to represents the elements of those data structures.

Do Ctrl+Alt+I in any window to pull up the GoGiEditor which will show you ample
examples of the ValueView interface in action, and also allow you to customize
your GUI.

TreeView

The TreeView displays GoKi Node Trees, using a standard tree-browser with
collapse / open widgets and a menu for typical actions such as adding and
deleting child nodes, along with full drag-n-drop and clipboard Copy/Cut/Paste
functionality.  You can connect to the selection signal to e.g., display a
StructView field / property editor of the selected node.

TableView

TableView displays a slice-of-struct as a table with columns as the struct fields
and rows as the elements in the struct.  You can sort by the column headers
and it supports full editing with drag-n-drop etc.  If set to Inactive, then it
serves as a chooser, as in the FileView.

MethodView

This is actually a collection of methods that provide a complete GUI for calling
methods.  Property lists defined on the kit Type registry are used for specifying
the methods to call and their properties.  Much of your toolbar and menu level
GUI can be implemented in this system.  See gi/prefs.go and giv/prefsview.go for
how the GoGi Prefs dialog is implemented, and see the gide project for a more
complex case.

*/
package giv

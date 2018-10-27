// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/mimedata"
)

// Clipper is the interface for standard clipboard operations
// Types can use this interface to support extensible clip functionality
// used in all relevant valueview types in giv package (e.g., TreeView)
type Clipper interface {
	// Copy copies item to the clipboard
	// e.g., use oswin.TheApp.ClipBoard(tv.Viewport.Win.OSWin).Write(md)
	// where md is mime-encoded data for the object
	Copy(reset bool)

	// Cut cuts item to the clipboard, typically calls Copy and then deletes
	// itself
	Cut()

	// Paste pastes from clipboard to item, e.g.,
	// md := oswin.TheApp.ClipBoard(tv.Viewport.Win.OSWin).Read([]string{mimedata.AppJSON})
	// reads mime-encoded data from the clipboard, in this case in the JSON format
	Paste()
}

// DragNDropper is the interface for standard drag-n-drop actions
// Types can use this interface to support extensible DND functionality
// used in all relevant valueview types in giv package (e.g., TreeView)
type DragNDropper interface {
	// Drop is called when something is dropped on this item
	// the mod is either dnd.DropCopy for a copy-like operation (the default)
	// or dnd.Move for a move-like operation (with Shift key held down)
	// drop must call Window.FinalizeDragNDrop with the mod actually used
	// to have the source update itself
	Drop(md mimedata.Mimes, mod dnd.DropMods)

	// Dragged is called on source of drag-n-drop after the drop is finalized
	Dragged(de *dnd.Event)
}

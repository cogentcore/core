// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"reflect"

	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// This file contains all the special-purpose interfaces
// beyond the basic Node interface

// Updater defines an interface for something that has an Update() method
// this will be called by GUI actions that update values of a type
// including struct, slice, and map views in giv
type Updater interface {
	// Update updates anything in this type that might depend on other state
	// which could have just been changed.  It is the responsibility of the
	// type to determine what might have changed, or just generically update
	// everything assuming anything could have changed.
	Update()
}

/////////////////////////////////////////////////////////////
//  Labeler

// Labeler interface provides a GUI-appropriate label for an item,
// via a Label() string method.
// Use ToLabel converter to attempt to use this interface and then fall
// back on Stringer via kit.ToString conversion function.
type Labeler interface {
	// Label returns a GUI-appropriate label for item
	Label() string
}

// ToLabel returns the gui-appropriate label for an item, using the Labeler
// interface if it is defined, and falling back on kit.ToString converter
// otherwise -- also contains label impls for basic interface types for which
// we cannot easily define the Labeler interface
func ToLabel(it interface{}) string {
	lbler, ok := it.(Labeler)
	if !ok {
		switch v := it.(type) {
		case reflect.Type:
			return v.Name()
		case ki.Ki:
			return v.Name()
		}
		return kit.ToString(it)
	}
	return lbler.Label()
}

// ToLabeler returns the Labeler label, true if it was defined, else "", false
func ToLabeler(it interface{}) (string, bool) {
	if lbler, ok := it.(Labeler); ok {
		return lbler.Label(), true
	}
	return "", false
}

// SliceLabeler interface provides a GUI-appropriate label
// for a slice item, given an index into the slice.
type SliceLabeler interface {
	// ElemLabel returns a GUI-appropriate label for slice element at given index
	ElemLabel(idx int) string
}

/////////////////////////////////////////////////////////////
//  Clipper

// Clipper is the interface for standard clipboard operations
// Types can use this interface to support extensible clip functionality
// used in all relevant valueview types in giv package (e.g., TreeView)
type Clipper interface {
	// MimeData adds mimedata  to given record to represent item on clipboard
	MimeData(md *mimedata.Mimes)

	// Copy copies item to the clipboard
	// e.g., use oswin.TheApp.ClipBoard(tv.ParentWindow().OSWin).Write(md)
	// where md is mime-encoded data for the object
	Copy(reset bool)

	// Cut cuts item to the clipboard, typically calls Copy and then deletes
	// itself
	Cut()

	// Paste pastes from clipboard to item, e.g.,
	// md := oswin.TheApp.ClipBoard(tv.ParentWindow().OSWin).Read([]string{filecat.DataJson})
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

	// DropExternal is called when something is dropped on this item from
	// an external source (not within same app).
	// the mod is either dnd.DropCopy for a copy-like operation (the default)
	// or dnd.Move for a move-like operation (with Shift key held down)
	// drop DOES NOT need to call Window.FinalizeDragNDrop with the mod actually used
	// to have the source update itself -- no harm if it does however, as the source
	// will be nil.
	DropExternal(md mimedata.Mimes, mod dnd.DropMods)
}

// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clip defines the system clipboard for the GoGi GUI system.  Data is
// represented using mimedata type codes and []byte raw data -- multiple
// different representations can be available -- in general when writing to
// the clipboard, having a text/plain version in addition to a more specific
// format is a good idea
package clip

import (
	"github.com/goki/gi/oswin/mimedata"
)

// clip.Board interface defines the methods for reading and writing data to
// the system clipboard -- uses mimedata to represent the data
type Board interface {

	// Read attempts to read data of the given MIME type(s), in preference
	// order, from the clipboard, returning mimedata.Mimes which can
	// potentially have multiple different types for the same data if an "Any"
	// mime type was used, or multiple items if multiple of the same type were
	// on the clipboard
	Read(types []string) mimedata.Mimes

	// Write writes given mimedata to the clipboard -- in general having a
	// text/plain version in addition to a more specific format is a good idea
	// -- clearFirst calls Clear before writing (generally want to do this)
	Write(data mimedata.Mimes, clearFirst bool) error

	// Clear clears the clipboard
	Clear()
}

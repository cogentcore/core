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
// the system clipboard -- uses mimedata to represent the data.  Due to
// limitations of Windows (and linux to a lesser extent), a multipart MIME
// formatted string is used if there are multiple elements in the mimedata,
// with any binary data text-encoded using base64
type Board interface {

	// IsEmpty returns true if there is nothing on the clipboard to read.  Can
	// be used for inactivating a Paste menu.
	IsEmpty() bool

	// Read attempts to read data of the given MIME type(s), in preference
	// order, from the clipboard, returning mimedata.Mimes which can
	// potentially have multiple types / multiple items, etc -- if first type
	// listed is a text type, then text-based retrieval is assumed -- always
	// put the most specific desired type first -- anything else present will
	// be returned
	Read(types []string) mimedata.Mimes

	// Write writes given mimedata to the clipboard -- in general having a
	// text/plain representation of the data in addition to a more specific
	// format is a good idea for anything more complex than plain text -- if
	// data has > 1 element, it is all encoded as a multipart MIME text string
	Write(data mimedata.Mimes) error

	// Clear clears the clipboard
	Clear()
}

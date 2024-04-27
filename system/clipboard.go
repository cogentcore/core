// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"cogentcore.org/core/base/fileinfo/mimedata"
)

// Clipboard defines the methods for reading and writing data to
// the system clipboard, which use mimedata to represent the data.
// Due to limitations of Windows (and Linux to a lesser extent),
// a multipart MIME formatted string is used if there are multiple
// elements in the mimedata, with any binary data text-encoded using base64.
type Clipboard interface {

	// IsEmpty returns true if there is nothing on the clipboard to read.
	// Can be used for disabling a Paste menu.
	IsEmpty() bool

	// Read attempts to read data of the given MIME type(s), in preference
	// order, from the clipboard, returning mimedata.Mimes which can
	// potentially have multiple types / multiple items, etc. If the first type
	// listed is a text type, then text-based retrieval is assumed; always
	// put the most specific desired type first; anything else present will
	// be returned
	Read(types []string) mimedata.Mimes

	// Write writes given mimedata to the clipboard; in general having a
	// text/plain representation of the data in addition to a more specific
	// format is a good idea for anything more complex than plain text; if
	// data has > 1 element, it is all encoded as a multipart MIME text string
	Write(data mimedata.Mimes) error

	// Clear clears the clipboard.
	Clear()
}

// ClipboardBase is a basic implementation of [Clipboard] that does nothing.
type ClipboardBase struct{}

var _ Clipboard = &ClipboardBase{}

func (bb *ClipboardBase) IsEmpty() bool                      { return false }
func (bb *ClipboardBase) Read(types []string) mimedata.Mimes { return nil }
func (bb *ClipboardBase) Write(data mimedata.Mimes) error    { return nil }
func (bb *ClipboardBase) Clear()                             {}

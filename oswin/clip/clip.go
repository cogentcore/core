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

	// Avail returns whether there is currently data available on the
	// clipboard, and if so, what format it is, using standard mime format
	// strings (e.g., text/plain, text/html, text/xml, text/uri-list, image/*
	// (jpg, png, etc)
	Avail() mimedata.Mimes

	// AvailType returns whether there is currently data available on the
	// clipboard of the given MIME type -- see mimedata package for standard types
	AvailType(typ string) bool

	// Read reads the data from the clipboard if available, returning a
	// mimedata.Mimes which can potentially have multiple different types for
	// the same data -- this may be different than what was advertised by
	// Avail(), depending on delays, updates, etc -- returns nil if no data
	// avail
	Read() mimedata.Mimes

	// ReadType reads the data from the clipboard of the specified MIME type
	// -- returns nil if no data avail or that type was not available
	ReadType(typ string) *mimedata.Data

	// Write writes given mimedata to the clipboard -- in general having a
	// text/plain version in addition to a more specific format is a good idea
	Write(data mimedata.Data) error

	// Clear clears the clipboard
	Clear() error
}

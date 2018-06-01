// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clip defines the system clipboard for the GoGi GUI system.
package clip

// clip.Board interface defines the methods for reading and writing data to
// the system clipboard
type Board interface {

	// Avail returns whether there is currently data available on the
	// clipboard, and if so, what format it is, using standard mime format
	// strings (e.g., text/plain, text/html, text/xml, text/uri-list, image/*
	// (jpg, png, etc)
	Avail() (avail bool, fmt string)

	// Read reads the data from the clipboard, along with the actual format of
	// the data read (which may be different than what was advertised by
	// Avail(), depending on delays, updates, etc
	Read() (data []byte, fmt string, err error)

	// Write writes data to the clipboard in the given format
	Write(data []byte, fmt string) error

	// Clear clears the clipboard
	Clear() error
}

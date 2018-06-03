// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clip defines the system clipboard for the GoGi GUI system.
package clip

import "errors"

var ErrNotAvail = errors.New("clip.Read: data was not available for reading from clipboard (of specified format for ReadFmt)")

// clip.Board interface defines the methods for reading and writing data to
// the system clipboard -- uses standard mime format strings to identify the
// type of data, and generic byte slices to represent the data
type Board interface {

	// Avail returns whether there is currently data available on the
	// clipboard, and if so, what format it is, using standard mime format
	// strings (e.g., text/plain, text/html, text/xml, text/uri-list, image/*
	// (jpg, png, etc)
	Avail() (avail bool, fmt string)

	// AvailFmt returns whether there is currently data available on the
	// clipboard of the given format, specified using standard mime format
	// strings (e.g., text/plain, text/html, text/xml, text/uri-list, image/*
	// (jpg, png, etc)
	AvailFmt(fmt string) bool

	// Read reads the data from the clipboard, along with the actual format of
	// the data read (which may be different than what was advertised by
	// Avail(), depending on delays, updates, etc -- returns ErrNotAvail if
	// data was not available (most typical error)
	Read() (data []byte, fmt string, err error)

	// ReadFmt reads the data from the clipboard in the specified format --
	// returns ErrNotAvail if data was not available (in the given format) (most
	// typical error)
	ReadFmt(fmt string) (data []byte, err error)

	// Write writes data to the clipboard with the given format
	Write(data []byte, fmt string) error

	// Clear clears the clipboard
	Clear() error
}

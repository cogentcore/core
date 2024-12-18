// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metadata

import "os"

// SetName sets the "Name" standard key.
func SetName(obj any, name string) {
	SetTo(obj, "Name", name)
}

// Name returns the "Name" standard key value (empty if not set).
func Name(obj any) string {
	nm, _ := GetFrom[string](obj, "Name")
	return nm
}

// SetDoc sets the "Doc" standard key.
func SetDoc(obj any, doc string) {
	SetTo(obj, "Doc", doc)
}

// Doc returns the "Doc" standard key value (empty if not set).
func Doc(obj any) string {
	doc, _ := GetFrom[string](obj, "Doc")
	return doc
}

// SetFile sets the "File" standard key for *os.File.
func SetFile(obj any, file *os.File) {
	SetTo(obj, "File", file)
}

// File returns the "File" standard key value (nil if not set).
func File(obj any) *os.File {
	doc, _ := GetFrom[*os.File](obj, "File")
	return doc
}

// SetFilename sets the "Filename" standard key.
func SetFilename(obj any, file string) {
	SetTo(obj, "Filename", file)
}

// Filename returns the "Filename" standard key value (empty if not set).
func Filename(obj any) string {
	doc, _ := GetFrom[string](obj, "Filename")
	return doc
}

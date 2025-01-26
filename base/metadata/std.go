// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metadata

import "os"

// SetName sets the "Name" standard key.
func SetName(obj any, name string) {
	Set(obj, "Name", name)
}

// Name returns the "Name" standard key value (empty if not set).
func Name(obj any) string {
	nm, _ := Get[string](obj, "Name")
	return nm
}

// SetDoc sets the "Doc" standard key.
func SetDoc(obj any, doc string) {
	Set(obj, "Doc", doc)
}

// Doc returns the "Doc" standard key value (empty if not set).
func Doc(obj any) string {
	doc, _ := Get[string](obj, "Doc")
	return doc
}

// SetFile sets the "File" standard key for *os.File.
func SetFile(obj any, file *os.File) {
	Set(obj, "File", file)
}

// File returns the "File" standard key value (nil if not set).
func File(obj any) *os.File {
	doc, _ := Get[*os.File](obj, "File")
	return doc
}

// SetFilename sets the "Filename" standard key.
func SetFilename(obj any, file string) {
	Set(obj, "Filename", file)
}

// Filename returns the "Filename" standard key value (empty if not set).
func Filename(obj any) string {
	doc, _ := Get[string](obj, "Filename")
	return doc
}

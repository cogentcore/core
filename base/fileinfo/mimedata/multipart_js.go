// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package mimedata

// ToMultipart produces a MIME multipart representation of multiple data
// elements present in the stream -- this should be used in system.Clipboard
// whenever there are multiple elements to be pasted, because windows doesn't
// support multiple clip elements, and linux isn't very convenient either
func (mi Mimes) ToMultipart() []byte {
	return writeMultipart(mi)
}

// FromMultipart parses a MIME multipart representation of multiple data
// elements into corresponding mime data components
func FromMultipart(body []byte, boundary string) Mimes {
	return readMultipart(body, boundary)
}

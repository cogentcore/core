// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package labels

import (
	"cogentcore.org/core/base/reflectx"
)

// Labeler interface provides a GUI-appropriate label for an item,
// via a Label string method. See [ToLabel] and [ToLabeler].
type Labeler interface {

	// Label returns a GUI-appropriate label for item
	Label() string
}

// ToLabel returns the GUI-appropriate label for an item, using the Labeler
// interface if it is defined, and falling back on [reflectx.ToString] converter
// otherwise.
func ToLabel(v any) string {
	if lb, ok := v.(Labeler); ok {
		return lb.Label()
	}
	return reflectx.ToString(v)
}

// ToLabeler returns the Labeler label, true if it was defined, else "", false
func ToLabeler(v any) (string, bool) {
	if lb, ok := v.(Labeler); ok {
		return lb.Label(), true
	}
	return "", false
}

// SliceLabeler interface provides a GUI-appropriate label
// for a slice item, given an index into the slice.
type SliceLabeler interface {

	// ElemLabel returns a GUI-appropriate label for slice element at given index.
	ElemLabel(idx int) string
}

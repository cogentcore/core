// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"reflect"

	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Labeler interface provides a GUI-appropriate label for an item,
// via a Label() string method.
// Use ToLabel converter to attempt to use this interface and then fall
// back on Stringer via kit.ToString conversion function.
type Labeler interface {
	// Label returns a GUI-appropriate label for item
	Label() string
}

// ToLabel returns the gui-appropriate label for an item, using the Labeler
// interface if it is defined, and falling back on kit.ToString converter
// otherwise -- also contains label impls for basic interface types for which
// we cannot easily define the Labeler interface
func ToLabel(it interface{}) string {
	lbler, ok := it.(Labeler)
	if !ok {
		switch v := it.(type) {
		case reflect.Type:
			return v.Name()
		case ki.Ki:
			return v.Name()
		}
		return kit.ToString(it)
	}
	return lbler.Label()
}

// ToLabeler returns the Labeler label, true if it was defined, else "", false
func ToLabeler(it interface{}) (string, bool) {
	if lbler, ok := it.(Labeler); ok {
		return lbler.Label(), true
	}
	return "", false
}

// SliceLabeler interface provides a GUI-appropriate label
// for a slice item, given an index into the slice.
type SliceLabeler interface {
	// ElemLabel returns a GUI-appropriate label for slice element at given index
	ElemLabel(idx int) string
}

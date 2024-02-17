// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"reflect"

	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
)

// This file contains all the special-purpose interfaces
// beyond the basic Node interface

// Updater defines an interface for something that has an Update() method
// this will be called by GUI actions that update values of a type
// including struct, slice, and map views in giv
type Updater interface {
	// Update updates anything in this type that might depend on other state
	// which could have just been changed.  It is the responsibility of the
	// type to determine what might have changed, or just generically update
	// everything assuming anything could have changed.
	Update()
}

/////////////////////////////////////////////////////////////
//  Labeler

// Labeler interface provides a GUI-appropriate label for an item,
// via a Label() string method.
// Use ToLabel converter to attempt to use this interface and then fall
// back on Stringer via kit.ToString conversion function.
type Labeler interface {
	// Label returns a GUI-appropriate label for item
	Label() string
}

// ToLabel returns the gui-appropriate label for an item, using the Labeler
// interface if it is defined, and falling back on [laser.ToString] converter
// otherwise -- also contains label impls for basic interface types for which
// we cannot easily define the Labeler interface
func ToLabel(it any) string {
	lbler, ok := it.(Labeler)
	if !ok {
		switch v := it.(type) {
		case reflect.Type:
			return v.Name()
		case ki.Ki:
			return v.Name()
		}
		return laser.ToString(it)
	}
	return lbler.Label()
}

// ToLabeler returns the Labeler label, true if it was defined, else "", false
func ToLabeler(it any) (string, bool) {
	if lbler, ok := it.(Labeler); ok {
		return lbler.Label(), true
	}
	return "", false
}

// SliceLabeler interface provides a GUI-appropriate label
// for a slice item, given an index into the slice.
type SliceLabeler interface {
	// ElemLabel returns a GUI-appropriate label for slice element at given index
	ElemLabel(index int) string
}

// Toolbarer interface is for ConfigToolbar function access for those that define it
type Toolbarer interface {
	ConfigToolbar(tb *Toolbar)
}

// Validator is an interface for types to provide a Validate method
// that is used in giv to validate string Values using [TextField.Validator].
type Validator interface {
	// Validate returns an error if the value is invalid.
	Validate() error
}

// FieldValidator is an interface for types to provide a ValidateField method
// that is used in giv to validate string field Values using [TextField.Validator].
type FieldValidator interface {
	// ValidateField returns an error if the value of the given named field is invalid.
	ValidateField(field string) error
}

// ShouldShower is an interface determining when you should take a shower.
// Actually, it determines whether a named field should be displayed
// (in giv.StructView and giv.StructViewInline).
type ShouldShower interface {
	// ShouldShow returns whether the given named field should be displayed.
	ShouldShow(field string) bool
}

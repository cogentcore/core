// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

// This file contains all the special-purpose interfaces
// beyond the basic [Widget] interface.

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

// Toolbarer interface is for ConfigToolbar function access for those that define it
type Toolbarer interface {
	ConfigToolbar(c *Config)
}

// AppChooserer is for ConfigAppBar function access for those that define it
type AppChooserer interface {
	ConfigAppChooser(ch *Chooser)
}

// Validator is an interface for types to provide a Validate method
// that is used in views to validate string Values using [TextField.Validator].
type Validator interface {
	// Validate returns an error if the value is invalid.
	Validate() error
}

// FieldValidator is an interface for types to provide a ValidateField method
// that is used in views to validate string field Values using [TextField.Validator].
type FieldValidator interface {
	// ValidateField returns an error if the value of the given named field is invalid.
	ValidateField(field string) error
}

// ShouldShower is an interface determining when you should take a shower.
// Actually, it determines whether a named field should be displayed
// (in views.StructView and views.StructViewInline).
type ShouldShower interface {
	// ShouldShow returns whether the given named field should be displayed.
	ShouldShow(field string) bool
}

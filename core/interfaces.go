// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

// This file contains all the special-purpose interfaces
// beyond the basic [Widget] interface.

// ToolbarMaker is an interface that types can implement to make a toolbar plan.
// It is automatically used when making value view dialogs.
type ToolbarMaker interface {
	MakeToolbar(p *Plan)
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
// (in [views.Form]).
type ShouldShower interface {
	// ShouldShow returns whether the given named field should be displayed.
	ShouldShow(field string) bool
}

// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bools

// A Booler is a type that can return
// its value as a boolean value
type Booler interface {
	// Bool returns the boolean
	// representation of the value
	Bool() bool
}

// A BoolSetter is a Booler that can also
// set its value from a bool value
type BoolSetter interface {
	Booler
	// SetBool sets the value from the
	// boolean representation of the value
	SetBool(val bool)
}

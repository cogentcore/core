// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

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

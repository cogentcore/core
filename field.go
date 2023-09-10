// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

// Field represents a field in a struct
type Field struct {
	Name       string
	Comment    string
	Directives Directives
	// Type       Type  // note: this does not exist here!  we don't require comprehensive parsing of all types!
}

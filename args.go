// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

// Args represents an arg
type Arg struct {
	Name string

	// Comment has all of the comment info as one string
	// with directives removed.
	Comment string

	// Directives has the parsed comment directives
	Directives Directives
}

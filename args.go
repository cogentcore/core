// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import "goki.dev/ordmap"

// Args represents an arg
type Arg struct {
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives
}

// Args represents multiple args to a function/method
type Args ordmap.Map[string, *Arg]

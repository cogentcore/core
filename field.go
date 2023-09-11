// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import "goki.dev/ordmap"

// Field represents a field in a struct
type Field struct {
	Name string

	// Doc has all of the comment documentation
	// info as one string with directives removed.
	Doc string

	// Directives has the parsed comment directives
	Directives Directives
}

// Fields represents multiple fields in a struct
type Fields ordmap.Map[string, *Field]

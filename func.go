// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import "goki.dev/ordmap"

// Func represents a global function
type Func struct {
	Info
	Args ordmap.Map[string, *Arg]
}

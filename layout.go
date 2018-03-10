// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
// "math"
)

// size preferences -- a value of 0 indicates no preference
type SizePrefs2D struct {
	Min  Size2D `desc:"minimum size -- will not be less than this"`
	Pref Size2D `desc:"preferred size -- start here"`
	Max  Size2D `desc:"maximum size -- will not be greater than this"`
}

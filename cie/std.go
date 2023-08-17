// Copyright (c) 2021, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import "github.com/goki/mat32"

// WhiteD65 is the standard white color for midday sun, D65, in XYZ coordinates.
// Used as a standard reference illumination condition for most cases.
var WhiteD65 = mat32.Vec3{95.047, 100.0, 108.883}

// WhiteD50 is the standard white color used for printing industry, D50,
// in XYZ coordinates.
var WhiteD50 = mat32.Vec3{96.4212, 100.0, 82.5188}

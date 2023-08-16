// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// adapted from: https://github.com/material-foundation/material-color-utilities
// Copyright 2022 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License")

package cam16

import "github.com/goki/mat32"

// View represents viewing conditions under which a color is being perceived
// which greatly affects the subjective perception.  Defaults represent the
// standard defined such conditions, under which the CAM16 computations operate.
type View struct {
	// white point illumination
	WhitePoint mat32.Vec3 `desc:"white point illumination"`

	// [def: 11.725676537]
	AdaptingLuminance float32 `def:"11.725676537" desc:""`

	// [def: 50] background luminance
	BackgroundLStar float32 `def:"50" desc:"background luminance"`

	// [def: 2] surround luminance
	Surround float32 `def:"2" desc:"surround luminance"`
}

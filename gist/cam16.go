// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import "github.com/goki/mat32"

// The code in this file is heavily based on
// https://github.com/material-foundation/material-color-utilities
//
// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Cam16 is a color in the Cam16 color model.
type Cam16 struct {
	Hue, Chroma, J, Q, M, S, Jstar, Astar, Bstar float32
}

// Distance returns the distance from the color to the other color
// in the CAM16-UCS color space.
func (c Cam16) Distance(other Cam16) float32 {
	dj := c.Jstar - other.Jstar
	da := c.Astar - other.Astar
	db := c.Bstar - other.Bstar
	dePrime := mat32.Sqrt(dj*dj + da*da + db*db)
	de := 1.41 * mat32.Pow(dePrime, 0.63)
	return de
}

func (c *Cam16) FromColor(rgba Color) {

}

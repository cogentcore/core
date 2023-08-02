// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import "image/color"

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

// HCTA is a color specified by Hue, Chroma,
// Tone, and Transparency values. See
// https://material.io/blog/science-of-color-design
// for more information about the HCT color system.
type HCTA struct {
	H, C, T, A float32
}

var _ = color.Color(HCTA{})

// RGBA implements the [color.Color] interface
func (c HCTA) RGBA() (r, g, b, a uint32) {
	return
}

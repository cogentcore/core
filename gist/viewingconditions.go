// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import "github.com/goki/mat32"

// The code in this file is heavily based on
// https://github.com/material-foundation/material-color-utilities
//
// # Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// ViewingConditions contains the information about
// the context in which an [HCTA] color is viewed.
type ViewingConditions struct {
	WhitePoint            mat32.Vec3 `desc:"the coordinates of white in XYZ color space"`
	AdaptingLuminance     float32    `desc:"the light strength in lux"`
	BackgroundLstar       float32    `desc:"the average luminance of 10 degrees around the color"`
	Surround              float32    `desc:"the brightness of the entire environment"`
	DiscountingIlluminant bool       `desc:"whether the person's eyes have adjusted to the lighting"`

	BackgroundYToWhitePointY float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	AW                       float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	NBB                      float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	NCB                      float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	C                        float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	NC                       float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	DRGBInverse              mat32.Vec3 `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	RGBD                     mat32.Vec3 `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	FL                       float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	FLRoot                   float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
	Z                        float32    `desc:"this is an intermediate computed value and should not be modified by the end-user"`
}

// Init initializes the values of the viewing conditions
// based on what has already been set.
func (vc *ViewingConditions) Init() {
	if vc.AdaptingLuminance <= 0 {
		vc.AdaptingLuminance = 200 / mat32.Pi
	}
}

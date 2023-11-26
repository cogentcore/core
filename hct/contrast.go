// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from https://github.com/material-foundation/material-color-utilities
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

package hct

import (
	"goki.dev/cam/cie"
	"goki.dev/mat32/v2"
)

// ContrastRatio returns the contrast ratio between the given two tones.
// The contrast ratio will be between 1 and 21, and the tones should be
// between 0 and 100 and will be clamped to such.
func ContrastRatio(a, b float32) float32 {
	a = mat32.Clamp(a, 0, 100)
	b = mat32.Clamp(b, 0, 100)
	return RatioOfYs(cie.LToY(a), cie.LToY(b))
}

// RatioOfYs returns the contrast ratio of two XYZ Y values.
func RatioOfYs(a, b float32) float32 {
	lighter := max(a, b)
	darker := min(a, b)
	return (lighter + 5) / (darker + 5)
}

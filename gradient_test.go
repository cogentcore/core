// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

func ExampleLinearGradient() {
	LinearGradient().AddStop(White, 0, 1).AddStop(Black, 1, 1)
}

func ExampleRadialGradient() {
	RadialGradient().AddStop(Green, 0, 1).AddStop(Yellow, 0.5, 1).AddStop(Red, 1, 1)
}

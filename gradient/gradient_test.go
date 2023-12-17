// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gradient

import "goki.dev/colors"

func ExampleLinear() {
	NewLinear().AddStop(colors.White, 0).AddStop(colors.Black, 1)
}

func ExampleRadial() {
	NewRadial().AddStop(colors.Green, 0).AddStop(colors.Yellow, 0.5).AddStop(colors.Red, 1)
}

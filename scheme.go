// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import "goki.dev/colors/matcolor"

// Scheme is the main currently active global Material Design 3
// color scheme. It is the main way that end-user code should
// access the color scheme; ideally, almost all color values should
// be set to something in here. For more specific tones of colors,
// see [Palette].
var Scheme *matcolor.Scheme

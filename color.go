// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

import "github.com/muesli/termenv"

// UseColor is whether to use color in log messages.
// It is on by default.
var UseColor = true

var colorProfile = termenv.ColorProfile()

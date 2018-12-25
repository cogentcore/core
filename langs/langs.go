// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package langs includes all the supported languages for GoPi -- need to
// import this package to get those all included in a given target
package langs

import (
	_ "github.com/goki/pi/langs/golang"
	_ "github.com/goki/pi/langs/markdown"
	_ "github.com/goki/pi/langs/tex"
)

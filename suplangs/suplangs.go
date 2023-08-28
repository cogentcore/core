// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package suplangs includes all the supported languages for GoPi -- need to
// import this package to get those all included in a given target
package suplangs

import (
	_ "goki.dev/pi/v2/langs/golang"
	_ "goki.dev/pi/v2/langs/markdown"
	_ "goki.dev/pi/v2/langs/tex"
)

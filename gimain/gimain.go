// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gimain provides a Main function that encapsulates the standard
// oswin driver main function, and also ensures that standard sub-packages
// that are required for typical gi gui usage are automatically included
package gimain

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver"
	"github.com/goki/gi/svg"
)

// these dummy variables force inclusion of relevant packages
var dummyGi gi.Vec2D
var dummSvg svg.Line
var dummyVV giv.ValueViewBase

func Main(mainrun func()) {
	DebugEnumSizes()

	driver.Main(func(app oswin.App) {
		mainrun()
	})
}

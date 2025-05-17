// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package renderers

import (
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/renderers/htmlcanvas"
	"cogentcore.org/core/paint/renderers/rasterx"
	_ "cogentcore.org/core/text/shaped/shapers"
)

func init() {
	paint.NewSourceRenderer = htmlcanvas.New
	paint.NewImageRenderer = rasterx.New
}

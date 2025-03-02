// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package renderers

import (
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/renderers/htmlcanvas"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/shaped/shapedgt"
)

func init() {
	paint.NewSourceRenderer = htmlcanvas.New
	shaped.NewShaper = shapedgt.NewShaper // todo: update when new js avail
}

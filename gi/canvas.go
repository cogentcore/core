// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "goki.dev/goki/paint"

// Canvas is a widget that can be arbitrarily drawn to.
type Canvas struct {
	WidgetBase
}

// Draw draws to the canvas by calling the given function with its paint context.
func (c *Canvas) Draw(f func(pc *paint.Context)) {
	if c.PushBounds() {
		pc, _ := c.RenderLock()
		f(pc)
		c.RenderUnlock()
		c.PopBounds()
	}
}

// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// Box is a simple base [Widget] that renders the Std Box model
type Box struct {
	WidgetBase
}

func (bx *Box) OnInit() {
	bx.WidgetBase.OnInit()
}

// RenderBox does the standard box model rendering
func (bx *Box) RenderBox() {
	_, st := bx.RenderLock()
	bx.RenderStdBox(st)
	bx.RenderUnlock()
}

func (bx *Box) Render() {
	if bx.PushBounds() {
		bx.RenderBox()
		bx.RenderParts()
		bx.RenderChildren()
		bx.PopBounds()
	}
}

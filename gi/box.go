// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// Box is a simple base [Widget] that renders the standard box model.
type Box struct {
	WidgetBase
}

func (bx *Box) Render() {
	bx.RenderStandardBox()
}

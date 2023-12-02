// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

// Flow represents SVG flow* elements
type Flow struct {
	NodeBase
	FlowType string
}

func (g *Flow) SVGName() string { return "flow" }

func (g *Flow) CopyFieldsFrom(frm any) {
	fr := frm.(*Flow)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.FlowType = fr.FlowType
}

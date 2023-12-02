// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

// Filter represents SVG filter* elements
type Filter struct {
	NodeBase
	FilterType string
}

func (g *Filter) SVGName() string { return "filter" }

func (g *Filter) CopyFieldsFrom(frm any) {
	fr := frm.(*Filter)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.FilterType = fr.FilterType
}

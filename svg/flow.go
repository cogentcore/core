// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"goki.dev/gi/gi"
)

// Flow represents SVG flow* elements
type Flow struct {
	NodeBase
	FlowType string
}

var TypeFlow = kit.Types.AddType(&Flow{}, ki.Props{ki.EnumTypeFlag: gi.TypeNodeFlags})

// AddNewFlow adds a new flow to given parent node, with given name.
func AddNewFlow(parent ki.Ki, name string) *Flow {
	return parent.AddNewChild(TypeFlow, name).(*Flow)
}

func (g *Flow) SVGName() string { return "flow" }

func (g *Flow) CopyFieldsFrom(frm any) {
	fr := frm.(*Flow)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.FlowType = fr.FlowType
}

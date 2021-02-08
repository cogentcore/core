// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Flow represents SVG flow* elements
type Flow struct {
	NodeBase
	FlowType string
}

var KiT_Flow = kit.Types.AddType(&Flow{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewFlow adds a new flow to given parent node, with given name.
func AddNewFlow(parent ki.Ki, name string) *Flow {
	return parent.AddNewChild(KiT_Flow, name).(*Flow)
}

func (g *Flow) SVGName() string { return "flow" }

func (g *Flow) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Flow)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.FlowType = fr.FlowType
}

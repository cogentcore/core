// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

/////////////////////////////////////////////////////////////////////////////
//  Gradient

// Gradient is used for holding a specified color gradient (ColorSpec)
// name is id for lookup in url
type Gradient struct {
	Node2DBase
	Grad      gist.ColorSpec `desc:"the color gradient"`
	StopsName string         `desc:"name of another gradient to get stops from"`
	RefCount  int            `view:"-" desc:"number of objects referring to this gradient"`
}

var KiT_Gradient = kit.Types.AddType(&Gradient{}, nil)

// AddNewGradient adds a new gradient to given parent node, with given name.
func AddNewGradient(parent ki.Ki, name string) *Gradient {
	return parent.AddNewChild(KiT_Gradient, name).(*Gradient)
}

func (gr *Gradient) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Gradient)
	gr.Node2DBase.CopyFieldsFrom(&fr.Node2DBase)
	gr.Grad = fr.Grad
	gr.StopsName = fr.StopsName
}

// UpdateStops copies stops from StopsName gradient if it is set
func (gr *Gradient) UpdateStops() {
	if gr.StopsName == "" {
		return
	}
	vp := gr.ViewportSafe()
	csn := vp.FindNamedElement(gr.StopsName)
	if grad, ok := csn.(*Gradient); ok {
		gr.Grad.CopyStopsFrom(&grad.Grad)
	}
}

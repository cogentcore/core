// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/girl/gist"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
)

/////////////////////////////////////////////////////////////////////////////
//  Gradient

// Gradient is used for holding a specified color gradient (ColorSpec)
// name is id for lookup in url
type Gradient struct {
	Node2DBase

	// the color gradient
	Grad gist.ColorSpec `desc:"the color gradient"`

	// name of another gradient to get stops from
	StopsName string `desc:"name of another gradient to get stops from"`
}

var TypeGradient = kit.Types.AddType(&Gradient{}, nil)

// AddNewGradient adds a new gradient to given parent node, with given name.
func AddNewGradient(parent ki.Ki, name string) *Gradient {
	return parent.AddNewChild(TypeGradient, name).(*Gradient)
}

func (gr *Gradient) CopyFieldsFrom(frm any) {
	fr := frm.(*Gradient)
	gr.Node2DBase.CopyFieldsFrom(&fr.Node2DBase)
	gr.Grad = fr.Grad
	gr.StopsName = fr.StopsName
}

// GradientType returns the SVG-style type name of gradient: linearGradient or radialGradient
func (gr *Gradient) GradientType() string {
	if gr.Grad.Gradient != nil && gr.Grad.Gradient.IsRadial {
		return "radialGradient"
	}
	return "linearGradient"
}

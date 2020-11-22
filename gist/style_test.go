// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"fmt"
	// "reflect"
	"testing"

	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
)

func TestStyle(t *testing.T) {
	props := make(ki.Props)
	props["color"] = "red"
	props["width"] = "24.7em"
	props["box-shadow.h-offset"] = "10px"
	props["box-shadow.v-offset"] = "initial"
	props["border-style"] = "groove"
	props["border-width"] = "2px"
	props["height"] = "inherit"
	var s, p, d Style
	s.Defaults()
	p.Defaults()
	d.Defaults()
	p.Layout.Height = units.NewValue(42.0, units.In)
	s.BoxShadow.VOffset = units.NewValue(22.0, units.Pc)
	s.SetStyleProps(&p, props, nil)

	fmt.Printf("style width: %v\n", s.Layout.Width)
	fmt.Printf("style height: %v\n", s.Layout.Height)
	fmt.Printf("style box-shadow.h-offset: %v\n", s.BoxShadow.HOffset)
	fmt.Printf("style box-shadow.v-offset: %v\n", s.BoxShadow.VOffset)
	fmt.Printf("style border-style: %v\n", s.Border.Style)
}

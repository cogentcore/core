// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image/color"
	"testing"

	"goki.dev/colors"
	"goki.dev/girl/units"
)

// "reflect"

type TestContext struct {
}

func (tc *TestContext) Base() color.RGBA {
	return colors.White
}

func (tc *TestContext) FullByURL(url string) *colors.Full {
	return nil
}

func TestStyle(t *testing.T) {
	st := &Style{}
	par := &Style{}
	ctx := &TestContext{}
	par.Font.Opacity = .34
	props := map[string]any{"opacity": float64(0.25)}
	st.StyleFromProps(par, props, ctx)
	if st.Font.Opacity != 0.25 {
		t.Errorf("opacity != 0.25: %g", st.Font.Opacity)
	}
	st.Font.Opacity = 0

	props = map[string]any{"opacity": float32(0.25)}
	st.StyleFromProps(par, props, ctx)
	if st.Font.Opacity != 0.25 {
		t.Errorf("opacity != 0.25: %g", st.Font.Opacity)
	}
	st.Font.Opacity = 0

	props = map[string]any{"opacity": 2}
	st.StyleFromProps(par, props, ctx)
	if st.Font.Opacity != 2 {
		t.Errorf("opacity != 2: %g", st.Font.Opacity)
	}
	st.Font.Opacity = 0

	props = map[string]any{"opacity": "0.25"}
	st.StyleFromProps(par, props, ctx)
	if st.Font.Opacity != 0.25 {
		t.Errorf("opacity != 2: %g", st.Font.Opacity)
	}
	st.Font.Opacity = 0

	props = map[string]any{"horizontal-align": AlignFlexEnd}
	st.StyleFromProps(par, props, ctx)
	if st.AlignH != AlignFlexEnd {
		t.Errorf("horizontal-align != AlignFlexEnd: %v", st.AlignH)
	}
	st.AlignH = AlignLeft

	props = map[string]any{"horizontal-align": "AlignFlexEnd"}
	st.StyleFromProps(par, props, ctx)
	if st.AlignH != AlignFlexEnd {
		t.Errorf("horizontal-align != AlignFlexEnd: %v", st.AlignH)
	}
	st.AlignH = AlignLeft

	props = map[string]any{"horizontal-align": 1}
	st.StyleFromProps(par, props, ctx)
	if st.AlignH != AlignTop {
		t.Errorf("horizontal-align != AlignTop: %v", st.AlignH)
	}
	st.AlignH = AlignLeft

	props = map[string]any{"x": "12px"}
	st.StyleFromProps(par, props, ctx)
	if st.PosX.Val != 12 || st.PosX.Un != units.UnitPx {
		t.Errorf("posx != 12px: %v", st.PosX)
	}
	st.PosX = units.Ex(0)

	props = map[string]any{"x": units.Px(12)}
	st.StyleFromProps(par, props, ctx)
	if st.PosX.Val != 12 || st.PosX.Un != units.UnitPx {
		t.Errorf("posx != 12px: %v", st.PosX)
	}
	st.PosX = units.Ex(0)

}

// func TestStyle(t *testing.T) {
// 	props := make(map[string]any)
// 	props["color"] = "red"
// 	props["width"] = "24.7em"
// 	props["box-shadow.h-offset"] = "10px"
// 	props["box-shadow.v-offset"] = "initial"
// 	props["border-style"] = "groove"
// 	props["border-width"] = "2px"
// 	props["height"] = "inherit"
// 	var s, p, d Style
// 	s.Defaults()
// 	p.Defaults()
// 	d.Defaults()
// 	p.Height = units.In(42)
// 	// s.BoxShadow.VOffset = units.NewValue(22.0, units.UnitPc)
// 	s.SetStyleProps(&p, props, nil)

// 	fmt.Printf("style width: %v\n", s.Width)
// 	fmt.Printf("style height: %v\n", s.Height)
// 	// fmt.Printf("style box-shadow.h-offset: %v\n", s.BoxShadow.HOffset)
// 	// fmt.Printf("style box-shadow.v-offset: %v\n", s.BoxShadow.VOffset)
// 	fmt.Printf("style border-style: %v\n", s.Border.Style)
// }

// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

func LayoutTestFile(t *testing.T) string {
	p := filepath.Join("testdata", "layout")
	tnm := strcase.ToSnake(strings.TrimPrefix(t.Name(), "TestLayout"))
	n := filepath.Join("layout", tnm)
	errors.Log(os.MkdirAll(p, 0750))
	return n
}

func LayoutTestDir(t *testing.T) string {
	tnm := strcase.ToSnake(strings.TrimPrefix(t.Name(), "TestLayout"))
	n := filepath.Join("layout", tnm)
	p := filepath.Join("testdata", n)
	errors.Log(os.MkdirAll(p, 0750))
	return n
}

func TestLayoutFramesAlignItems(t *testing.T) {
	wraps := []bool{false, true}
	dirs := []styles.Directions{styles.Row, styles.Column}
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	for _, wrap := range wraps {
		for _, dir := range dirs {
			for _, align := range aligns {
				tnm := fmt.Sprintf("wrap_%v_dir_%v_align_%v", wrap, dir, align)
				b := NewBody()
				b.Styler(func(s *styles.Style) {
					s.Overflow.Set(styles.OverflowVisible)
					s.Direction = dir
					s.Wrap = wrap
					s.Align.Items = align
				})
				PlainFrames(b, math32.Vec2(0, 0))
				b.AssertRender(t, tdir+tnm)
			}
		}
	}
}

func TestLayoutFramesAlignContent(t *testing.T) {
	wraps := []bool{false, true}
	dirs := []styles.Directions{styles.Row, styles.Column}
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	for _, wrap := range wraps {
		for _, dir := range dirs {
			for _, align := range aligns {
				tnm := fmt.Sprintf("wrap-%v-dir-%v-align-%v", wrap, dir, align)
				b := NewBody()
				b.Styler(func(s *styles.Style) {
					if dir == styles.Row {
						s.Min.Y.Px(300)
					} else {
						s.Min.X.Px(300)
					}
					s.Overflow.Set(styles.OverflowVisible)
					s.Direction = dir
					s.Wrap = wrap
					s.Align.Content = align
				})
				PlainFrames(b, math32.Vec2(0, 0))
				b.AssertRender(t, tdir+tnm)
			}
		}
	}
}

func TestLayoutFramesJustifyContent(t *testing.T) {
	wraps := []bool{false, true}
	dirs := []styles.Directions{styles.Row, styles.Column}
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	for _, wrap := range wraps {
		dsz := float32(600)
		if wrap {
			dsz = 400
		}
		for _, dir := range dirs {
			for _, align := range aligns {
				tnm := fmt.Sprintf("wrap_%v_dir_%v_align_%v", wrap, dir, align)
				b := NewBody()
				b.Styler(func(s *styles.Style) {
					if dir == styles.Row {
						s.Min.X.Px(dsz)
					} else {
						s.Min.Y.Px(dsz)
					}
					s.Overflow.Set(styles.OverflowVisible)
					s.Direction = dir
					s.Wrap = wrap
					s.Justify.Content = align
				})
				PlainFrames(b, math32.Vec2(0, 0))
				b.AssertRender(t, tdir+tnm)
			}
		}
	}
}

func TestLayoutFramesJustifyItems(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	// dsz := float32(600)
	for _, align := range aligns {
		tnm := fmt.Sprintf("align_%v", align)
		b := NewBody()
		b.Styler(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Justify.Items = align
		})
		PlainFrames(b, math32.Vec2(0, 0))
		b.AssertRender(t, tdir+tnm)
	}
}

func TestLayoutFramesJustifySelf(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	// dsz := float32(600)
	for ai, align := range aligns {
		tnm := fmt.Sprintf("align_%v", align)
		b := NewBody()
		b.Styler(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Justify.Items = align
		})
		PlainFrames(b, math32.Vec2(0, 0))
		b.Child(2).(Widget).AsWidget().Styler(func(s *styles.Style) {
			s.Justify.Self = aligns[(ai+1)%len(aligns)]
		})
		b.AssertRender(t, tdir+tnm)
	}
}

func TestLayoutFramesAlignSelf(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	// dsz := float32(600)
	for ai, align := range aligns {
		tnm := fmt.Sprintf("align_%v", align)
		b := NewBody()
		b.Styler(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Align.Items = align
		})
		PlainFrames(b, math32.Vec2(0, 0))
		b.Child(2).(Widget).AsWidget().Styler(func(s *styles.Style) {
			s.Align.Self = aligns[(ai+1)%len(aligns)]
		})
		b.AssertRender(t, tdir+tnm)
	}
}

/*

	case "frames-horiz":
		row := HorizRow(sc)
		row.Styler(func(s *styles.Style) {
			// s.Align.X = styles.End
			s.Wrap = true
		})
		PlainFrames(row, math32.Vec2(0, 0))
		// NewText(sc).SetText(ShortText).Styler(func(s *styles.Style) {
		// })
		HorizRow(sc).Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
	case "text-align":
		// 	row := HorizRow(sc)
		sc.Styler(func(s *styles.Style) {
			s.Align.X = styles.Center
		})
		NewText(sc).SetText(AlignText).Styler(func(s *styles.Style) {
			s.Align.X = styles.Center
			s.Text.Align = styles.Center
		})
	case "long-text-wrap": // just text
		WrapText(sc, VeryLongText)
	case "long-text-wrap-box": // text in box -- failing to adjust to full height
		row := HorizRow(sc)
		text := WrapText(row, VeryLongText)
		row.Styler(func(s *styles.Style) {
			// s.Align.X = styles.End
		})
		text.Styler(func(s *styles.Style) {
			s.Align.X = styles.Center
		})
		fr := BoxFrame(sc) // this takes up slack
		sm := WrapText(fr, ShortText)
		_ = sm
	case "long-text-wrap-max-box": // text in constrained box
		row := HorizRow(sc)
		text := WrapText(row, VeryLongText) // VeryLongText)
		row.Styler(func(s *styles.Style) {
			// s.Align.X = styles.End
			s.Max.X.Ch(100) // todo: this is *sometimes* failing to constrain..
			// s.Overflow.X = styles.OverflowAuto
		})
		text.Styler(func(s *styles.Style) {
			s.Text.Align = styles.Center
		})
		// fr := BoxFrame(sc) // this takes up slack
		// sm := WrapText(fr, ShortText)
		// _ = sm
	case "scroll-absorb": // Auto scroll should absorb extra size
		row := HorizRow(sc, "row")
		f1, sp := SpaceFrame(row)
		f1.Styler(func(s *styles.Style) {
			s.Overflow.Y = styles.OverflowAuto // this should absorb the size
		})
		sp.Styler(func(s *styles.Style) {
			s.Min.Y.Em(100)
		})
		BoxFrame(row).Styler(func(s *styles.Style) {
			s.Min.Y.Em(20) // fix size
			s.Max.Y.Em(20) // fix size
		})
	case "scroll-absorb-splits": // Auto scroll should absorb extra size
		sp, f1, f2 := Splits2(sc)
		_ = sp
		f1.Styler(func(s *styles.Style) {
			s.Overflow.Y = styles.OverflowAuto // this should absorb the size
		})
		NewSpace(f1).Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Min.Y.Em(100)
		})
		f2.Styler(func(s *styles.Style) {
			s.Grow.Set(0, 0)
		})
		NewSpace(f2).Styler(func(s *styles.Style) {
			s.Min.X.Ch(20)
			s.Min.Y.Em(20)
		})
	case "tabs-stack": // recreates the issue with demo tabs
		// does not grow -- stacked not doing the right thing
		tab, tfr := TabFrame(sc)
		_ = tab
		parent := tfr // or sc
		row := HorizRow(par)

		sp := NewSpace(row)
		_ = sp
		WrapText(par, LongText)
		fr, sp2 := SpaceFrame(par)
		_ = fr
		_ = sp2
		fr.Styler(func(s *styles.Style) {
			s.Grow.Set(0, 1)
			s.Min.X.Em(20)
			s.Min.Y.Em(10)
		})
	case "splits": // splits
		sp, f1, f2 := Splits2(sc)
		_ = f1
		_ = f2
		sp.SetSplits(.3, .7)
	case "textfield-parts": // textfield parts alignment
		NewTextField(sc).AddClearButton()
	case "switch":
		NewSwitch(sc)
	case "button":
		NewButton(sc).SetText("Test")
	case "small-round-button":
		bt := NewButton(sc).SetType(ButtonAction).SetText("22").Styler(func(s *styles.Style) {
			s.Min.X.Dp(40)
			s.Min.Y.Dp(40)
			s.Padding.Zero()
			s.BackgroundColor.SetSolid(colors.Scheme.Primary.Base)
			s.Color = colors.C(colors.Scheme.Primary.On)
		})
		bt.Config(sc)
		bt.Parts.Styler(func(s *styles.Style) {
			s.Text.Align = styles.Center
			s.Text.AlignV = styles.Center
			s.Align.Set(styles.Center)
			s.Padding.Zero()
			s.Margin.Zero()
		})
	case "form": // form
		ts := &Test{}
		views.NewForm(sc).SetStruct(ts)
	case "timepicker": // time picker
		ts := &TestTime{}
		ts.Date = time.Now()
		views.NewForm(sc).SetStruct(ts)
	case "center-dialog":
		d := NewBody(sc).FullWindow(true)
		d.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Align.Set(styles.Center)
		})
		fr := NewFrame(d).Styler(func(s *styles.Style) { // note: this is critical for separating from topbar
			s.Direction = styles.Column
			s.Grow.Set(1, 1)
			s.Align.Set(styles.Center)
		})
		NewText(fr).SetType(LabelDisplayMedium).SetText("Event recorded!").
			Style(func(s *styles.Style) {
				s.Align.Set(styles.Center)
			})
		NewText(fr).SetType(LabelBodyLarge).
			SetText("Thank you for reporting your issue!").
			Style(func(s *styles.Style) {
				s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
				s.Align.Set(styles.Center)
			})
		NewButton(fr).SetType(ButtonTonal).SetText("Return home").
			Style(func(s *styles.Style) {
				s.Align.Set(styles.Center)
			})
		NewButton(sc).SetText("Click Me").OnClick(func(e events.Event) {
			d.Run()
		})
	default:
		fmt.Println("error: case didn't match:", doCase)
	}

	NewWindow(sc).Run().Wait()
}
*/

func BoxFrame(parent Widget) *Frame {
	fr := NewFrame(parent)
	fr.Styler(func(s *styles.Style) {
		s.Border.Color.Set(colors.C(colors.Scheme.Outline))
		s.Border.Width.Set(units.Dp(2))
	})
	return fr
}

func SpaceFrame(parent Widget) (*Frame, *Space) {
	fr := NewFrame(parent)
	fr.Styler(func(s *styles.Style) {
		s.Border.Color.Set(colors.C(colors.Scheme.Outline))
		s.Border.Width.Set(units.Dp(2))
	})
	sp := NewSpace(fr)
	return fr, sp
}

func HorizontalRow(parent Widget) *Frame {
	row := BoxFrame(parent)
	row.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	return row
}

func Splits2(parent Widget) (*Splits, *Frame, *Frame) {
	sp := NewSplits(parent)
	f1 := BoxFrame(sp)
	f2 := BoxFrame(sp)
	return sp, f1, f2
}

func TabFrame(parent Widget) (*Frame, *Frame) {
	tab := BoxFrame(parent)
	tab.Styler(func(s *styles.Style) {
		s.Display = styles.Stacked
		tab.StackTop = 0
	})
	tfr := BoxFrame(tab)
	tfr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	return tab, tfr
}

func WrapText(parent Widget, txt string) *Text {
	return NewText(parent).SetText(txt)
}

func PlainFrames(parent Widget, grow math32.Vector2) {
	for _, sz := range FrameSizes {
		fr := BoxFrame(parent)
		fr.Styler(func(s *styles.Style) {
			s.Min.X.Px(sz.X)
			s.Min.Y.Px(sz.Y)
			s.Grow = grow
		})
	}
}

type Wide struct {
	Name  string
	Title string
	F2    string
	F3    string
}

type Test struct {
	Wide Wide `view:"inline"`
	Vec  math32.Vector2
}

type TestTime struct {
	Date time.Time
}

var (
	ShortText = "This is a test of layout."

	LongText = "This is a test of the layout logic, which is pretty complex and requires some experimenting to understand how it all works.  The styling and behavior is the same as the CSS / HTML Flex model, except we only support Grow, not Shrink. "

	VeryLongText = LongText + LongText + LongText

	AlignText = "This is text to test for text align<br>This line is short<br>This is text to test for text align, this one is longer"

	FrameSizes = [5]math32.Vector2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}
)

func TestLayoutScrollLabel(t *testing.T) {
	// TODO(#808)
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Max.Set(units.Dp(50))
	})
	fr := NewFrame(b).Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
	})
	NewText(fr).SetText(LongText)
	b.AssertRender(t, "layout/scroll/label")
}

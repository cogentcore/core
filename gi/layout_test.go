// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iancoleman/strcase"
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/grr"
	"goki.dev/mat32/v2"
)

func LayoutTestFile(t *testing.T) string {
	p := filepath.Join("testdata", "layout")
	tnm := strcase.ToSnake(strings.TrimPrefix(t.Name(), "TestLayout"))
	n := filepath.Join("layout", tnm)
	grr.Log(os.MkdirAll(p, 0750))
	return n
}

func LayoutTestDir(t *testing.T) string {
	tnm := strcase.ToSnake(strings.TrimPrefix(t.Name(), "TestLayout"))
	n := filepath.Join("layout", tnm)
	p := filepath.Join("testdata", n)
	grr.Log(os.MkdirAll(p, 0750))
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
				sc := NewScene()
				sc.Style(func(s *styles.Style) {
					s.Overflow.Set(styles.OverflowVisible)
					s.Direction = dir
					s.Wrap = wrap
					s.Align.Items = align
				})
				PlainFrames(sc, mat32.Vec2{0, 0})
				sc.AssertPixelsOnShow(t, filepath.Join(tdir, tnm))
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
		wrap := wrap
		for _, dir := range dirs {
			dir := dir
			for _, align := range aligns {
				align := align
				tnm := fmt.Sprintf("wrap_%v_dir_%v_align_%v", wrap, dir, align)
				sc := NewScene()
				sc.Style(func(s *styles.Style) {
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
				PlainFrames(sc, mat32.Vec2{0, 0})
				sc.AssertPixelsOnShow(t, filepath.Join(tdir, tnm))
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
		wrap := wrap
		dsz := float32(600)
		if wrap {
			dsz = 400
		}
		for _, dir := range dirs {
			dir := dir
			for _, align := range aligns {
				align := align
				tnm := fmt.Sprintf("wrap_%v_dir_%v_align_%v", wrap, dir, align)
				sc := NewScene()
				sc.Style(func(s *styles.Style) {
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
				PlainFrames(sc, mat32.Vec2{0, 0})
				sc.AssertPixelsOnShow(t, filepath.Join(tdir, tnm))
			}
		}
	}
}

func TestLayoutFramesJustifyItems(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	// dsz := float32(600)
	for _, align := range aligns {
		align := align
		tnm := fmt.Sprintf("align_%v", align)
		sc := NewScene()
		sc.Style(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Justify.Items = align
		})
		PlainFrames(sc, mat32.Vec2{0, 0})
		sc.AssertPixelsOnShow(t, filepath.Join(tdir, tnm))
	}
}

func TestLayoutFramesJustifySelf(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	// dsz := float32(600)
	for ai, align := range aligns {
		ai := ai
		align := align
		tnm := fmt.Sprintf("align_%v", align)
		sc := NewScene()
		sc.Style(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Justify.Items = align
		})
		PlainFrames(sc, mat32.Vec2{0, 0})
		_, fr2 := AsWidget(sc.ChildByName("fr2"))
		fr2.Style(func(s *styles.Style) {
			s.Justify.Self = aligns[(ai+1)%len(aligns)]
		})
		sc.AssertPixelsOnShow(t, filepath.Join(tdir, tnm))
	}
}

func TestLayoutFramesAlignSelf(t *testing.T) {
	aligns := []styles.Aligns{styles.Start, styles.Center, styles.End}
	tdir := LayoutTestDir(t)
	// dsz := float32(600)
	for ai, align := range aligns {
		ai := ai
		align := align
		tnm := fmt.Sprintf("align_%v", align)
		sc := NewScene()
		sc.Style(func(s *styles.Style) {
			s.Overflow.Set(styles.OverflowVisible)
			s.Display = styles.Grid
			s.Columns = 2
			s.Align.Items = align
		})
		PlainFrames(sc, mat32.Vec2{0, 0})
		_, fr2 := AsWidget(sc.ChildByName("fr2"))
		fr2.Style(func(s *styles.Style) {
			s.Align.Self = aligns[(ai+1)%len(aligns)]
		})
		sc.AssertPixelsOnShow(t, filepath.Join(tdir, tnm))
	}
}

/*

	case "frames-horiz":
		row := HorizRow(sc)
		row.Style(func(s *styles.Style) {
			// s.Align.X = styles.End
			s.Wrap = true
		})
		PlainFrames(row, mat32.Vec2{0, 0})
		// NewLabel(sc, "lbl").SetText(ShortText).Style(func(s *styles.Style) {
		// })
		HorizRow(sc).Style(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
	case "text-align":
		// 	row := HorizRow(sc)
		sc.Style(func(s *styles.Style) {
			s.Align.X = styles.Center
		})
		NewLabel(sc, "lbl").SetText(AlignText).Style(func(s *styles.Style) {
			s.Align.X = styles.Center
			s.Text.Align = styles.Center
		})
	case "long-text-wrap": // just text
		WrapText(sc, VeryLongText)
	case "long-text-wrap-box": // text in box -- failing to adjust to full height
		row := HorizRow(sc)
		lbl := WrapText(row, VeryLongText)
		row.Style(func(s *styles.Style) {
			// s.Align.X = styles.End
		})
		lbl.Style(func(s *styles.Style) {
			s.Align.X = styles.Center
		})
		fr := BoxFrame(sc) // this takes up slack
		sm := WrapText(fr, ShortText)
		_ = sm
	case "long-text-wrap-max-box": // text in constrained box
		row := HorizRow(sc)
		lbl := WrapText(row, VeryLongText) // VeryLongText)
		row.Style(func(s *styles.Style) {
			// s.Align.X = styles.End
			s.Max.X.Ch(100) // todo: this is *sometimes* failing to constrain..
			// s.Overflow.X = styles.OverflowAuto
		})
		lbl.Style(func(s *styles.Style) {
			s.Text.Align = styles.Center
		})
		// fr := BoxFrame(sc) // this takes up slack
		// sm := WrapText(fr, ShortText)
		// _ = sm
	case "scroll-absorb": // Auto scroll should absorb extra size
		row := HorizRow(sc, "row")
		f1, sp := SpaceFrame(row)
		f1.Style(func(s *styles.Style) {
			s.Overflow.Y = styles.OverflowAuto // this should absorb the size
		})
		sp.Style(func(s *styles.Style) {
			s.Min.Y.Em(100)
		})
		BoxFrame(row).Style(func(s *styles.Style) {
			s.Min.Y.Em(20) // fix size
			s.Max.Y.Em(20) // fix size
		})
	case "scroll-absorb-splits": // Auto scroll should absorb extra size
		sp, f1, f2 := Splits2(sc)
		_ = sp
		f1.Style(func(s *styles.Style) {
			s.Overflow.Y = styles.OverflowAuto // this should absorb the size
		})
		NewSpace(f1).Style(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Min.Y.Em(100)
		})
		f2.Style(func(s *styles.Style) {
			s.Grow.Set(0, 0)
		})
		NewSpace(f2).Style(func(s *styles.Style) {
			s.Min.X.Ch(20)
			s.Min.Y.Em(20)
		})
	case "tabs-stack": // recreates the issue with demo tabs
		// does not grow -- stacked not doing the right thing
		tab, tfr := TabFrame(sc)
		_ = tab
		par := tfr // or sc
		row := HorizRow(par)

		sp := NewSpace(row)
		_ = sp
		WrapText(par, LongText)
		fr, sp2 := SpaceFrame(par)
		_ = fr
		_ = sp2
		fr.Style(func(s *styles.Style) {
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
		bt := NewButton(sc).SetType(ButtonAction).SetText("22").Style(func(s *styles.Style) {
			s.Min.X.Dp(40)
			s.Min.Y.Dp(40)
			s.Padding.Zero()
			s.BackgroundColor.SetSolid(colors.Scheme.Primary.Base)
			s.Color = colors.Scheme.Primary.On
		})
		bt.Config(sc)
		bt.Parts.Style(func(s *styles.Style) {
			s.Text.Align = styles.Center
			s.Text.AlignV = styles.Center
			s.Align.Set(styles.Center)
			s.Padding.Zero()
			s.Margin.Zero()
		})
	case "structview": // structview
		ts := &Test{}
		giv.NewStructView(sc).SetStruct(ts)
	case "timeview": // time view
		ts := &TestTime{}
		ts.Date = time.Now()
		giv.NewStructView(sc).SetStruct(ts)
	case "center-dialog":
		d := NewBody(sc).FullWindow(true)
		d.Style(func(s *styles.Style) {
			s.Grow.Set(1, 1)
			s.Align.Set(styles.Center)
		})
		fr := NewFrame(d).Style(func(s *styles.Style) { // note: this is critical for separating from topbar
			s.Direction = styles.Column
			s.Grow.Set(1, 1)
			s.Align.Set(styles.Center)
		})
		NewLabel(fr).SetType(LabelDisplayMedium).SetText("Event recorded!").
			Style(func(s *styles.Style) {
				s.Align.Set(styles.Center)
			})
		NewLabel(fr).SetType(LabelBodyLarge).
			SetText("Thank you for reporting your issue!").
			Style(func(s *styles.Style) {
				s.Color = colors.Scheme.OnSurfaceVariant
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

func BoxFrame(par Widget, nm ...string) *Frame {
	fr := NewFrame(par, nm...)
	fr.Style(func(s *styles.Style) {
		s.Border.Color.Set(colors.Black)
		s.Border.Width.Set(units.Dp(2))
	})
	return fr
}

func SpaceFrame(par Widget, nm ...string) (*Frame, *Space) {
	fr := NewFrame(par, nm...)
	fr.Style(func(s *styles.Style) {
		s.Border.Color.Set(colors.Black)
		s.Border.Width.Set(units.Dp(2))
	})
	sp := NewSpace(fr)
	return fr, sp
}

func HorizRow(par Widget, nm ...string) *Frame {
	row := BoxFrame(par, nm...)
	row.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	return row
}

func Splits2(par Widget) (*Splits, *Frame, *Frame) {
	sp := NewSplits(par)
	f1 := BoxFrame(sp)
	f2 := BoxFrame(sp)
	return sp, f1, f2
}

func TabFrame(par Widget) (*Frame, *Frame) {
	tab := BoxFrame(par)
	tab.Style(func(s *styles.Style) {
		s.Display = styles.Stacked
		tab.StackTop = 0
	})
	tfr := BoxFrame(tab)
	tfr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	return tab, tfr
}

func WrapText(par Widget, txt string) *Label {
	lbl := NewLabel(par, "wrap-text").SetText(txt)
	return lbl
}

func PlainFrames(par Widget, grow mat32.Vec2) {
	for i, sz := range FrameSizes {
		i := i
		sz := sz
		nm := fmt.Sprintf("fr%v", i)
		fr := BoxFrame(par, nm)
		fr.Style(func(s *styles.Style) {
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
	Vec  mat32.Vec2
}

type TestTime struct {
	Date time.Time
}

var (
	ShortText = "This is a test of layout."

	LongText = "This is a test of the layout logic, which is pretty complex and requires some experimenting to understand how it all works.  The styling and behavior is the same as the CSS / HTML Flex model, except we only support Grow, not Shrink. "

	VeryLongText = LongText + LongText + LongText

	AlignText = "This is text to test for text align<br>This line is short<br>This is text to test for text align, this one is longer"

	FrameSizes = [5]mat32.Vec2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}
)

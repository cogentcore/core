// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/stretchr/testify/assert"
)

func TestStyleStates(t *testing.T) {
	s := NewStyle()
	s.SetState(true, states.Hovered, states.Focused)
	assert.True(t, s.Is(states.Hovered))
	assert.True(t, s.Is(states.Focused))
	assert.False(t, s.IsReadOnly())
	s.SetEnabled(false)
	assert.True(t, s.IsReadOnly(), "disabled is read-only")
	s.SetEnabled(true)
	s.SetReadOnly(true)
	assert.True(t, s.IsReadOnly())

	s.SetAbilities(true, abilities.Selectable, abilities.Slideable)
	assert.True(t, s.AbilityIs(abilities.Selectable))
	s.SetNonSelectable()
	assert.False(t, s.AbilityIs(abilities.Selectable))
	assert.False(t, s.AbilityIs(abilities.Slideable))
	assert.Equal(t, cursors.None, s.Cursor)
}

func TestStyleSetters(t *testing.T) {
	s := NewStyle()
	s.CenterAll()
	assert.Equal(t, Center, s.Justify.Content)
	assert.Equal(t, Center, s.Align.Items)
	assert.Equal(t, text.Center, s.Text.AlignV)

	s.SetTextWrap(false)
	assert.Equal(t, text.WrapNever, s.Text.WhiteSpace)
	assert.False(t, s.GrowWrap)
	s.SetTextWrap(true)
	assert.Equal(t, text.WrapAsNeeded, s.Text.WhiteSpace)
	assert.True(t, s.GrowWrap)

	assert.False(t, s.IsFlexWrap())
	s.Wrap = true
	assert.True(t, s.IsFlexWrap())
	s.Display = Grid
	assert.False(t, s.IsFlexWrap())

	s.Padding.Set(units.Dp(2))
	s.Margin.Set(units.Dp(3))
	s.Gap.Set(units.Dp(4))
	s.Border.Width.Set(units.Dp(1))
	s.AddBoxShadow(BoxShadow1()...)
	s.MaxBoxShadow = BoxShadow2()
	s.ZeroSpace()
	assert.Equal(t, float32(0), s.Padding.Top.Value)
	assert.Equal(t, float32(0), s.Gap.X.Value)
	assert.Nil(t, s.BoxShadow)
	assert.Nil(t, s.MaxBoxShadow)
}

func TestStyleInheritFields(t *testing.T) {
	parent := NewStyle()
	parent.Color = colors.Uniform(colors.Red)
	parent.Opacity = 0.5
	parent.Font.Size = units.Dp(20)
	parent.Font.Weight = rich.Bold
	parent.Text.TabSize = 2
	parent.Text.LineHeight = 2
	s := NewStyle()
	s.InheritFields(parent)
	assert.Equal(t, parent.Color, s.Color)
	assert.Equal(t, float32(0.5), s.Opacity)
	assert.Equal(t, units.Dp(20), s.Font.Size)
	assert.Equal(t, rich.Bold, s.Font.Weight)
	assert.Equal(t, 2, s.Text.TabSize)

	var f Font
	f.Size = units.Dp(10)
	f.InheritFields(&Font{}) // a zero size is not inherited
	assert.Equal(t, units.Dp(10), f.Size)
	f.SetDecoration(rich.Underline, rich.Overline)
	assert.True(t, f.Decoration.HasFlag(rich.Overline))
}

func TestStyleToDots(t *testing.T) {
	s := NewStyle()
	s.UnitContext.Defaults()
	s.Min.Set(units.Dp(100), units.Dp(50))
	s.Max.Set(units.Dp(50), units.Dp(0))
	s.Padding.Set(units.Dp(2))
	s.Margin.Set(units.Dp(3))
	s.Border.Width.Set(units.Dp(1))
	s.Border.Style.Set(BorderSolid)
	s.Text.Indent = units.Dp(5)
	s.Font.Size = units.Em(1) // recursive: reset to 16dp
	s.ToDots()
	assert.Equal(t, float32(100), s.Min.X.Dots)
	assert.Equal(t, float32(100), s.Max.X.Dots, "max is at least min")
	assert.Equal(t, float32(0), s.Max.Y.Dots, "an unset max stays unset")
	assert.Equal(t, units.Dp(16).Value, s.Font.Size.Value)
	assert.Equal(t, units.UnitDp, s.Font.Size.Unit)
	assert.Equal(t, float32(5), s.Text.Indent.Dots)
	assert.Equal(t, sides.NewFloats(4), s.TotalMargin())
	assert.Equal(t, sides.NewFloats(6), s.BoxSpace())

	assert.Equal(t, float32(16), s.Font.FontHeight())
	s.Text.LineHeight = 1.3
	assert.Equal(t, float32(21), s.LineHeightDots())
	var uc units.Context
	s.Font.SetUnitContext(&uc)
	assert.Equal(t, float32(16), uc.FontEm)
	var z Font
	z.SetUnitContext(&uc)
	assert.Equal(t, float32(16), uc.FontEm, "zero is the default size")

	// a border style of none has no width
	s.Border.Style.Set(BorderNone)
	assert.Equal(t, sides.NewFloats(3), s.TotalMargin())

	// the max border is used if set
	s.Border.Style.Set(BorderSolid)
	s.MaxBorder.Width.Set(units.Dp(2))
	s.MaxBorder.Offset.Set(units.Dp(1))
	s.ToDots()
	assert.Equal(t, sides.NewFloats(6), s.TotalMargin())

	// box shadows add to the margin, unless they are outside
	s.MaxBorder = Border{}
	s.Border.Offset.Set(units.Dp(0))
	s.BoxShadow = []Shadow{{Spread: units.Dp(2)}}
	s.ToDots()
	assert.True(t, s.HasBoxShadow())
	assert.Equal(t, sides.NewFloats(6), s.TotalMargin())
	s.MaxBoxShadow = []Shadow{{Spread: units.Dp(4)}}
	s.ToDots()
	assert.Equal(t, sides.NewFloats(8), s.TotalMargin())
	s.BoxShadowOutside = true
	assert.Equal(t, sides.NewFloats(4), s.TotalMargin())
}

func TestShadow(t *testing.T) {
	uc := units.NewContext()
	sh := Shadow{OffsetX: units.Dp(2), OffsetY: units.Dp(-1), Blur: units.Dp(6), Spread: units.Dp(1)}
	assert.False(t, sh.HasShadow(), "no dots yet")
	sh.ToDots(uc)
	assert.True(t, sh.HasShadow())
	start := math32.Vec2(10, 20)
	assert.Equal(t, math32.Vec2(11, 18), sh.BasePos(start))
	assert.Equal(t, math32.Vec2(12, 22), sh.BaseSize(start))
	assert.Equal(t, math32.Vec2(8, 15), sh.Pos(start))
	assert.Equal(t, math32.Vec2(18, 28), sh.Size(start))
	assert.Equal(t, sides.NewFloats(5, 6, 3, 2), sh.Margin())

	tiny := Shadow{Blur: units.Dot(1), OffsetX: units.Dot(-10)}
	assert.Equal(t, sides.NewFloats(2, 0, 2, 12), tiny.Margin(), "a blur is at least 2, and no side is negative")

	assert.Empty(t, BoxShadow0())
	for i, f := range []func() []Shadow{BoxShadow1, BoxShadow2, BoxShadow3, BoxShadow4, BoxShadow5} {
		s := NewStyle()
		s.BoxShadow = f()
		s.BoxShadowToDots(uc)
		assert.True(t, s.HasBoxShadow(), "elevation %d", i+1)
		m := s.BoxShadowMargin()
		assert.Greater(t, m.Bottom, m.Top, "the shadows are below, elevation %d", i+1)
	}
	s := NewStyle()
	assert.False(t, s.HasBoxShadow())
	assert.Equal(t, sides.Floats{}, s.MaxBoxShadowMargin())
}

func TestComputeActualBackground(t *testing.T) {
	red := colors.Uniform(colors.Red)
	white := colors.Uniform(colors.White)
	s := NewStyle()
	s.ComputeActualBackground(white)
	assert.Equal(t, white, s.ActualBackground, "no background is the parent's")
	assert.Equal(t, white, s.ComputeActualBackgroundFor(image.NewUniform(color.RGBA{}), white), "a nil color is the parent's")
	assert.Equal(t, red, s.ComputeActualBackgroundFor(red, white))

	half := image.NewUniform(color.RGBA{128, 0, 0, 128})
	got := colors.ToUniform(s.ComputeActualBackgroundFor(half, white))
	assert.Equal(t, uint8(255), colors.AsRGBA(got).A, "a translucent color is blended over the parent")

	s.Opacity = 0.5
	got = colors.ToUniform(s.ComputeActualBackgroundFor(red, white))
	assert.Equal(t, colors.AlphaBlend(colors.White, colors.ApplyOpacity(colors.Red, 0.5)), got)

	s.Opacity = 1
	s.StateLayer = 0.5
	s.StateColor = colors.Uniform(colors.Black)
	got = colors.ToUniform(s.ComputeActualBackgroundFor(white, white))
	assert.Less(t, colors.AsRGBA(got).R, uint8(255), "the state layer darkens")
	s.StateColor = nil
	s.Color = colors.Uniform(colors.Black)
	assert.Equal(t, got, colors.ToUniform(s.ComputeActualBackgroundFor(white, white)), "the state color defaults to the color")
}

func TestSubProperties(t *testing.T) {
	props := map[string]any{":hover": map[string]any{"color": "red"}, ":active": 1}
	sp, ok := SubProperties(props, ":hover")
	assert.True(t, ok)
	assert.Equal(t, "red", sp["color"])
	_, ok = SubProperties(props, ":active")
	assert.False(t, ok, "not a map")
	_, ok = SubProperties(props, ":focus")
	assert.False(t, ok)
}

func TestLayoutHelpers(t *testing.T) {
	assert.Equal(t, float32(0), AlignPos(Start, 10, 30))
	assert.Equal(t, float32(10), AlignPos(Center, 10, 30))
	assert.Equal(t, float32(20), AlignPos(End, 10, 30))
	assert.Equal(t, float32(0), AlignPos(End, 40, 30), "no extra space")
	assert.Equal(t, float32(0), AlignFactor(SpaceBetween))

	assert.Equal(t, Center, ItemAlign(Center, Auto))
	assert.Equal(t, End, ItemAlign(Center, End))
	var as AlignSet
	as.Defaults()
	assert.Equal(t, AlignSet{Content: Start, Items: Start, Self: Auto}, as)

	assert.Equal(t, math32.X, Row.Dim())
	assert.Equal(t, math32.Y, Column.Dim())
	assert.Equal(t, Column, Row.Other())
	assert.Equal(t, Row, Column.Other())
}

func TestClamp(t *testing.T) {
	assert.Equal(t, float32(5), ClampMax(5, 0), "no max")
	assert.Equal(t, float32(3), ClampMax(5, 3))
	assert.Equal(t, float32(5), ClampMin(5, 0), "no min")
	assert.Equal(t, float32(7), ClampMin(5, 7))
	v := float32(5)
	SetClampMax(&v, 0)
	SetClampMax(&v, 4)
	assert.Equal(t, float32(4), v)
	SetClampMin(&v, 0)
	SetClampMin(&v, 6)
	assert.Equal(t, float32(6), v)

	vec := math32.Vec2(5, 5)
	assert.Equal(t, math32.Vec2(3, 5), ClampMaxVector(vec, math32.Vec2(3, 0)))
	assert.Equal(t, math32.Vec2(5, 8), ClampMinVector(vec, math32.Vec2(0, 8)))
	SetClampMaxVector(&vec, math32.Vec2(4, 0))
	assert.Equal(t, math32.Vec2(4, 5), vec)
	SetClampMinVector(&vec, math32.Vec2(0, 9))
	assert.Equal(t, math32.Vec2(4, 9), vec)
}

func TestXY(t *testing.T) {
	var xy XY[float32]
	xy.Set(3)
	assert.Equal(t, "(3, 3)", xy.String())
	xy.Set(1, 2)
	assert.Equal(t, float32(1), xy.Dim(math32.X))
	assert.Equal(t, float32(2), xy.Dim(math32.Y))
	xy.SetDim(math32.X, 5)
	xy.SetDim(math32.Y, 6)
	assert.Equal(t, XY[float32]{X: 5, Y: 6}, xy)
	xy.Set()
	assert.Equal(t, XY[float32]{}, xy)
	assert.Panics(t, func() { xy.Dim(math32.Z) })
	assert.Panics(t, func() { xy.SetDim(math32.Z, 1) })
}

func TestObjectSizeFromFit(t *testing.T) {
	wide := math32.Vec2(200, 100)
	box := math32.Vec2(100, 100)
	assert.Equal(t, box, ObjectSizeFromFit(FitFill, wide, box))
	assert.Equal(t, math32.Vec2(100, 50), ObjectSizeFromFit(FitContain, wide, box))
	assert.Equal(t, math32.Vec2(200, 100), ObjectSizeFromFit(FitCover, wide, box))
	tall := math32.Vec2(50, 200)
	assert.Equal(t, math32.Vec2(25, 100), ObjectSizeFromFit(FitContain, tall, box))
	assert.Equal(t, math32.Vec2(100, 400), ObjectSizeFromFit(FitCover, tall, box))
}

func TestResizeImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	box := math32.Vec2(100, 100)
	s := NewStyle()
	for _, resize := range []func(image.Image, math32.Vector2) image.Image{s.ResizeImage, s.ResizeImageLocal} {
		s.ObjectFit = FitContain
		assert.Equal(t, image.Pt(100, 50), resize(img, box).Bounds().Size())
		s.ObjectFit = FitCover
		assert.Equal(t, image.Pt(100, 100), resize(img, box).Bounds().Size(), "cropped to the box")
		s.ObjectFit = FitFill
		assert.Equal(t, image.Pt(100, 100), resize(img, box).Bounds().Size())
		s.ObjectFit = FitScaleDown
		assert.Equal(t, img, resize(img, math32.Vec2(400, 400)), "not scaled up")
		assert.Equal(t, image.Pt(100, 50), resize(img, box).Bounds().Size())
	}
}

func TestToCSS(t *testing.T) {
	s := NewStyle()
	s.UnitContext.Defaults()
	s.Color = colors.Uniform(colors.Red)
	s.Background = colors.Scheme.Primary.Base
	s.Display = Grid
	s.Columns = 3
	s.Direction = Column
	s.Grow.Set(1, 1)
	s.Gap.Set(units.Dp(4))
	s.Wrap = true
	s.Shrink.Set(2, 2)
	s.FlexBasis = units.Dp(50)
	s.Align.Content = Center
	s.Justify.Items = End
	s.Justify.Self = Center
	s.Overflow.Set(OverflowAuto, OverflowHidden)
	s.BoxSizing = BoxBorderBox
	s.AspectRatio = 1.5
	s.ColSpan = 2
	s.RowSpan = 3
	s.Position = PositionAbsolute
	s.Inset.Set(units.Dp(1), units.Dp(2), units.Dp(3), units.Dp(4))
	s.InsetSet.Set(true, false, false, true)
	s.ZIndex = 5
	s.Float = FloatLeft
	s.Clear = ClearLeft
	s.Translate.Set(units.Dp(1), units.Dot(2))
	s.Rotate = 45
	s.Min.Set(units.Dp(10), units.Em(2))
	s.Max.Set(units.Pw(50))
	s.Padding.Set(units.Dp(1))
	s.Margin.Set(units.Dp(2))
	s.Font.Size = units.Dp(20)
	s.Font.Family = rich.Monospace
	s.Font.Weight = rich.Bold
	s.Font.Stretch = rich.Condensed
	s.Font.Slant = rich.Italic
	s.Text.Align = text.Center
	s.Text.WhiteSpace = text.WrapNever
	s.Text.Indent = units.Em(1)
	s.Text.TabSize = 8
	s.Text.AlignV = text.Center
	s.Border.Width.Set(units.Dp(1))
	s.Border.Style.Set(BorderDashed)
	s.Border.Color.Set(nil)
	s.Border.Radius.Set(units.Dp(4))
	s.BoxShadow = []Shadow{{OffsetY: units.Dp(1), Blur: units.Dp(2), Color: colors.Uniform(colors.Black), Inset: true}, {}}
	s.BoxShadowToDots(&s.UnitContext)
	s.Opacity = 0.5

	css := ToCSS(s, "frame", "td")
	for _, want := range []string{
		"color:#FF0000", "background:var(--primary-color)", "display:grid",
		"flex-direction:column", "flex-grow:1", "grid-template-columns:repeat(3,auto)",
		"row-gap:4px", "flex-wrap:wrap", "flex-shrink:2", "flex-basis:50px",
		"align-content:center", "justify-items:end", "justify-self:center",
		"overflow-x:auto", "overflow-y:hidden", "box-sizing:border-box", "aspect-ratio:1.5",
		"grid-column:span 2", "grid-row:span 3", "position:absolute", "top:1px", "left:4px",
		"z-index:5", "float:left", "clear:left", "transform:translate(1px,2px) rotate(45deg)",
		"min-width:10px", "min-height:2em", "max-width:50%", "padding-top:1px", "margin-left:2px",
		"font-size:20px", "font-family:monospace", "font-weight:700", "font-stretch:condensed",
		"font-style:italic", "text-align:center",
		"white-space:nowrap", "text-indent:1em", "tab-size:8", "vertical-align:middle",
		"border-top:1px dashed currentColor", "border-radius:4px",
		"box-shadow:inset 0px 1px 2px 0px #000000", "opacity:0.5",
	} {
		assert.Contains(t, strings.Split(css, ";"), want)
	}
	parts := strings.Split(css, ";")
	assert.NotContains(t, parts, "right:2px", "only the inset sides that are set")
	assert.NotContains(t, parts, "width:10px", "no width with grow")

	// text has no layout, and an svg has stroke and fill
	css = ToCSS(s, "text", "svg")
	assert.NotContains(t, css, "display:")
	assert.Contains(t, css, "stroke:#FF0000")
	assert.Contains(t, css, "fill:#FF0000")
	assert.NotContains(t, css, "vertical-align")

	s = NewStyle()
	s.SetState(true, states.Invisible)
	s.Font.Family = rich.Custom
	s.Font.CustomFont = "Inter"
	s.Text.WhiteSpace = text.WhiteSpacePre
	s.Text.AlignV = text.End
	s.Border.Radius.Set(units.Dp(1), units.Dp(2), units.Dp(3), units.Dp(4))
	css = ToCSS(s, "frame", "th")
	parts = strings.Split(css, ";")
	assert.Contains(t, parts, "display:none")
	assert.Contains(t, parts, `font-family:"Inter"`)
	assert.Contains(t, parts, "white-space:pre")
	assert.Contains(t, parts, "vertical-align:bottom")
	assert.Contains(t, parts, "border-radius:1px 2px 3px 4px")
	s.Text.WhiteSpace = text.WhiteSpacePreWrap
	assert.Contains(t, ToCSS(s, "frame", "div"), "white-space:pre-wrap")
}

func TestColorToCSS(t *testing.T) {
	assert.Equal(t, "", colorToCSS(nil))
	assert.Equal(t, "", colorToCSS(colors.Scheme.Surface), "the default")
	assert.Equal(t, "var(--primary-on-color)", colorToCSS(colors.Scheme.Primary.On))
	assert.Equal(t, "var(--secondary-container-color)", colorToCSS(colors.Scheme.Secondary.Container))
	assert.Equal(t, "var(--secondary-on-container-color)", colorToCSS(colors.Scheme.Secondary.OnContainer))
	assert.Equal(t, "var(--surface-container-color)", colorToCSS(colors.Scheme.SurfaceContainerHigh))

	g := gradient.NewLinear().AddStop(colors.White, 0).AddStop(colors.Black, 1, 0.5)
	assert.Equal(t, "linear-gradient(90deg,#FFFFFF 0%,#0000007F 100%)", colorToCSS(g))
	g.End = math32.Vec2(0, 1)
	assert.Contains(t, colorToCSS(g), "180deg", "to the bottom")

	assert.Equal(t, "currentColor", colorToCSSOr(nil, "currentColor"))
	assert.Equal(t, colors.AsHex(colors.ToUniform(colors.Scheme.OnSurface)), colorToCSSOr(colors.Scheme.OnSurface, "x"), "the default text color is written")

	s := NewStyle()
	s.UnitContext.Defaults()
	assert.Equal(t, "8px", ToCSSLength(s, units.Dot(8)))
	assert.Equal(t, "2em", ToCSSLength(s, units.Em(2)))
	assert.Equal(t, "", decorationToCSS(0))
	var d rich.Decorations
	d.SetFlag(true, rich.Underline, rich.LineThrough)
	assert.Equal(t, "underline line-through", decorationToCSS(d))
	d = 0
	d.SetFlag(true, rich.DottedUnderline)
	assert.Equal(t, "underline dotted", decorationToCSS(d))
}

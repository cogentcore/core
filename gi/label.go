// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"

	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

////////////////////////////////////////////////////////////////////////////////////////
// Label

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering, and full HTML styling, including links -- LinkSig
// emits link with data of URL -- opens default browser if nobody receiving
// signal.  The default white-space option is 'pre' -- set to 'normal' or
// other options to get word-wrapping etc.
type Label struct {
	WidgetBase
	Text               string                   `xml:"text" desc:"label to display"`
	Selectable         bool                     `desc:"is this label selectable? if so, it will change background color in response to selection events and update selection state on mouse clicks"`
	Redrawable         bool                     `desc:"is this label going to be redrawn frequently without an overall full re-render?  if so, you need to set this flag to avoid weird overlapping rendering results from antialiasing.  Also, if the label will change dynamically, this must be set to true, otherwise labels will illegibly overlay on top of each other."`
	Type               LabelTypes               `desc:"the type of label (p, h1, h2, etc)"`
	State              LabelStates              `desc:"the current state of the label (active, inactive, selected, etc)"`
	LinkSig            ki.Signal                `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, calls TextLinkHandler then URLHandler"`
	StateStyles        [LabelStatesN]gist.Style `copy:"-" json:"-" xml:"-" desc:"styles for different states of label"`
	Render             girl.Text                `copy:"-" xml:"-" json:"-" desc:"render data for text label"`
	RenderPos          mat32.Vec2               `copy:"-" xml:"-" json:"-" desc:"position offset of start of text rendering, from last render -- AllocPos plus alignment factors for center, right etc."`
	CurBackgroundColor gist.Color               `copy:"-" xml:"-" json:"-" desc:"current background color -- grabbed when rendering for first time, and used when toggling off of selected mode, or for redrawable, to wipe out bg"`
}

var TypeLabel = kit.Types.AddType(&Label{}, LabelProps)

// LabelTypes is an enum containing the different
// possible types of labels
type LabelTypes int

const (
	// LabelStandard is a standard label that just contains
	// text, similar to HTML's <span> and <label> elements
	LabelStandard LabelTypes = iota
	// LabelP is a paragraph-style label,
	// similar to HTML's <p> element
	LabelP
	// LabelH1 is a large heading-style label,
	// similar to HTML's <h1> element
	LabelH1
	// LabelH2 is a heading-style label slightly
	// smaller than H1, similar to HTML's <h2> element
	LabelH2
	// LabelH3 is a heading-style label slightly
	// smaller than H2, similar to HTML's <h3> element
	LabelH3
	// LabelH4 is a heading-style label slightly
	// smaller than H3, similar to HTML's <h4> element
	LabelH4
	// LabelH5 is a heading-style label slightly
	// smaller than H4, similar to HTML's <h5> element
	LabelH5
	// LabelH6 is a heading-style label slightly
	// smaller than H5, similar to HTML's <h6> element
	LabelH6

	LabelTypesN
)

var TypeLabelTypes = kit.Enums.AddEnumAltLower(LabelTypesN, kit.NotBitFlag, gist.StylePropProps, "Label")

//go:generate stringer -type=LabelTypes

// AddNewLabel adds a new label to given parent node, with given name and text.
func AddNewLabel(parent ki.Ki, name string, text string) *Label {
	lb := parent.AddNewChild(TypeLabel, name).(*Label)
	lb.Text = text
	return lb
}

func (lb *Label) CopyFieldsFrom(frm any) {
	fr := frm.(*Label)
	lb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	lb.Text = fr.Text
	lb.Selectable = fr.Selectable
	lb.Redrawable = fr.Redrawable
}

func (lb *Label) Disconnect() {
	lb.WidgetBase.Disconnect()
	lb.LinkSig.DisconnectAll()
}

// // DefaultStyle implements the [DefaultStyler] interface
// func (lb *Label) DefaultStyle() {
// 	cs := CurrentColorScheme()
// 	s := &lb.Style

// 	s.Text.WhiteSpace = gist.WhiteSpacePre
// 	s.Padding.Set(units.Px(2))
// 	s.Margin.Set(units.Px(2))
// 	s.AlignV = gist.AlignTop
// 	s.Color.SetColor(cs.Font)
// 	s.BackgroundColor.SetColor(color.Transparent)

// 	switch lb.Type {
// 	case LabelP:
// 		s.Font.Size.SetRem(1)
// 	case LabelLabel:
// 		s.Font.Size.SetRem(0.75)
// 	case LabelH1:
// 		s.Font.Size.SetRem(2)
// 		s.Font.Weight = gist.WeightBold
// 	case LabelH2:
// 		s.Font.Size.SetRem(1.5)
// 		s.Font.Weight = gist.WeightBold
// 	case LabelH3:
// 		s.Font.Size.SetRem(1.25)
// 		s.Font.Weight = gist.WeightBold
// 	}
// }

var LabelProps = ki.Props{
	"EnumType:Flag": TypeNodeFlags,
	// "white-space":      gist.WhiteSpacePre, // no wrap, use spaces unless otherwise specified!
	// "padding":          units.Px(2),
	// "margin":           units.Px(2),
	// "vertical-align":   gist.AlignTop,
	// "color":            &Prefs.Colors.Font,
	// "background-color": color.Transparent,
	// LabelSelectors[LabelActive]: ki.Props{
	// 	"background-color": color.Transparent,
	// },
	// LabelSelectors[LabelInactive]: ki.Props{
	// 	"color": "lighter-50",
	// },
	// LabelSelectors[LabelSelected]: ki.Props{
	// 	"background-color": &Prefs.Colors.Select,
	// },
}

// LabelStates are mutually-exclusive label states -- determines appearance
type LabelStates int32

const (
	// normal active state
	LabelActive LabelStates = iota

	// inactive -- font is dimmed
	LabelInactive

	// selected -- background is selected color
	LabelSelected

	// total number of button states
	LabelStatesN
)

//go:generate stringer -type=LabelStates

var TypeLabelStates = kit.Enums.AddEnumAltLower(LabelStatesN, kit.NotBitFlag, gist.StylePropProps, "Label")

func (ev LabelStates) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *LabelStates) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// LabelSelectors are Style selector names for the different states:
var LabelSelectors = []string{":active", ":inactive", ":selected"}

// SetText sets the text and updates the rendered version.
// Note: if there is already a label set, and no other
// larger updates are taking place, the new label may just
// illegibly overlay on top of the old one.
// Set Redrawable = true to fix this issue (it will redraw
// the background -- sampling from actual if none is set).
func (lb *Label) SetText(txt string) {
	updt := lb.UpdateStart()
	// if lb.Text != "" { // not good to automate this -- better to use docs -- bg can be bad
	// 	lb.Redrawable = true
	// }

	lb.StyMu.RLock()
	needSty := lb.Style.Font.Size.Val == 0
	lb.StyMu.RUnlock()
	if needSty {
		lb.StyleLabel()
	}
	lb.SetStateStyle()
	lb.StyMu.RLock()
	lb.Text = txt
	lb.Style.BackgroundColor.Color.SetToNil() // always use transparent bg for actual text
	// this makes it easier for it to update with dynamic bgs
	if lb.Text == "" {
		lb.Render.SetHTML(" ", lb.Style.FontRender(), &lb.Style.Text, &lb.Style.UnContext, lb.CSSAgg)
	} else {
		lb.Render.SetHTML(lb.Text, lb.Style.FontRender(), &lb.Style.Text, &lb.Style.UnContext, lb.CSSAgg)
	}
	spc := lb.BoxSpace()
	sz := lb.LayState.Alloc.Size
	if sz.IsNil() {
		sz = lb.LayState.SizePrefOrMax()
	}
	if !sz.IsNil() {
		sz.SetSub(spc.Size())
	}
	lb.Render.LayoutStdLR(&lb.Style.Text, lb.Style.FontRender(), &lb.Style.UnContext, sz)
	lb.StyMu.RUnlock()
	lb.UpdateEnd(updt)
}

// SetStateStyle sets the style based on the inactive, selected flags
func (lb *Label) SetStateStyle() {
	lb.StyMu.Lock()
	if lb.IsInactive() {
		lb.State = LabelInactive
		lb.Style = lb.StateStyles[LabelInactive]
		if lb.Redrawable && !lb.CurBackgroundColor.IsNil() {
			lb.Style.BackgroundColor.SetColor(lb.CurBackgroundColor)
		}
	} else if lb.IsSelected() {
		lb.State = LabelSelected
		lb.Style = lb.StateStyles[LabelSelected]
	} else {
		lb.State = LabelActive
		lb.Style = lb.StateStyles[LabelActive]
		if (lb.Selectable || lb.Redrawable) && !lb.CurBackgroundColor.IsNil() {
			lb.Style.BackgroundColor.SetColor(lb.CurBackgroundColor)
		}
	}
	lb.StyMu.Unlock()
}

// OpenLink opens given link, either by sending LinkSig signal if there are
// receivers, or by calling the TextLinkHandler if non-nil, or URLHandler if
// non-nil (which by default opens user's default browser via
// oswin/App.OpenURL())
func (lb *Label) OpenLink(tl *girl.TextLink) {
	tl.Widget = lb.This()
	if len(lb.LinkSig.Cons) == 0 {
		if girl.TextLinkHandler != nil {
			if girl.TextLinkHandler(*tl) {
				return
			}
		}
		if girl.URLHandler != nil {
			girl.URLHandler(tl.URL)
		}
		return
	}
	lb.LinkSig.Emit(lb.This(), 0, tl.URL) // todo: could potentially signal different target=_blank kinds of options here with the sig
}

func (lb *Label) HoverEvent() {
	lb.ConnectEvent(oswin.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		llb := recv.Embed(TypeLabel).(*Label)
		hasLinks := len(lb.Render.Links) > 0
		if hasLinks {
			pos := llb.RenderPos
			for ti := range llb.Render.Links {
				tl := &llb.Render.Links[ti]
				tlb := tl.Bounds(&llb.Render, pos)
				if me.Where.In(tlb) {
					PopupTooltip(tl.URL, tlb.Max.X, tlb.Max.Y, llb.Viewport, llb.Nm)
					me.SetProcessed()
					return
				}
			}
		}
		if llb.Tooltip != "" {
			me.SetProcessed()
			llb.BBoxMu.RLock()
			pos := llb.WinBBox.Max
			llb.BBoxMu.RUnlock()
			pos.X -= 20
			PopupTooltip(llb.Tooltip, pos.X, pos.Y, llb.Viewport, llb.Nm)
		}
	})
}

func (lb *Label) MouseEvent() {
	lb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		llb := recv.Embed(TypeLabel).(*Label)
		hasLinks := len(llb.Render.Links) > 0
		pos := llb.RenderPos
		if llb.Selectable || hasLinks {
			if me.Action == mouse.Press && me.Button == mouse.Left {
				if hasLinks {
					for ti := range llb.Render.Links {
						tl := &llb.Render.Links[ti]
						tlb := tl.Bounds(&llb.Render, pos)
						if me.Where.In(tlb) {
							llb.OpenLink(tl)
							me.SetProcessed()
							return
						}
					}
				}
				if llb.Selectable {
					llb.SetSelectedState(!llb.IsSelected())
					llb.EmitSelectedSignal()
					llb.UpdateSig()
				}
			}
		}
		if me.Action == mouse.Release && me.Button == mouse.Right {
			me.SetProcessed()
			llb.EmitContextMenuSignal()
			llb.This().(Node2D).ContextMenu()
		}
	})
}

func (lb *Label) MouseMoveEvent() {
	hasLinks := len(lb.Render.Links) > 0
	if !hasLinks {
		return
	}
	lb.ConnectEvent(oswin.MouseMoveEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.MoveEvent)
		me.SetProcessed()
		llb := recv.Embed(TypeLabel).(*Label)
		pos := llb.RenderPos
		inLink := false
		for _, tl := range llb.Render.Links {
			tlb := tl.Bounds(&llb.Render, pos)
			if me.Where.In(tlb) {
				inLink = true
				break
			}
		}
		if inLink {
			oswin.TheApp.Cursor(lb.ParentWindow().OSWin).PushIfNot(cursor.HandPointing)
		} else {
			oswin.TheApp.Cursor(lb.ParentWindow().OSWin).PopIf(cursor.HandPointing)
		}
	})
}

func (lb *Label) LabelEvents() {
	lb.HoverEvent()
	lb.MouseEvent()
	lb.MouseMoveEvent()
}

func (lb *Label) GrabCurBackgroundColor() {
	if lb.Viewport == nil || lb.IsSelected() {
		return
	}
	if !gist.RebuildDefaultStyles && !lb.CurBackgroundColor.IsNil() {
		return
	}
	pos := lb.ContextMenuPos()
	clr := lb.Viewport.Pixels.At(pos.X, pos.Y)
	lb.CurBackgroundColor.SetColor(clr)
}

// StyleLabel does label styling -- it sets the StyMu Lock
func (lb *Label) StyleLabel() {
	lb.StyMu.Lock()
	defer lb.StyMu.Unlock()

	hasTempl, saveTempl := lb.Style.FromTemplate()
	if !hasTempl || saveTempl {
		lb.Style2DWidget()
	}
	if hasTempl && saveTempl {
		lb.Style.SaveTemplate()
	}
	parSty := lb.ParentActiveStyle()
	if hasTempl && !saveTempl {
		for i := 0; i < int(LabelStatesN); i++ {
			lb.StateStyles[i].Template = lb.Style.Template + LabelSelectors[i]
			lb.StateStyles[i].FromTemplate()
		}
	} else {
		for i := 0; i < int(LabelStatesN); i++ {
			lb.StateStyles[i].CopyFrom(&lb.Style)
			lb.StateStyles[i].SetStyleProps(parSty, lb.StyleProps(LabelSelectors[i]), lb.Viewport)
			lb.StateStyles[i].CopyUnitContext(&lb.Style.UnContext)
		}
	}
	if hasTempl && saveTempl {
		for i := 0; i < int(LabelStatesN); i++ {
			lb.StateStyles[i].Template = lb.Style.Template + LabelSelectors[i]
			lb.StateStyles[i].SaveTemplate()
		}
	}
	if lb.CurBackgroundColor.IsNil() && !lb.Style.BackgroundColor.Color.IsNil() {
		lb.CurBackgroundColor = lb.Style.BackgroundColor.Color
	}
	lb.ParentStyleRUnlock()
}

func (lb *Label) LayoutLabel() {
	lb.StyMu.RLock()
	defer lb.StyMu.RUnlock()

	lb.Style.BackgroundColor.Color.SetToNil() // always use transparent bg for actual text
	lb.Render.SetHTML(lb.Text, lb.Style.FontRender(), &lb.Style.Text, &lb.Style.UnContext, lb.CSSAgg)
	spc := lb.BoxSpace()
	sz := lb.LayState.SizePrefOrMax()
	if !sz.IsNil() {
		sz.SetSub(spc.Size())
	}
	lb.Render.LayoutStdLR(&lb.Style.Text, lb.Style.FontRender(), &lb.Style.UnContext, sz)
}

func (lb *Label) Style2D() {
	lb.StyleLabel()
	lb.StyMu.Lock()
	lb.LayState.SetFromStyle(&lb.Style) // also does reset
	lb.StyMu.Unlock()
	lb.LayoutLabel()
}

func (lb *Label) Size2D(iter int) {
	if iter > 0 && lb.Style.Text.HasWordWrap() {
		return // already updated in previous iter, don't redo!
	} else {
		lb.InitLayout2D()
		sz := lb.LayState.Size.Pref // SizePrefOrMax()
		sz = sz.Max(lb.Render.Size)
		lb.Size2DFromWH(sz.X, sz.Y)
	}
}

func (lb *Label) Layout2D(parBBox image.Rectangle, iter int) bool {
	lb.Layout2DBase(parBBox, true, iter)
	for i := 0; i < int(LabelStatesN); i++ {
		lb.StateStyles[i].CopyUnitContext(&lb.Style.UnContext)
	}
	lb.Layout2DChildren(iter) // todo: maybe shouldn't call this on known terminals?
	sz := lb.Size2DSubSpace()
	lb.Style.BackgroundColor.Color.SetToNil() // always use transparent bg for actual text
	lb.Render.SetHTML(lb.Text, lb.Style.FontRender(), &lb.Style.Text, &lb.Style.UnContext, lb.CSSAgg)
	lb.Render.LayoutStdLR(&lb.Style.Text, lb.Style.FontRender(), &lb.Style.UnContext, sz)
	if lb.Style.Text.HasWordWrap() {
		if lb.Render.Size.Y < (sz.Y - 1) { // allow for numerical issues
			lb.LayState.SetFromStyle(&lb.Style)
			lb.Size2DFromWH(lb.Render.Size.X, lb.Render.Size.Y)
			return true // needs a redo!
		}
	}
	return false
}

func (lb *Label) TextPos() mat32.Vec2 {
	lb.StyMu.RLock()
	pos := lb.LayState.Alloc.Pos.Add(lb.Style.BoxSpace().Pos())
	lb.StyMu.RUnlock()
	return pos
}

func (lb *Label) RenderLabel() {
	lb.GrabCurBackgroundColor()
	lb.SetStateStyle()
	rs, _, st := lb.RenderLock()
	defer lb.RenderUnlock(rs)
	lb.RenderPos = lb.TextPos()
	lb.RenderStdBox(st)
	lb.Render.Render(rs, lb.RenderPos)
}

func (lb *Label) Render2D() {
	if lb.FullReRenderIfNeeded() {
		return
	}
	if lb.PushBounds() {
		lb.This().(Node2D).ConnectEvents2D()
		lb.RenderLabel()
		lb.Render2DChildren()
		lb.PopBounds()
	} else {
		lb.DisconnectAllEvents(RegPri)
	}
}

func (lb *Label) ConnectEvents2D() {
	lb.LabelEvents()
}

func (lb *Label) Init2D() {
	lb.Init2DWidget()
	lb.ConfigStyles()
}

func (lb *Label) ConfigStyles() {
	lb.AddStyleFunc(StyleFuncDefault, func() {
		lb.Style.Text.WhiteSpace = gist.WhiteSpaceNormal
		lb.Style.AlignV = gist.AlignTop
		lb.Style.Color = ColorScheme.Text
		lb.Style.BackgroundColor.SetColor(color.Transparent)
		lb.Style.MaxWidth.SetPct(100)
		switch lb.State {
		case LabelActive:
		case LabelInactive:
			lb.Style.Color = ColorScheme.Text.Highlight(30)
		case LabelSelected:
			lb.Style.BackgroundColor.SetColor(ColorScheme.Accent)
		}
		// Styling based on https://www.w3schools.com/tags/tag_hn.asp
		// and https://www.w3schools.com/tags/tag_p.asp
		switch lb.Type {
		case LabelStandard:
			lb.Style.Font.Size.SetRem(1)
			lb.Style.Text.WhiteSpace = gist.WhiteSpacePre
			lb.Style.Margin.Set(units.Px(2 * Prefs.DensityMul()))
			lb.Style.Padding.Set(units.Px(2 * Prefs.DensityMul()))
		case LabelP:
			lb.Style.Font.Size.SetRem(1)
			lb.Style.Margin.Set(units.Em(1*Prefs.DensityMul()), units.Px(0))
		case LabelH1:
			lb.Style.Font.Size.SetRem(2)
			lb.Style.Font.Weight = gist.WeightBold
			lb.Style.Margin.Set(units.Em(0.67*Prefs.DensityMul()), units.Px(0))
		case LabelH2:
			lb.Style.Font.Size.SetRem(1.5)
			lb.Style.Font.Weight = gist.WeightBold
			lb.Style.Margin.Set(units.Em(0.83*Prefs.DensityMul()), units.Px(0))
		case LabelH3:
			lb.Style.Font.Size.SetRem(1.17)
			lb.Style.Font.Weight = gist.WeightBold
			lb.Style.Margin.Set(units.Em(1*Prefs.DensityMul()), units.Px(0))
		case LabelH4:
			lb.Style.Font.Size.SetRem(1)
			lb.Style.Font.Weight = gist.WeightBold
			lb.Style.Margin.Set(units.Em(1.33*Prefs.DensityMul()), units.Px(0))
		case LabelH5:
			lb.Style.Font.Size.SetRem(0.83)
			lb.Style.Font.Weight = gist.WeightBold
			lb.Style.Margin.Set(units.Em(1.67*Prefs.DensityMul()), units.Px(0))
		case LabelH6:
			lb.Style.Font.Size.SetRem(0.67)
			lb.Style.Font.Weight = gist.WeightBold
			lb.Style.Margin.Set(units.Em(2.33*Prefs.DensityMul()), units.Px(0))
		}
	})
}

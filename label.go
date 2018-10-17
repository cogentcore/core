// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"reflect"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Labeler Interface and ToLabel method

// the labeler interface provides a GUI-appropriate label (todo: rich text
// html tags!?) for an item -- use ToLabel converter to attempt to use this
// interface and then fall back on Stringer via kit.ToString conversion
// function
type Labeler interface {
	Label() string
}

// ToLabel returns the gui-appropriate label for an item, using the Labeler
// interface if it is defined, and falling back on kit.ToString converter
// otherwise -- also contains label impls for basic interface types for which
// we cannot easily define the Labeler interface
func ToLabel(it interface{}) string {
	lbler, ok := it.(Labeler)
	if !ok {
		// typ := reflect.TypeOf(it)
		// if kit.EmbeddedTypeImplements(typ, reflect.TypeOf((*reflect.Type)(nil)).Elem()) {
		// 	to, ok :=
		// }
		switch v := it.(type) {
		case reflect.Type:
			return v.Name()
		case ki.Ki:
			return v.Name()
		}
		return kit.ToString(it)
	}
	return lbler.Label()
}

////////////////////////////////////////////////////////////////////////////////////////
// Label

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering, and full HTML styling, including links -- LinkSig
// emits link with data of URL -- opens default browser if nobody receiving
// signal.  The default white-space option is 'pre' -- set to 'normal' or
// other options to get word-wrapping etc.
type Label struct {
	WidgetBase
	Text        string              `xml:"text" desc:"label to display"`
	Selectable  bool                `desc:"is this label selectable? if so, it will change background color in response to selection events and update selection state on mouse clicks"`
	Redrawable  bool                `desc:"is this label going to be redrawn frequently without an overall full re-render?  if so, you need to set this flag to avoid weird overlapping rendering results from antialiasing"`
	LinkSig     ki.Signal           `json:"-" xml:"-" view:"-" desc:"signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, calls TextLinkHandler then URLHandler"`
	StateStyles [LabelStatesN]Style `json:"-" xml:"-" desc:"styles for different states of label"`
	Render      TextRender          `xml:"-" json:"-" desc:"render data for text label"`
	RenderPos   Vec2D               `xml:"-" json:"-" desc:"position offset of start of text rendering, from last render -- AllocPos plus alignment factors for center, right etc."`
	CurBgColor  Color               `xml:"-" json:"-" desc:"current background color -- grabbed when rendering for first time, and used when toggling off of selected mode, or for redrawable, to wipe out bg"`
}

var KiT_Label = kit.Types.AddType(&Label{}, LabelProps)

var LabelProps = ki.Props{
	"white-space":      WhiteSpacePre, // no wrap, use spaces unless otherwise specified!
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   AlignTop,
	"color":            &Prefs.Colors.Font,
	"background-color": color.Transparent,
	LabelSelectors[LabelActive]: ki.Props{
		"background-color": color.Transparent,
	},
	LabelSelectors[LabelInactive]: ki.Props{
		"color": "lighter-50",
	},
	LabelSelectors[LabelSelected]: ki.Props{
		"background-color": &Prefs.Colors.Select,
	},
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

var KiT_LabelStates = kit.Enums.AddEnumAltLower(LabelStatesN, false, StylePropProps, "Label")

func (ev LabelStates) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *LabelStates) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// LabelSelectors are Style selector names for the different states:
var LabelSelectors = []string{":active", ":inactive", ":selected"}

// SetText sets the text and updates the rendered version
func (lb *Label) SetText(txt string) {
	updt := lb.UpdateStart()
	if updt {
		fmt.Printf("label is updating: %v\n", txt)
	}
	if lb.Sty.Font.Size.Val == 0 { // not yet styled
		lb.StyleLabel()
	}
	lb.Text = txt
	if lb.Text == "" {
		lb.Render.SetHTML(" ", &lb.Sty.Font, &lb.Sty.Text, &lb.Sty.UnContext, lb.CSSAgg)
	} else {
		lb.Render.SetHTML(lb.Text, &lb.Sty.Font, &lb.Sty.Text, &lb.Sty.UnContext, lb.CSSAgg)
	}
	spc := lb.Sty.BoxSpace()
	sz := lb.LayData.AllocSize
	if sz.IsZero() {
		sz = lb.LayData.SizePrefOrMax()
	}
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}
	lb.Render.LayoutStdLR(&lb.Sty.Text, &lb.Sty.Font, &lb.Sty.UnContext, sz)
	lb.UpdateEnd(updt)
}

// Label returns the display label for this node, satisfying the Labeler interface
func (lb *Label) Label() string {
	if lb.Text != "" {
		return lb.Text
	}
	return lb.Nm
}

// SetStateStyle sets the style based on the inactive, selected flags
func (lb *Label) SetStateStyle() {
	if lb.IsInactive() {
		lb.Sty = lb.StateStyles[LabelInactive]
		if lb.Redrawable && !lb.CurBgColor.IsNil() {
			lb.Sty.Font.BgColor.SetColor(lb.CurBgColor)
		}
	} else if lb.IsSelected() {
		lb.Sty = lb.StateStyles[LabelSelected]
	} else {
		lb.Sty = lb.StateStyles[LabelActive]
		if (lb.Selectable || lb.Redrawable) && !lb.CurBgColor.IsNil() {
			lb.Sty.Font.BgColor.SetColor(lb.CurBgColor)
		}
	}
}

// OpenLink opens given link, either by sending LinkSig signal if there are
// receivers, or by calling the TextLinkHandler if non-nil, or URLHandler if
// non-nil (which by default opens user's default browser via
// oswin/App.OpenURL())
func (lb *Label) OpenLink(tl *TextLink) {
	tl.Widget = lb.This.(Node2D)
	if len(lb.LinkSig.Cons) == 0 {
		if TextLinkHandler != nil {
			if TextLinkHandler(*tl) {
				return
			}
		}
		if URLHandler != nil {
			URLHandler(tl.URL)
		}
		return
	}
	lb.LinkSig.Emit(lb.This, 0, tl.URL) // todo: could potentially signal different target=_blank kinds of options here with the sig
}

func (lb *Label) HoverEvent() {
	lb.ConnectEvent(oswin.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		llb := recv.Embed(KiT_Label).(*Label)
		hasLinks := len(lb.Render.Links) > 0
		if hasLinks {
			pos := llb.RenderPos
			for ti, _ := range llb.Render.Links {
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
			pos := llb.WinBBox.Max
			pos.X -= 20
			PopupTooltip(llb.Tooltip, pos.X, pos.Y, llb.Viewport, llb.Nm)
		}
	})
}

func (lb *Label) MouseEvent() {
	lb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		llb := recv.Embed(KiT_Label).(*Label)
		hasLinks := len(llb.Render.Links) > 0
		pos := llb.RenderPos
		if llb.Selectable || hasLinks {
			if me.Action == mouse.Press && me.Button == mouse.Left {
				if hasLinks {
					for ti, _ := range llb.Render.Links {
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
			llb.This.(Node2D).ContextMenu()
		}
	})
}

func (lb *Label) MouseMoveEvent() {
	hasLinks := len(lb.Render.Links) > 0
	if hasLinks {
		lb.ConnectEvent(oswin.MouseMoveEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			me := d.(*mouse.MoveEvent)
			me.SetProcessed()
			llb := recv.Embed(KiT_Label).(*Label)
			pos := llb.RenderPos
			inLink := false
			for _, tl := range llb.Render.Links {
				tlb := tl.Bounds(&llb.Render, pos)
				if me.Where.In(tlb) {
					inLink = true
				}
			}
			if inLink {
				oswin.TheApp.Cursor(lb.Viewport.Win.OSWin).PushIfNot(cursor.HandPointing)
			} else {
				oswin.TheApp.Cursor(lb.Viewport.Win.OSWin).PopIf(cursor.HandPointing)
			}
		})
	}
}

func (lb *Label) LabelEvents() {
	lb.HoverEvent()
	lb.MouseEvent()
	lb.MouseMoveEvent()
}

func (lb *Label) GrabCurBgColor() {
	if lb.Viewport == nil || lb.IsSelected() {
		return
	}
	if !RebuildDefaultStyles && !lb.CurBgColor.IsNil() {
		return
	}
	pos := lb.ContextMenuPos()
	clr := lb.Viewport.Pixels.At(pos.X, pos.Y)
	lb.CurBgColor.SetColor(clr)
}

func (lb *Label) TextPos() Vec2D {
	sty := &lb.Sty
	pos := lb.LayData.AllocPos.AddVal(sty.BoxSpace())
	if !sty.Text.HasWordWrap() { // word-wrap case already deals with this b/c it has final alloc size -- otherwise it lays out "blind" and can't do this.
		if lb.LayData.AllocSize.X > lb.Render.Size.X {
			if IsAlignMiddle(sty.Layout.AlignH) {
				pos.X += 0.5 * (lb.LayData.AllocSize.X - lb.Render.Size.X)
			} else if IsAlignEnd(sty.Layout.AlignH) {
				pos.X += (lb.LayData.AllocSize.X - lb.Render.Size.X)
			}
		}
		if lb.LayData.AllocSize.Y > lb.Render.Size.Y {
			if IsAlignMiddle(sty.Layout.AlignV) {
				pos.Y += 0.5 * (lb.LayData.AllocSize.Y - lb.Render.Size.Y)
			} else if IsAlignEnd(sty.Layout.AlignV) {
				pos.Y += (lb.LayData.AllocSize.Y - lb.Render.Size.Y)
			}
		}
	}
	return pos
}

func (lb *Label) StyleLabel() {
	lb.Style2DWidget()
	if lb.Sty.Text.Align != AlignLeft && lb.Sty.Layout.AlignH == AlignLeft {
		lb.Sty.Layout.AlignH = lb.Sty.Text.Align // keep them consistent -- this is what people expect
	} else if lb.Sty.Layout.AlignH != AlignLeft && lb.Sty.Text.Align == AlignLeft {
		lb.Sty.Text.Align = lb.Sty.Layout.AlignH // keep them consistent -- this is what people expect
	}
	pst := lb.ParentStyle()
	for i := 0; i < int(LabelStatesN); i++ {
		lb.StateStyles[i].CopyFrom(&lb.Sty)
		lb.StateStyles[i].SetStyleProps(pst, lb.StyleProps(LabelSelectors[i]))
		lb.StateStyles[i].CopyUnitContext(&lb.Sty.UnContext)
	}
}

func (lb *Label) LayoutLabel() {
	lb.Render.SetHTML(lb.Text, &lb.Sty.Font, &lb.Sty.Text, &lb.Sty.UnContext, lb.CSSAgg)
	spc := lb.Sty.BoxSpace()
	sz := lb.LayData.SizePrefOrMax()
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}
	lb.Render.LayoutStdLR(&lb.Sty.Text, &lb.Sty.Font, &lb.Sty.UnContext, sz)
}

func (lb *Label) Style2D() {
	lb.StyleLabel()
	lb.LayData.SetFromStyle(&lb.Sty.Layout) // also does reset
	lb.LayoutLabel()
}

func (lb *Label) Size2D(iter int) {
	if iter > 0 && lb.Sty.Text.HasWordWrap() {
		return // already updated in previous iter, don't redo!
	} else {
		lb.InitLayout2D()
		lb.Size2DFromWH(lb.Render.Size.X, lb.Render.Size.Y)
	}
}

func (lb *Label) Layout2D(parBBox image.Rectangle, iter int) bool {
	lb.Layout2DBase(parBBox, true, iter)
	for i := 0; i < int(LabelStatesN); i++ {
		lb.StateStyles[i].CopyUnitContext(&lb.Sty.UnContext)
	}
	lb.Layout2DChildren(iter) // todo: maybe shouldn't call this on known terminals?
	sz := lb.Size2DSubSpace()
	if lb.Sty.Text.HasWordWrap() {
		lb.Render.SetHTML(lb.Text, &lb.Sty.Font, &lb.Sty.Text, &lb.Sty.UnContext, lb.CSSAgg)
		lb.Render.LayoutStdLR(&lb.Sty.Text, &lb.Sty.Font, &lb.Sty.UnContext, sz)
		if lb.Render.Size.Y < (sz.Y - 1) { // allow for numerical issues
			// fmt.Printf("label layout less vert: %v  new: %v  prev: %v\n", lb.Nm, lb.Render.Size.Y, sz.Y)
			lb.LayData.SetFromStyle(&lb.Sty.Layout)
			lb.Size2DFromWH(lb.Render.Size.X, lb.Render.Size.Y)
			return true // needs a redo!
		}
	}
	return false
}

func (lb *Label) Render2D() {
	if lb.FullReRenderIfNeeded() {
		return
	}
	if lb.PushBounds() {
		lb.This.(Node2D).ConnectEvents2D()
		lb.GrabCurBgColor()
		lb.SetStateStyle()
		st := &lb.Sty
		rs := &lb.Viewport.Render
		lb.RenderPos = lb.TextPos()
		lb.RenderStdBox(st)
		lb.Render.Render(rs, lb.RenderPos)
		lb.Render2DChildren()
		lb.PopBounds()
	} else {
		lb.DisconnectAllEvents(RegPri)
	}
}

func (lb *Label) ConnectEvents2D() {
	lb.LabelEvents()
}

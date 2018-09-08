// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
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
// signal.
type Label struct {
	WidgetBase
	Text        string              `xml:"text" desc:"label to display"`
	Selectable  bool                `desc:"is this label selectable? if so, it will change background color in response to selection events and update selection state on mouse clicks"`
	Redrawable  bool                `desc:"is this label going to be redrawn frequently without an overall full re-render?  if so, you need to set this flag to avoid weird overlapping rendering results from antialiasing"`
	LinkSig     ki.Signal           `json:"-" xml:"-" view:"-" desc:"signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, opens default browser"`
	StateStyles [LabelStatesN]Style `json:"-" xml:"-" desc:"styles for different states of label"`
	Render      TextRender          `xml:"-" json:"-" desc:"render data for text label"`
	RenderPos   Vec2D               `xml:"-" json:"-" desc:"position offset of start of text rendering, from last render -- AllocPos plus alignment factors for center, right etc."`
	CurBgColor  Color               `xml:"-" json:"-" desc:"current background color -- grabbed when rendering for first time, and used when toggling off of selected mode, or for redrawable, to wipe out bg"`
}

var KiT_Label = kit.Types.AddType(&Label{}, LabelProps)

var LabelProps = ki.Props{
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
func (g *Label) SetText(txt string) {
	g.Text = txt
	g.Render.SetHTML(g.Text, &g.Sty.Font, &g.Sty.UnContext, g.CSSAgg)
	spc := g.Sty.BoxSpace()
	sz := g.LayData.AllocSize
	if sz.IsZero() {
		sz = g.LayData.SizePrefOrMax()
	}
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}
	g.Render.LayoutStdLR(&g.Sty.Text, &g.Sty.Font, &g.Sty.UnContext, sz)
}

// Label returns the display label for this node, satisfying the Labeler interface
func (g *Label) Label() string {
	if g.Text != "" {
		return g.Text
	}
	return g.Nm
}

// SetTextAction sets the text and triggers an update action
func (g *Label) SetTextAction(txt string) {
	g.SetText(txt)
	g.UpdateSig()
}

// SetStateStyle sets the style based on the inactive, selected flags
func (g *Label) SetStateStyle() {
	if g.IsInactive() {
		g.Sty = g.StateStyles[LabelInactive]
		if g.Redrawable && !g.CurBgColor.IsNil() {
			g.Sty.Font.BgColor.SetColor(g.CurBgColor)
		}
	} else if g.IsSelected() {
		g.Sty = g.StateStyles[LabelSelected]
	} else {
		g.Sty = g.StateStyles[LabelActive]
		if (g.Selectable || g.Redrawable) && !g.CurBgColor.IsNil() {
			g.Sty.Font.BgColor.SetColor(g.CurBgColor)
		}
	}
}

func (g *Label) Style2D() {
	g.Style2DWidget()
	if g.Sty.Text.Align != AlignLeft && g.Sty.Layout.AlignH == AlignLeft {
		g.Sty.Layout.AlignH = g.Sty.Text.Align // keep them consistent -- this is what people expect
	} else if g.Sty.Layout.AlignH != AlignLeft && g.Sty.Text.Align == AlignLeft {
		g.Sty.Text.Align = g.Sty.Layout.AlignH // keep them consistent -- this is what people expect
	}
	pst := g.ParentStyle()
	for i := 0; i < int(LabelStatesN); i++ {
		g.StateStyles[i].CopyFrom(&g.Sty)
		g.StateStyles[i].SetStyleProps(pst, g.StyleProps(LabelSelectors[i]))
		g.StateStyles[i].CopyUnitContext(&g.Sty.UnContext)
	}
	g.Render.SetHTML(g.Text, &(g.Sty.Font), &(g.Sty.UnContext), g.CSSAgg)
	spc := g.Sty.BoxSpace()
	sz := g.LayData.SizePrefOrMax()
	if !sz.IsZero() {
		sz.SetSubVal(2 * spc)
	}
	g.Render.LayoutStdLR(&(g.Sty.Text), &(g.Sty.Font), &(g.Sty.UnContext), sz)
}

func (g *Label) Size2D(iter int) {
	if iter > 0 && g.Sty.Text.WordWrap {
		return // already updated in previous iter, don't redo!
	} else {
		g.InitLayout2D()
		g.Size2DFromWH(g.Render.Size.X, g.Render.Size.Y)
	}
}

func (g *Label) Layout2D(parBBox image.Rectangle, iter int) bool {
	g.Layout2DBase(parBBox, true, iter)
	g.Layout2DChildren(iter) // todo: maybe shouldn't call this on known terminals?
	sz := g.Size2DSubSpace()
	if g.Sty.Text.WordWrap {
		g.Render.SetHTML(g.Text, &(g.Sty.Font), &(g.Sty.UnContext), g.CSSAgg)
		g.Render.LayoutStdLR(&(g.Sty.Text), &(g.Sty.Font), &(g.Sty.UnContext), sz)
		if g.Render.Size.Y < (sz.Y - 1) { // allow for numerical issues
			// fmt.Printf("label layout less vert: %v  new: %v  prev: %v\n", g.Nm, g.Render.Size.Y, sz.Y)
			g.LayData.SetFromStyle(&g.Sty.Layout)
			g.Size2DFromWH(g.Render.Size.X, g.Render.Size.Y)
			return true // needs a redo!
		}
	}
	return false
}

// OpenLink opens given link, either by sending LinkSig signal if there are
// receivers, or by opening in user's default browser (see oswin/App.OpenURL()
// method for more info)
func (g *Label) OpenLink(tl *TextLink) {
	if len(g.LinkSig.Cons) == 0 {
		oswin.TheApp.OpenURL(tl.URL)
		return
	}
	g.LinkSig.Emit(g.This, 0, tl.URL) // todo: could potentially signal different target=_blank kinds of options here with the sig
}

func (g *Label) LabelHoverEvent() {
	g.ConnectEvent(oswin.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		lb := recv.Embed(KiT_Label).(*Label)
		hasLinks := len(lb.Render.Links) > 0
		if hasLinks {
			pos := g.RenderPos
			for ti, _ := range lb.Render.Links {
				tl := &lb.Render.Links[ti]
				tlb := tl.Bounds(&lb.Render, pos)
				if me.Where.In(tlb) {
					PopupTooltip(tl.URL, tlb.Max.X, tlb.Max.Y, g.Viewport, lb.Nm)
					me.SetProcessed()
					return
				}
			}
		}
		if lb.Tooltip != "" {
			me.SetProcessed()
			pos := lb.WinBBox.Max
			pos.X -= 20
			PopupTooltip(lb.Tooltip, pos.X, pos.Y, g.Viewport, lb.Nm)
		}
	})
}

func (g *Label) LabelMouseEvent() {
	g.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		lb := recv.Embed(KiT_Label).(*Label)
		hasLinks := len(lb.Render.Links) > 0
		pos := g.RenderPos
		if lb.Selectable || hasLinks {
			if me.Action == mouse.Press && me.Button == mouse.Left {
				if hasLinks {
					for ti, _ := range lb.Render.Links {
						tl := &lb.Render.Links[ti]
						tlb := tl.Bounds(&lb.Render, pos)
						if me.Where.In(tlb) {
							lb.OpenLink(tl)
							me.SetProcessed()
							return
						}
					}
				}
				if lb.Selectable {
					lb.SetSelectedState(!lb.IsSelected())
					lb.EmitSelectedSignal()
					lb.UpdateSig()
				}
			}
		}
		if me.Action == mouse.Release && me.Button == mouse.Right {
			me.SetProcessed()
			lb.EmitContextMenuSignal()
			lb.This.(Node2D).ContextMenu()
		}
	})
}

func (g *Label) LabelMouseMoveEvent() {
	hasLinks := len(g.Render.Links) > 0
	if hasLinks {
		g.ConnectEvent(oswin.MouseMoveEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			me := d.(*mouse.MoveEvent)
			me.SetProcessed()
			lb := recv.Embed(KiT_Label).(*Label)
			pos := g.RenderPos
			inLink := false
			for _, tl := range lb.Render.Links {
				tlb := tl.Bounds(&lb.Render, pos)
				if me.Where.In(tlb) {
					inLink = true
				}
			}
			if inLink {
				oswin.TheApp.Cursor().PushIfNot(cursor.HandPointing)
			} else {
				oswin.TheApp.Cursor().PopIf(cursor.HandPointing)
			}
		})
	}
}

func (g *Label) ConnectEvents2D() {
	g.LabelHoverEvent()
	g.LabelMouseEvent()
	g.LabelMouseMoveEvent()
}

func (g *Label) GrabCurBgColor() {
	if g.Viewport == nil || g.IsSelected() {
		return
	}
	if !RebuildDefaultStyles && !g.CurBgColor.IsNil() {
		return
	}
	pos := g.ContextMenuPos()
	clr := g.Viewport.Pixels.At(pos.X, pos.Y)
	g.CurBgColor.SetColor(clr)
}

func (g *Label) TextPos() Vec2D {
	st := &g.Sty
	pos := g.LayData.AllocPos.AddVal(st.BoxSpace())
	if g.LayData.AllocSize.X > g.Render.Size.X {
		if IsAlignMiddle(st.Layout.AlignH) {
			pos.X += 0.5 * (g.LayData.AllocSize.X - g.Render.Size.X)
		} else if IsAlignEnd(st.Layout.AlignH) {
			pos.X += (g.LayData.AllocSize.X - g.Render.Size.X)
		}
	}
	if g.LayData.AllocSize.Y > g.Render.Size.Y {
		if IsAlignMiddle(st.Layout.AlignV) {
			pos.Y += 0.5 * (g.LayData.AllocSize.Y - g.Render.Size.Y)
		} else if IsAlignEnd(st.Layout.AlignV) {
			pos.Y += (g.LayData.AllocSize.Y - g.Render.Size.Y)
		}
	}
	return pos
}

func (g *Label) Render2D() {
	if g.FullReRenderIfNeeded() {
		return
	}
	if g.PushBounds() {
		g.This.(Node2D).ConnectEvents2D()
		g.GrabCurBgColor()
		g.SetStateStyle()
		st := &g.Sty
		rs := &g.Viewport.Render
		g.RenderPos = g.TextPos()
		g.RenderStdBox(st)
		g.Render.Render(rs, g.RenderPos)
		g.Render2DChildren()
		g.PopBounds()
	} else {
		g.DisconnectAllEvents(RegPri)
	}
}

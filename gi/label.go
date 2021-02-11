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
	Text        string                   `xml:"text" desc:"label to display"`
	Selectable  bool                     `desc:"is this label selectable? if so, it will change background color in response to selection events and update selection state on mouse clicks"`
	Redrawable  bool                     `desc:"is this label going to be redrawn frequently without an overall full re-render?  if so, you need to set this flag to avoid weird overlapping rendering results from antialiasing.  Also, if the label will change dynamically, this must be set to true, otherwise labels will illegibly overlay on top of each other."`
	LinkSig     ki.Signal                `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, calls TextLinkHandler then URLHandler"`
	StateStyles [LabelStatesN]gist.Style `copy:"-" json:"-" xml:"-" desc:"styles for different states of label"`
	Render      girl.Text                `copy:"-" xml:"-" json:"-" desc:"render data for text label"`
	RenderPos   mat32.Vec2               `copy:"-" xml:"-" json:"-" desc:"position offset of start of text rendering, from last render -- AllocPos plus alignment factors for center, right etc."`
	CurBgColor  gist.Color               `copy:"-" xml:"-" json:"-" desc:"current background color -- grabbed when rendering for first time, and used when toggling off of selected mode, or for redrawable, to wipe out bg"`
}

var KiT_Label = kit.Types.AddType(&Label{}, LabelProps)

// AddNewLabel adds a new label to given parent node, with given name and text.
func AddNewLabel(parent ki.Ki, name string, text string) *Label {
	lb := parent.AddNewChild(KiT_Label, name).(*Label)
	lb.Text = text
	return lb
}

func (lb *Label) CopyFieldsFrom(frm interface{}) {
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

var LabelProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
	"white-space":      gist.WhiteSpacePre, // no wrap, use spaces unless otherwise specified!
	"padding":          units.NewPx(2),
	"margin":           units.NewPx(2),
	"vertical-align":   gist.AlignTop,
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

var KiT_LabelStates = kit.Enums.AddEnumAltLower(LabelStatesN, kit.NotBitFlag, gist.StylePropProps, "Label")

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
	needSty := lb.Sty.Font.Size.Val == 0
	lb.StyMu.RUnlock()
	if needSty {
		lb.StyleLabel()
	}
	lb.SetStateStyle()
	lb.StyMu.RLock()
	lb.Text = txt
	lb.Sty.Font.BgColor.Color.SetToNil() // always use transparent bg for actual text
	// this makes it easier for it to update with dynamic bgs
	if lb.Text == "" {
		lb.Render.SetHTML(" ", &lb.Sty.Font, &lb.Sty.Text, &lb.Sty.UnContext, lb.CSSAgg)
	} else {
		lb.Render.SetHTML(lb.Text, &lb.Sty.Font, &lb.Sty.Text, &lb.Sty.UnContext, lb.CSSAgg)
	}
	spc := lb.BoxSpace()
	sz := lb.LayState.Alloc.Size
	if sz.IsNil() {
		sz = lb.LayState.SizePrefOrMax()
	}
	if !sz.IsNil() {
		sz.SetSubScalar(2 * spc)
	}
	lb.Render.LayoutStdLR(&lb.Sty.Text, &lb.Sty.Font, &lb.Sty.UnContext, sz)
	lb.StyMu.RUnlock()
	lb.UpdateEnd(updt)
}

// SetStateStyle sets the style based on the inactive, selected flags
func (lb *Label) SetStateStyle() {
	lb.StyMu.Lock()
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
	lb.ConnectEvent(oswin.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		llb := recv.Embed(KiT_Label).(*Label)
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
	lb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		llb := recv.Embed(KiT_Label).(*Label)
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

func (lb *Label) GrabCurBgColor() {
	if lb.Viewport == nil || lb.IsSelected() {
		return
	}
	if !gist.RebuildDefaultStyles && !lb.CurBgColor.IsNil() {
		return
	}
	pos := lb.ContextMenuPos()
	clr := lb.Viewport.Pixels.At(pos.X, pos.Y)
	lb.CurBgColor.SetColor(clr)
}

// StyleLabel does label styling -- it sets the StyMu Lock
func (lb *Label) StyleLabel() {
	lb.StyMu.Lock()
	defer lb.StyMu.Unlock()

	hasTempl, saveTempl := lb.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		lb.Style2DWidget()
	}
	if hasTempl && saveTempl {
		lb.Sty.SaveTemplate()
	}
	parSty := lb.ParentStyle()
	if hasTempl && !saveTempl {
		for i := 0; i < int(LabelStatesN); i++ {
			lb.StateStyles[i].Template = lb.Sty.Template + LabelSelectors[i]
			lb.StateStyles[i].FromTemplate()
		}
	} else {
		for i := 0; i < int(LabelStatesN); i++ {
			lb.StateStyles[i].CopyFrom(&lb.Sty)
			lb.StateStyles[i].SetStyleProps(parSty, lb.StyleProps(LabelSelectors[i]), lb.Viewport)
			lb.StateStyles[i].CopyUnitContext(&lb.Sty.UnContext)
		}
	}
	if hasTempl && saveTempl {
		for i := 0; i < int(LabelStatesN); i++ {
			lb.StateStyles[i].Template = lb.Sty.Template + LabelSelectors[i]
			lb.StateStyles[i].SaveTemplate()
		}
	}
	if lb.CurBgColor.IsNil() && !lb.Sty.Font.BgColor.Color.IsNil() {
		lb.CurBgColor = lb.Sty.Font.BgColor.Color
	}
	lb.ParentStyleRUnlock()
}

func (lb *Label) LayoutLabel() {
	lb.StyMu.RLock()
	defer lb.StyMu.RUnlock()

	lb.Sty.Font.BgColor.Color.SetToNil() // always use transparent bg for actual text
	lb.Render.SetHTML(lb.Text, &lb.Sty.Font, &lb.Sty.Text, &lb.Sty.UnContext, lb.CSSAgg)
	spc := lb.BoxSpace()
	sz := lb.LayState.SizePrefOrMax()
	if !sz.IsNil() {
		sz.SetSubScalar(2 * spc)
	}
	lb.Render.LayoutStdLR(&lb.Sty.Text, &lb.Sty.Font, &lb.Sty.UnContext, sz)
}

func (lb *Label) Style2D() {
	lb.StyleLabel()
	lb.StyMu.Lock()
	lb.LayState.SetFromStyle(&lb.Sty.Layout) // also does reset
	lb.StyMu.Unlock()
	lb.LayoutLabel()
}

func (lb *Label) Size2D(iter int) {
	if iter > 0 && lb.Sty.Text.HasWordWrap() {
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
		lb.StateStyles[i].CopyUnitContext(&lb.Sty.UnContext)
	}
	lb.Layout2DChildren(iter) // todo: maybe shouldn't call this on known terminals?
	sz := lb.Size2DSubSpace()
	lb.Sty.Font.BgColor.Color.SetToNil() // always use transparent bg for actual text
	lb.Render.SetHTML(lb.Text, &lb.Sty.Font, &lb.Sty.Text, &lb.Sty.UnContext, lb.CSSAgg)
	lb.Render.LayoutStdLR(&lb.Sty.Text, &lb.Sty.Font, &lb.Sty.UnContext, sz)
	if lb.Sty.Text.HasWordWrap() {
		if lb.Render.Size.Y < (sz.Y - 1) { // allow for numerical issues
			lb.LayState.SetFromStyle(&lb.Sty.Layout)
			lb.Size2DFromWH(lb.Render.Size.X, lb.Render.Size.Y)
			return true // needs a redo!
		}
	}
	return false
}

func (lb *Label) TextPos() mat32.Vec2 {
	lb.StyMu.RLock()
	sty := &lb.Sty
	pos := lb.LayState.Alloc.Pos.AddScalar(sty.BoxSpace())
	lb.StyMu.RUnlock()
	return pos
}

func (lb *Label) RenderLabel() {
	lb.GrabCurBgColor()
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

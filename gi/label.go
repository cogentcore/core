// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/cursors"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/mat32/v2"
)

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering, and full HTML styling, including links -- LinkSig
// emits link with data of URL -- opens default browser if nobody receiving
// signal.  The default white-space option is 'pre' -- set to 'normal' or
// other options to get word-wrapping etc.
type Label struct { //goki:embedder
	WidgetBase

	// label to display
	Text string `xml:"text"`

	// the type of label
	Type LabelTypes

	// [view: -] signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, calls TextLinkHandler then URLHandler
	// LinkSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for clicking on a link -- data is a string of the URL -- if nobody receiving this signal, calls TextLinkHandler then URLHandler"`

	// render data for text label
	TextRender paint.Text `copy:"-" xml:"-" json:"-" set:"-"`

	// position offset of start of text rendering, from last render -- AllocPos plus alignment factors for center, right etc.
	RenderPos mat32.Vec2 `copy:"-" xml:"-" json:"-" set:"-"`
}

func (lb *Label) CopyFieldsFrom(frm any) {
	fr := frm.(*Label)
	lb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	lb.Text = fr.Text
}

// LabelTypes is an enum containing the different
// possible types of labels
type LabelTypes int32 //enums:enum -trim-prefix Label

const (
	// LabelDisplayLarge is a large, short, and important
	// display label with a default font size of 57dp.
	LabelDisplayLarge LabelTypes = iota
	// LabelDisplayMedium is a medium-sized, short, and important
	// display label with a default font size of 45dp.
	LabelDisplayMedium
	// LabelDisplaySmall is a small, short, and important
	// display label with a default font size of 36dp.
	LabelDisplaySmall

	// LabelHeadlineLarge is a large, high-emphasis
	// headline label with a default font size of 32dp.
	LabelHeadlineLarge
	// LabelHeadlineMedium is a medium-sized, high-emphasis
	// headline label with a default font size of 28dp.
	LabelHeadlineMedium
	// LabelHeadlineSmall is a small, high-emphasis
	// headline label with a default font size of 24dp.
	LabelHeadlineSmall

	// LabelTitleLarge is a large, medium-emphasis
	// title label with a default font size of 22dp.
	LabelTitleLarge
	// LabelTitleMedium is a medium-sized, medium-emphasis
	// title label with a default font size of 16dp.
	LabelTitleMedium
	// LabelTitleSmall is a small, medium-emphasis
	// title label with a default font size of 14dp.
	LabelTitleSmall

	// LabelBodyLarge is a large body label used for longer
	// passages of text with a default font size of 16dp.
	LabelBodyLarge
	// LabelBodyMedium is a medium-sized body label used for longer
	// passages of text with a default font size of 14dp.
	LabelBodyMedium
	// LabelBodySmall is a small body label used for longer
	// passages of text with a default font size of 12dp.
	LabelBodySmall

	// LabelLabelLarge is a large label used for label text (like a caption
	// or the text inside a button) with a default font size of 14dp.
	LabelLabelLarge
	// LabelLabelMedium is a medium-sized label used for label text (like a caption
	// or the text inside a button) with a default font size of 12dp.
	LabelLabelMedium
	// LabelLabelSmall is a small label used for label text (like a caption
	// or the text inside a button) with a default font size of 11dp.
	LabelLabelSmall
)

func (lb *Label) OnInit() {
	lb.HandleLabelEvents()
	lb.LabelStyles()
}

func (lb *Label) LabelStyles() {
	lb.Type = LabelBodyLarge
	lb.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Selectable, abilities.DoubleClickable)
		if !lb.IsReadOnly() {
			s.Cursor = cursors.Text
		}

		s.Text.WhiteSpace = styles.WhiteSpaceNormal
		s.AlignV = styles.AlignMiddle
		s.SetStretchMaxWidth() // critical for avoiding excessive word wrapping
		// Label styles based on https://m3.material.io/styles/typography/type-scale-tokens
		// TODO: maybe support brand and plain global fonts with larger labels defaulting to brand and smaller to plain
		switch lb.Type {
		case LabelLabelLarge:
			s.Text.LineHeight.Dp(20)
			s.Font.Size.Dp(14)
			s.Text.LetterSpacing.Dp(0.1)
			s.Font.Weight = styles.WeightMedium
		case LabelLabelMedium:
			s.Text.LineHeight.Dp(16)
			s.Font.Size.Dp(12)
			s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = styles.WeightMedium
		case LabelLabelSmall:
			s.Text.LineHeight.Dp(16)
			s.Font.Size.Dp(11)
			s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = styles.WeightMedium
		case LabelBodyLarge:
			s.Text.LineHeight.Dp(24)
			s.Font.Size.Dp(16)
			s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = styles.WeightNormal
		case LabelBodyMedium:
			s.Text.LineHeight.Dp(20)
			s.Font.Size.Dp(14)
			s.Text.LetterSpacing.Dp(0.25)
			s.Font.Weight = styles.WeightNormal
		case LabelBodySmall:
			s.Text.LineHeight.Dp(16)
			s.Font.Size.Dp(12)
			s.Text.LetterSpacing.Dp(0.4)
			s.Font.Weight = styles.WeightNormal
		case LabelTitleLarge:
			s.Text.LineHeight.Dp(28)
			s.Font.Size.Dp(22)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case LabelTitleMedium:
			s.Text.LineHeight.Dp(24)
			s.Font.Size.Dp(16)
			s.Text.LetterSpacing.Dp(0.15)
			s.Font.Weight = styles.WeightMedium
		case LabelTitleSmall:
			s.Text.LineHeight.Dp(20)
			s.Font.Size.Dp(14)
			s.Text.LetterSpacing.Dp(0.1)
			s.Font.Weight = styles.WeightMedium
		case LabelHeadlineLarge:
			s.Text.LineHeight.Dp(40)
			s.Font.Size.Dp(32)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case LabelHeadlineMedium:
			s.Text.LineHeight.Dp(36)
			s.Font.Size.Dp(28)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case LabelHeadlineSmall:
			s.Text.LineHeight.Dp(32)
			s.Font.Size.Dp(24)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case LabelDisplayLarge:
			s.Text.LineHeight.Dp(64)
			s.Font.Size.Dp(57)
			s.Text.LetterSpacing.Dp(-0.25)
			s.Font.Weight = styles.WeightNormal
		case LabelDisplayMedium:
			s.Text.LineHeight.Dp(52)
			s.Font.Size.Dp(45)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case LabelDisplaySmall:
			s.Text.LineHeight.Dp(44)
			s.Font.Size.Dp(36)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		}
	})
}

// OpenLink opens given link, either by sending LinkSig signal if there are
// receivers, or by calling the TextLinkHandler if non-nil, or URLHandler if
// non-nil (which by default opens user's default browser via
// oswin/App.OpenURL())
func (lb *Label) OpenLink(tl *paint.TextLink) {
	// tl.Widget = lb.This() // todo: needs this
	// if len(lb.LinkSig.Cons) == 0 {
	// 	if paint.TextLinkHandler != nil {
	// 		if paint.TextLinkHandler(*tl) {
	// 			return
	// 		}
	// 	}
	// 	if paint.URLHandler != nil {
	// 		paint.URLHandler(tl.URL)
	// 	}
	// 	return
	// }
	// lb.LinkSig.Emit(lb.This(), 0, tl.URL) // todo: could potentially signal different target=_blank kinds of options here with the sig
}

// func (lb *Label) HandleEvent(ev events.Event) {
// 	// hasLinks := len(lb.TextRender.Links) > 0
// 	// if !hasLinks {
// 	// 	lb.Events.Ex(events.MouseMove)
// 	// }
// }

func (lb *Label) HandleLabelEvents() {
	lb.HandleWidgetEvents()
	lb.HandleLabelLongHover()
	lb.HandleLabelClick()
	lb.HandleLabelMouseMove()
	lb.HandleLabelKeys()
}

func (lb *Label) HandleLabelLongHover() {
	lb.On(events.LongHoverStart, func(e events.Event) {
		// hasLinks := len(lb.TextRender.Links) > 0
		// if hasLinks {
		// 	pos := llb.RenderPos
		// 	for ti := range llb.TextRender.Links {
		// 		tl := &llb.TextRender.Links[ti]
		// 		tlb := tl.Bounds(&llb.TextRender, pos)
		// 		if me.LocalPos().In(tlb) {
		// 			PopupTooltip(tl.URL, tlb.Max.X, tlb.Max.Y, llb.Sc, llb.Nm)
		// 			me.SetHandled()
		// 			return
		// 		}
		// 	}
		// }
		/*
			todo:
			if llb.Tooltip != "" {
				me.SetHandled()
				llb.BBoxMu.RLock()
				pos := llb.WinBBox.Max
				llb.BBoxMu.RUnlock()
				pos.X -= 20
				PopupTooltip(llb.Tooltip, pos.X, pos.Y, llb.Sc, llb.Nm)
			}
		*/
	})
}

func (lb *Label) HandleLabelClick() {
	lb.OnClick(func(e events.Event) {
		hasLinks := len(lb.TextRender.Links) > 0
		if !hasLinks {
			return
		}
		pos := lb.RenderPos
		for ti := range lb.TextRender.Links {
			tl := &lb.TextRender.Links[ti]
			tlb := tl.Bounds(&lb.TextRender, pos)
			if e.LocalPos().In(tlb) {
				lb.OpenLink(tl)
				e.SetHandled()
				return
			}
		}
	})
}

func (lb *Label) HandleLabelMouseMove() {
	lb.On(events.MouseMove, func(e events.Event) {
		pos := lb.RenderPos
		inLink := false
		for _, tl := range lb.TextRender.Links {
			tlb := tl.Bounds(&lb.TextRender, pos)
			if e.LocalPos().In(tlb) {
				inLink = true
				break
			}
		}
		_ = inLink
		/*
			// TODO: figure out how to get links to work with new cursor setup
			if inLink {
				goosi.TheApp.Cursor(lb.ParentRenderWin().RenderWin).PushIfNot(cursors.Pointer)
			} else {
				goosi.TheApp.Cursor(lb.ParentRenderWin().RenderWin).PopIf(cursors.Pointer)
			}
		*/
	})
}

func (lb *Label) HandleLabelKeys() {
	lb.OnKeyChord(func(e events.Event) {
		// TODO(kai): get label copying working
		fmt.Println("kc", e)
		if !lb.StateIs(states.Selected) {
			return
		}
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Copy {
			e.SetHandled()
			md := mimedata.NewText(lb.Text)
			lb.This().(Clipper).MimeData(&md)
			lb.This().(Clipper).Copy(true)
			fmt.Println("cp", md)
		}
	})
}

// StyleLabel does label styling -- it sets the StyMu Lock
func (lb *Label) StyleLabel(sc *Scene) {
	lb.StyMu.Lock()
	defer lb.StyMu.Unlock()

	lb.ApplyStyleWidget(sc)
}

func (lb *Label) LayoutLabel(sc *Scene) {
	lb.StyMu.RLock()
	defer lb.StyMu.RUnlock()

	lb.TextRender.SetHTML(lb.Text, lb.Styles.FontRender(), &lb.Styles.Text, &lb.Styles.UnContext, lb.CSSAgg)
	spc := lb.BoxSpace()
	sz := lb.LayState.SizePrefOrMax()
	if LayoutTrace {
		fmt.Println("Label:", lb.Nm, "LayoutLabel Size:", sz)
	}
	if !sz.IsNil() {
		sz.SetSub(spc.Size())
	}
	lb.TextRender.LayoutStdLR(&lb.Styles.Text, lb.Styles.FontRender(), &lb.Styles.UnContext, sz)
}

func (lb *Label) ApplyStyle(sc *Scene) {
	lb.StyleLabel(sc)
	lb.LayoutLabel(sc)
}

func (lb *Label) GetSize(sc *Scene, iter int) {
	if iter > 0 && lb.Styles.Text.HasWordWrap() {
		return // already updated in previous iter, don't redo!
	} else {
		lb.InitLayout(sc)
		sz := lb.LayState.Size.Pref // SizePrefOrMax()
		sz = sz.Max(lb.TextRender.Size)
		lb.GetSizeFromWH(sz.X, sz.Y)
		if LayoutTrace {
			fmt.Println("Label:", lb.Nm, "GetSize:", sz)
		}
	}
}

func (lb *Label) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	lb.DoLayoutBase(sc, parBBox, iter)
	lb.DoLayoutChildren(sc, iter) // todo: maybe shouldn't call this on known terminals?
	sz := lb.GetSizeSubSpace()
	lb.TextRender.SetHTML(lb.Text, lb.Styles.FontRender(), &lb.Styles.Text, &lb.Styles.UnContext, lb.CSSAgg)
	lb.TextRender.LayoutStdLR(&lb.Styles.Text, lb.Styles.FontRender(), &lb.Styles.UnContext, sz)
	if lb.Styles.Text.HasWordWrap() {
		if lb.TextRender.Size.Y < (sz.Y - 1) { // allow for numerical issues
			lb.LayState.SetFromStyle(&lb.Styles) // todo: revisit!!
			lb.GetSizeFromWH(lb.TextRender.Size.X, lb.TextRender.Size.Y)
			return true // needs a redo!
		}
	}
	return false
}

func (lb *Label) TextPos() mat32.Vec2 {
	lb.StyMu.RLock()
	pos := lb.LayState.Alloc.Pos.Add(lb.Styles.BoxSpace().Pos())
	lb.StyMu.RUnlock()
	return pos
}

func (lb *Label) RenderLabel(sc *Scene) {
	rs, _, st := lb.RenderLock(sc)
	defer lb.RenderUnlock(rs)
	lb.RenderPos = lb.TextPos()
	lb.RenderStdBox(sc, st)
	lb.TextRender.Render(rs, lb.RenderPos)
}

func (lb *Label) Render(sc *Scene) {
	if lb.PushBounds(sc) {
		lb.RenderLabel(sc)
		lb.RenderChildren(sc)
		lb.PopBounds(sc)
	}
}

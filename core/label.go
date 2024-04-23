// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
)

// Text is a widget for rendering text. It supports full HTML styling,
// including links. By default, labels wrap and collapse whitespace, although
// you can change this by changing [styles.Text.WhiteSpace].
type Text struct { //core:embedder
	WidgetBase

	// Text is the text to display.
	Text string

	// Type is the styling type of label to use.
	Type TextTypes

	// paintText is the [paint.Text] for the text.
	paintText paint.Text

	// normalCursor is the cached cursor to display when there
	// is no link being hovered.
	normalCursor cursors.Cursor
}

// TextTypes is an enum containing the different
// possible styling types of [Text] widgets.
type TextTypes int32 //enums:enum -trim-prefix Text

const (
	// TextDisplayLarge is large, short, and important
	// display text with a default font size of 57dp.
	TextDisplayLarge TextTypes = iota
	// TextDisplayMedium is medium-sized, short, and important
	// display text with a default font size of 45dp.
	TextDisplayMedium
	// TextDisplaySmall is small, short, and important
	// display text with a default font size of 36dp.
	TextDisplaySmall

	// TextHeadlineLarge is large, high-emphasis
	// headline text with a default font size of 32dp.
	TextHeadlineLarge
	// TextHeadlineMedium is medium-sized, high-emphasis
	// headline text with a default font size of 28dp.
	TextHeadlineMedium
	// TextHeadlineSmall is small, high-emphasis
	// headline text with a default font size of 24dp.
	TextHeadlineSmall

	// TextTitleLarge is large, medium-emphasis
	// title text with a default font size of 22dp.
	TextTitleLarge
	// TextTitleMedium is medium-sized, medium-emphasis
	// title text with a default font size of 16dp.
	TextTitleMedium
	// TextTitleSmall is small, medium-emphasis
	// title text with a default font size of 14dp.
	TextTitleSmall

	// TextBodyLarge is large body text used for longer
	// passages of text with a default font size of 16dp.
	TextBodyLarge
	// TextBodyMedium is medium-sized body text used for longer
	// passages of text with a default font size of 14dp.
	TextBodyMedium
	// TextBodySmall is small body text used for longer
	// passages of text with a default font size of 12dp.
	TextBodySmall

	// TextLabelLarge is large text used for label text (like a caption
	// or the text inside a button) with a default font size of 14dp.
	TextLabelLarge
	// TextLabelMedium is medium-sized text used for label text (like a caption
	// or the text inside a button) with a default font size of 12dp.
	TextLabelMedium
	// TextLabelSmall is small text used for label text (like a caption
	// or the text inside a button) with a default font size of 11dp.
	TextLabelSmall
)

func (lb *Text) OnInit() {
	lb.WidgetBase.OnInit()
	lb.HandleEvents()
	lb.SetStyles()
}

func (lb *Text) SetStyles() {
	lb.Type = TextBodyLarge
	lb.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Selectable, abilities.DoubleClickable)
		if len(lb.paintText.Links) > 0 {
			s.SetAbilities(true, abilities.Clickable, abilities.LongHoverable, abilities.LongPressable)
		}
		if !lb.IsReadOnly() {
			s.Cursor = cursors.Text
		}
		s.GrowWrap = true

		// Label styles based on https://m3.material.io/styles/typography/type-scale-tokens
		switch lb.Type {
		case TextLabelLarge:
			s.Text.LineHeight.Dp(20)
			s.Font.Size.Dp(14)
			s.Text.LetterSpacing.Dp(0.1)
			s.Font.Weight = styles.WeightMedium // note: excludes all fonts except Go!
		case TextLabelMedium:
			s.Text.LineHeight.Dp(16)
			s.Font.Size.Dp(12)
			s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = styles.WeightMedium
		case TextLabelSmall:
			s.Text.LineHeight.Dp(16)
			s.Font.Size.Dp(11)
			s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = styles.WeightMedium
		case TextBodyLarge:
			s.Text.LineHeight.Dp(24)
			s.Font.Size.Dp(16)
			s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = styles.WeightNormal
		case TextBodyMedium:
			s.Text.LineHeight.Dp(20)
			s.Font.Size.Dp(14)
			s.Text.LetterSpacing.Dp(0.25)
			s.Font.Weight = styles.WeightNormal
		case TextBodySmall:
			s.Text.LineHeight.Dp(16)
			s.Font.Size.Dp(12)
			s.Text.LetterSpacing.Dp(0.4)
			s.Font.Weight = styles.WeightNormal
		case TextTitleLarge:
			s.Text.LineHeight.Dp(28)
			s.Font.Size.Dp(22)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case TextTitleMedium:
			s.Text.LineHeight.Dp(24)
			s.Font.Size.Dp(16)
			s.Text.LetterSpacing.Dp(0.15)
			s.Font.Weight = styles.WeightMedium
		case TextTitleSmall:
			s.Text.LineHeight.Dp(20)
			s.Font.Size.Dp(14)
			s.Text.LetterSpacing.Dp(0.1)
			s.Font.Weight = styles.WeightMedium
		case TextHeadlineLarge:
			s.Text.LineHeight.Dp(40)
			s.Font.Size.Dp(32)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case TextHeadlineMedium:
			s.Text.LineHeight.Dp(36)
			s.Font.Size.Dp(28)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case TextHeadlineSmall:
			s.Text.LineHeight.Dp(32)
			s.Font.Size.Dp(24)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case TextDisplayLarge:
			s.Text.LineHeight.Dp(64)
			s.Font.Size.Dp(57)
			s.Text.LetterSpacing.Dp(-0.25)
			s.Font.Weight = styles.WeightNormal
		case TextDisplayMedium:
			s.Text.LineHeight.Dp(52)
			s.Font.Size.Dp(45)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		case TextDisplaySmall:
			s.Text.LineHeight.Dp(44)
			s.Font.Size.Dp(36)
			s.Text.LetterSpacing.Zero()
			s.Font.Weight = styles.WeightNormal
		}
	})
	lb.StyleFinal(func(s *styles.Style) {
		lb.normalCursor = s.Cursor
	})
}

func (lb *Text) HandleEvents() {
	lb.HandleLabelClick(func(tl *paint.TextLink) {
		system.TheApp.OpenURL(tl.URL)
	})
	lb.OnDoubleClick(func(e events.Event) {
		lb.SetSelected(true)
		lb.SetFocus()
	})
	lb.OnFocusLost(func(e events.Event) {
		lb.SetSelected(false)
	})
	lb.OnKeyChord(func(e events.Event) {
		// TODO(kai): get label copying working
		if !lb.StateIs(states.Selected) {
			return
		}
		kf := keymap.Of(e.KeyChord())
		if kf == keymap.Copy {
			e.SetHandled()
			lb.Copy(true)
		}
	})
	lb.On(events.MouseMove, func(e events.Event) {
		pos := lb.Geom.Pos.Content
		inLink := false
		for _, tl := range lb.paintText.Links {
			// TODO(kai/link): is there a better way to be safe here?
			if tl.Label == "" {
				continue
			}
			tlb := tl.Bounds(&lb.paintText, pos)
			if e.Pos().In(tlb) {
				inLink = true
				if lb.StateIs(states.LongHovered) || lb.StateIs(states.LongPressed) {
					NewTooltipTextAt(lb, tl.URL, tlb.Min, tlb.Size()).Run()
					e.SetHandled()
				}
				break
			}
		}
		if inLink {
			lb.Styles.Cursor = cursors.Pointer
		} else {
			lb.Styles.Cursor = lb.normalCursor
		}
	})
}

// HandleLabelClick handles click events such that the given function will be called
// on any links that are clicked on.
func (lb *Text) HandleLabelClick(openLink func(tl *paint.TextLink)) {
	lb.OnClick(func(e events.Event) {
		pos := lb.Geom.Pos.Content
		for _, tl := range lb.paintText.Links {
			// TODO(kai/link): is there a better way to be safe here?
			if tl.Label == "" {
				continue
			}
			tlb := tl.Bounds(&lb.paintText, pos)
			if e.Pos().In(tlb) {
				openLink(&tl)
				e.SetHandled()
				return
			}
		}
	})
}

func (lb *Text) Copy(reset bool) {
	md := mimedata.NewText(lb.Text)
	em := lb.Events()
	if em != nil {
		em.Clipboard().Write(md)
	}
}

func (lb *Text) Label() string {
	if lb.Text != "" {
		return lb.Text
	}
	return lb.Nm
}

// todo: ideally it would be possible to only call SetHTML once during config
// and then do the layout only during sizing.  However, layout starts with
// existing line breaks (which could come from <br> and <p> in HTML),
// so that is never able to undo initial word wrapping from constrained sizes.

// ConfigLabel does the HTML and Layout in TextRender for label text,
// using actual content size to constrain layout.
func (lb *Text) Config() {
	lb.ConfigLabelSize(lb.Geom.Size.Actual.Content)
}

// ConfigLabel does the HTML and Layout in TextRender for label text,
// using given size to constrain layout.
func (lb *Text) ConfigLabelSize(sz math32.Vector2) {
	// todo: last arg is CSSAgg.  Can synthesize that some other way?
	fs := lb.Styles.FontRender()
	txs := &lb.Styles.Text
	lb.paintText.SetHTML(lb.Text, fs, txs, &lb.Styles.UnitContext, nil)
	lb.paintText.Layout(txs, fs, &lb.Styles.UnitContext, sz)
}

// ConfigLabelAlloc is used for determining how much space the label
// takes, using given size (typically Alloc).
// In this case, alignment factors are turned off,
// because they otherwise can absorb much more space, which should
// instead be controlled by the base Align X,Y factors.
func (lb *Text) ConfigLabelAlloc(sz math32.Vector2) math32.Vector2 {
	// todo: last arg is CSSAgg.  Can synthesize that some other way?
	fs := lb.Styles.FontRender()
	txs := &lb.Styles.Text
	align, alignV := txs.Align, txs.AlignV
	txs.Align, txs.AlignV = styles.Start, styles.Start
	lb.paintText.SetHTML(lb.Text, fs, txs, &lb.Styles.UnitContext, nil)
	lb.paintText.Layout(txs, fs, &lb.Styles.UnitContext, sz)
	rsz := lb.paintText.Size.Ceil()
	txs.Align, txs.AlignV = align, alignV
	lb.paintText.Layout(txs, fs, &lb.Styles.UnitContext, rsz)
	return rsz
}

// TextWrapSizeEstimate is the size to use for layout during the SizeUp pass,
// for word wrap case, where the sizing actually matters,
// based on trying to fit the given number of characters into the given content size
// with given font height.
func TextWrapSizeEstimate(csz math32.Vector2, nChars int, fs *styles.Font) math32.Vector2 {
	chars := float32(nChars)
	fht := float32(16)
	if fs.Face != nil {
		fht = fs.Face.Metrics.Height
	}
	area := chars * fht * fht
	ratio := float32(1.618) // default to golden
	if csz.X > 0 && csz.Y > 0 {
		ratio = csz.X / csz.Y
		// fmt.Println(lb, "content size ratio:", ratio)
	}
	// w = ratio * h
	// w^2 + h^2 = a
	// (ratio*h)^2 + h^2 = a
	h := math32.Sqrt(area) / math32.Sqrt(ratio+1)
	w := ratio * h
	if w < csz.X { // must be at least this
		w = csz.X
		h = area / w
		h = max(h, csz.Y)
	}
	sz := math32.Vec2(w, h)
	if DebugSettings.LayoutTrace {
		fmt.Println("TextWrapSizeEstimate chars:", chars, "area:", area, "sz:", sz)
	}
	return sz
}

func (lb *Text) SizeUp() {
	lb.WidgetBase.SizeUp() // sets Actual size based on styles
	sz := &lb.Geom.Size
	if lb.Styles.Text.HasWordWrap() {
		lb.ConfigLabelSize(TextWrapSizeEstimate(lb.Geom.Size.Actual.Content, len(lb.Text), &lb.Styles.Font))
	} else {
		lb.ConfigLabelSize(sz.Actual.Content)
	}
	rsz := lb.paintText.Size.Ceil()
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.SetTotalFromContent(&sz.Actual)
	if DebugSettings.LayoutTrace {
		fmt.Println(lb, "Label SizeUp:", rsz, "Actual:", sz.Actual.Content)
	}
}

func (lb *Text) SizeDown(iter int) bool {
	if !lb.Styles.Text.HasWordWrap() || iter > 1 {
		return false
	}
	sz := &lb.Geom.Size
	rsz := lb.ConfigLabelAlloc(sz.Alloc.Content) // use allocation
	prevContent := sz.Actual.Content
	// start over so we don't reflect hysteresis of prior guess
	sz.SetInitContentMin(lb.Styles.Min.Dots().Ceil())
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.SetTotalFromContent(&sz.Actual)
	chg := prevContent != sz.Actual.Content
	if chg {
		if DebugSettings.LayoutTrace {
			fmt.Println(lb, "Label Size Changed:", sz.Actual.Content, "was:", prevContent)
		}
	}
	return chg
}

func (lb *Text) Render() {
	lb.RenderStandardBox()
	lb.paintText.Render(&lb.Scene.PaintContext, lb.Geom.Pos.Content)
}

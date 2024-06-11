// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/system"
)

// Text is a widget for rendering text. It supports full HTML styling,
// including links. By default, text wraps and collapses whitespace, although
// you can change this by changing [styles.Text.WhiteSpace].
type Text struct {
	WidgetBase

	// Text is the text to display.
	Text string

	// Type is the styling type of text to use.
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

func (tx *Text) WidgetValue() any { return &tx.Text }

func (tx *Text) Init() {
	tx.WidgetBase.Init()
	tx.SetType(TextBodyLarge)
	tx.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Selectable, abilities.DoubleClickable)
		if len(tx.paintText.Links) > 0 {
			s.SetAbilities(true, abilities.Clickable, abilities.LongHoverable, abilities.LongPressable)
		}
		if !tx.IsReadOnly() {
			s.Cursor = cursors.Text
		}
		s.GrowWrap = true

		// Text styles based on https://m3.material.io/styles/typography/type-scale-tokens
		switch tx.Type {
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
	tx.FinalStyler(func(s *styles.Style) {
		tx.normalCursor = s.Cursor
	})

	tx.HandleTextClick(func(tl *paint.TextLink) {
		system.TheApp.OpenURL(tl.URL)
	})
	tx.OnDoubleClick(func(e events.Event) {
		tx.SetSelected(true)
		tx.SetFocus()
	})
	tx.OnFocusLost(func(e events.Event) {
		tx.SetSelected(false)
	})
	tx.OnKeyChord(func(e events.Event) {
		if !tx.StateIs(states.Selected) {
			return
		}
		kf := keymap.Of(e.KeyChord())
		if kf == keymap.Copy {
			e.SetHandled()
			tx.Copy(true)
		}
	})
	tx.On(events.MouseMove, func(e events.Event) {
		tl, _ := tx.FindLink(e.Pos())
		if tl != nil {
			tx.Styles.Cursor = cursors.Pointer
		} else {
			tx.Styles.Cursor = tx.normalCursor
		}
	})

	// todo: ideally it would be possible to only call SetHTML once during config
	// and then do the layout only during sizing.  However, layout starts with
	// existing line breaks (which could come from <br> and <p> in HTML),
	// so that is never able to undo initial word wrapping from constrained sizes.
	tx.Updater(func() {
		tx.configTextSize(tx.Geom.Size.Actual.Content)
	})
}

// FindLink finds the text link at the given scene-local position. If it
// finds it, it returns it and its bounds; otherwise, it returns nil.
func (tx *Text) FindLink(pos image.Point) (*paint.TextLink, image.Rectangle) {
	for _, tl := range tx.paintText.Links {
		// TODO(kai/link): is there a better way to be safe here?
		if tl.Label == "" {
			continue
		}
		tlb := tl.Bounds(&tx.paintText, tx.Geom.Pos.Content)
		if pos.In(tlb) {
			return &tl, tlb
		}
	}
	return nil, image.Rectangle{}
}

// HandleTextClick handles click events such that the given function will be called
// on any links that are clicked on.
func (tx *Text) HandleTextClick(openLink func(tl *paint.TextLink)) {
	tx.OnClick(func(e events.Event) {
		tl, _ := tx.FindLink(e.Pos())
		if tl == nil {
			return
		}
		openLink(tl)
		e.SetHandled()
	})
}

func (tx *Text) WidgetTooltip(pos image.Point) (string, image.Point) {
	if pos == image.Pt(-1, -1) {
		return tx.Tooltip, image.Point{}
	}
	tl, bounds := tx.FindLink(pos)
	if tl == nil {
		return tx.Tooltip, tx.DefaultTooltipPos()
	}
	return tl.URL, bounds.Min
}

func (tx *Text) Copy(reset bool) {
	md := mimedata.NewText(tx.Text)
	em := tx.Events()
	if em != nil {
		em.Clipboard().Write(md)
	}
}

func (tx *Text) Label() string {
	if tx.Text != "" {
		return tx.Text
	}
	return tx.Name
}

// configTextSize does the HTML and Layout in paintText for text,
// using given size to constrain layout.
func (tx *Text) configTextSize(sz math32.Vector2) {
	// todo: last arg is CSSAgg.  Can synthesize that some other way?
	fs := tx.Styles.FontRender()
	txs := &tx.Styles.Text
	tx.paintText.SetHTML(tx.Text, fs, txs, &tx.Styles.UnitContext, nil)
	tx.paintText.Layout(txs, fs, &tx.Styles.UnitContext, sz)
}

// configTextAlloc is used for determining how much space the text
// takes, using given size (typically Alloc).
// In this case, alignment factors are turned off,
// because they otherwise can absorb much more space, which should
// instead be controlled by the base Align X,Y factors.
func (tx *Text) configTextAlloc(sz math32.Vector2) math32.Vector2 {
	// todo: last arg is CSSAgg.  Can synthesize that some other way?
	fs := tx.Styles.FontRender()
	txs := &tx.Styles.Text
	align, alignV := txs.Align, txs.AlignV
	txs.Align, txs.AlignV = styles.Start, styles.Start
	tx.paintText.SetHTML(tx.Text, fs, txs, &tx.Styles.UnitContext, nil)
	tx.paintText.Layout(txs, fs, &tx.Styles.UnitContext, sz)
	rsz := tx.paintText.BBox.Size().Ceil()
	txs.Align, txs.AlignV = align, alignV
	tx.paintText.Layout(txs, fs, &tx.Styles.UnitContext, rsz)
	return rsz
}

// textWrapSizeEstimate is the size to use for layout during the SizeUp pass,
// for word wrap case, where the sizing actually matters,
// based on trying to fit the given number of characters into the given content size
// with given font height.
func textWrapSizeEstimate(csz math32.Vector2, nChars int, fs *styles.Font) math32.Vector2 {
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

func (tx *Text) SizeUp() {
	tx.WidgetBase.SizeUp() // sets Actual size based on styles
	sz := &tx.Geom.Size
	if tx.Styles.Text.HasWordWrap() {
		tx.configTextSize(textWrapSizeEstimate(tx.Geom.Size.Actual.Content, len(tx.Text), &tx.Styles.Font))
	} else {
		tx.configTextSize(sz.Actual.Content)
	}
	rsz := tx.paintText.BBox.Size().Ceil()
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.SetTotalFromContent(&sz.Actual)
	if DebugSettings.LayoutTrace {
		fmt.Println(tx, "Label SizeUp:", rsz, "Actual:", sz.Actual.Content)
	}
}

func (tx *Text) SizeDown(iter int) bool {
	if !tx.Styles.Text.HasWordWrap() || iter > 1 {
		return false
	}
	sz := &tx.Geom.Size
	rsz := tx.configTextAlloc(sz.Alloc.Content) // use allocation
	prevContent := sz.Actual.Content
	// start over so we don't reflect hysteresis of prior guess
	sz.SetInitContentMin(tx.Styles.Min.Dots().Ceil())
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.SetTotalFromContent(&sz.Actual)
	chg := prevContent != sz.Actual.Content
	if chg {
		if DebugSettings.LayoutTrace {
			fmt.Println(tx, "Label Size Changed:", sz.Actual.Content, "was:", prevContent)
		}
	}
	return chg
}

func (tx *Text) Render() {
	tx.WidgetBase.Render()
	tx.paintText.Render(&tx.Scene.PaintContext, tx.Geom.Pos.Content)
}

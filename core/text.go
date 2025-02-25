// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/htmltext"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
)

// Text is a widget for rendering text. It supports full HTML styling,
// including links. By default, text wraps and collapses whitespace, although
// you can change this by changing [styles.Text.WhiteSpace].
type Text struct {
	WidgetBase

	// Text is the text to display.
	Text string

	// Type is the styling type of text to use.
	// It defaults to [TextBodyLarge].
	Type TextTypes

	// Links is the list of links in the text.
	Links []rich.Hyperlink

	// richText is the conversion of the HTML text source.
	richText rich.Text

	// paintText is the [shaped.Lines] for the text.
	paintText *shaped.Lines

	// normalCursor is the cached cursor to display when there
	// is no link being hovered.
	normalCursor cursors.Cursor

	// selectRange is the selected range.
	selectRange textpos.Range
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

	// TextSupporting is medium-sized supporting text typically used for
	// secondary dialog information below the title. It has a default font
	// size of 14dp and color of [colors.Scheme.OnSurfaceVariant].
	TextSupporting
)

func (tx *Text) WidgetValue() any { return &tx.Text }

func (tx *Text) Init() {
	tx.WidgetBase.Init()
	tx.SetType(TextBodyLarge)
	tx.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Selectable, abilities.Slideable, abilities.DoubleClickable, abilities.TripleClickable)
		if len(tx.Links) > 0 {
			s.SetAbilities(true, abilities.Clickable, abilities.LongHoverable, abilities.LongPressable)
		}
		if !tx.IsReadOnly() {
			s.Cursor = cursors.Text
		}
		s.GrowWrap = true

		// Text styles based on https://m3.material.io/styles/typography/type-scale-tokens
		// We use Em for line height so that it scales properly with font size changes.
		switch tx.Type {
		case TextLabelLarge:
			s.Text.LineSpacing = 20.0 / 14
			s.Text.FontSize.Dp(14)
			// s.Text.LetterSpacing.Dp(0.1)
			s.Font.Weight = rich.Medium
		case TextLabelMedium:
			s.Text.LineSpacing = 16.0 / 12
			s.Text.FontSize.Dp(12)
			// s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = rich.Medium
		case TextLabelSmall:
			s.Text.LineSpacing = 16.0 / 11
			s.Text.FontSize.Dp(11)
			// s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = rich.Medium
		case TextBodyLarge:
			s.Text.LineSpacing = 24.0 / 16
			s.Text.FontSize.Dp(16)
			// s.Text.LetterSpacing.Dp(0.5)
			s.Font.Weight = rich.Normal
		case TextSupporting:
			s.Color = colors.Scheme.OnSurfaceVariant
			fallthrough
		case TextBodyMedium:
			s.Text.LineSpacing = 20.0 / 14
			s.Text.FontSize.Dp(14)
			// s.Text.LetterSpacing.Dp(0.25)
			s.Font.Weight = rich.Normal
		case TextBodySmall:
			s.Text.LineSpacing = 16.0 / 12
			s.Text.FontSize.Dp(12)
			// s.Text.LetterSpacing.Dp(0.4)
			s.Font.Weight = rich.Normal
		case TextTitleLarge:
			s.Text.LineSpacing = 28.0 / 22
			s.Text.FontSize.Dp(22)
			// s.Text.LetterSpacing.Zero()
			s.Font.Weight = rich.Normal
		case TextTitleMedium:
			s.Text.LineSpacing = 24.0 / 16
			s.Text.FontSize.Dp(16)
			// s.Text.LetterSpacing.Dp(0.15)
			s.Font.Weight = rich.Bold
		case TextTitleSmall:
			s.Text.LineSpacing = 20.0 / 14
			s.Text.FontSize.Dp(14)
			// s.Text.LetterSpacing.Dp(0.1)
			s.Font.Weight = rich.Medium
		case TextHeadlineLarge:
			s.Text.LineSpacing = 40.0 / 32
			s.Text.FontSize.Dp(32)
			// s.Text.LetterSpacing.Zero()
			s.Font.Weight = rich.Normal
		case TextHeadlineMedium:
			s.Text.LineSpacing = 36.0 / 28
			s.Text.FontSize.Dp(28)
			// s.Text.LetterSpacing.Zero()
			s.Font.Weight = rich.Normal
		case TextHeadlineSmall:
			s.Text.LineSpacing = 32.0 / 24
			s.Text.FontSize.Dp(24)
			// s.Text.LetterSpacing.Zero()
			s.Font.Weight = rich.Normal
		case TextDisplayLarge:
			s.Text.LineSpacing = 64.0 / 57
			s.Text.FontSize.Dp(57)
			// s.Text.LetterSpacing.Dp(-0.25)
			s.Font.Weight = rich.Normal
		case TextDisplayMedium:
			s.Text.LineSpacing = 52.0 / 45
			s.Text.FontSize.Dp(45)
			// s.Text.LetterSpacing.Zero()
			s.Font.Weight = rich.Normal
		case TextDisplaySmall:
			s.Text.LineSpacing = 44.0 / 36
			s.Text.FontSize.Dp(36)
			// s.Text.LetterSpacing.Zero()
			s.Font.Weight = rich.Normal
		}
		// the above linespacing factors are based on an em-based multiplier
		// instead, we are now using actual font height, so we need to reduce.
		s.Text.LineSpacing /= 1.25
	})
	tx.FinalStyler(func(s *styles.Style) {
		tx.normalCursor = s.Cursor
		// tx.paintText.UpdateColors(s.FontRender()) TODO(text):
		tx.updateRichText() // note: critical to update with final styles
	})

	tx.HandleTextClick(func(tl *rich.Hyperlink) {
		system.TheApp.OpenURL(tl.URL)
	})
	tx.OnFocusLost(func(e events.Event) {
		tx.selectReset()
	})
	tx.OnKeyChord(func(e events.Event) {
		if tx.selectRange.Len() == 0 {
			return
		}
		kf := keymap.Of(e.KeyChord())
		if kf == keymap.Copy {
			e.SetHandled()
			tx.copy()
		}
	})
	tx.On(events.MouseMove, func(e events.Event) {
		tl, _ := tx.findLink(e.Pos())
		if tl != nil {
			tx.Styles.Cursor = cursors.Pointer
		} else {
			tx.Styles.Cursor = tx.normalCursor
		}
	})
	tx.On(events.DoubleClick, func(e events.Event) {
		e.SetHandled()
		tx.selectWord(tx.pixelToRune(e.Pos()))
		tx.SetFocusQuiet()
	})
	tx.On(events.TripleClick, func(e events.Event) {
		e.SetHandled()
		tx.selectAll()
		tx.SetFocusQuiet()
	})
	tx.On(events.SlideStart, func(e events.Event) {
		e.SetHandled()
		tx.SetState(true, states.Sliding)
		tx.selectRange.Start = tx.pixelToRune(e.Pos())
		tx.selectRange.End = tx.selectRange.Start
		tx.paintText.SelectReset()
		tx.NeedsRender()
	})
	tx.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		tx.selectUpdate(tx.pixelToRune(e.Pos()))
		tx.NeedsRender()
	})

	tx.Updater(func() {
		tx.updateRichText()
		tx.configTextAlloc(tx.Geom.Size.Alloc.Content)
	})
}

// updateRichText gets the richtext from Text, using HTML parsing.
func (tx *Text) updateRichText() {
	if tx.Styles.Text.WhiteSpace.KeepWhiteSpace() {
		tx.richText = errors.Log1(htmltext.HTMLPreToRich([]byte(tx.Text), &tx.Styles.Font, nil))
	} else {
		tx.richText = errors.Log1(htmltext.HTMLToRich([]byte(tx.Text), &tx.Styles.Font, nil))
	}
}

// findLink finds the text link at the given scene-local position. If it
// finds it, it returns it and its bounds; otherwise, it returns nil.
func (tx *Text) findLink(pos image.Point) (*rich.Hyperlink, image.Rectangle) {
	if tx.paintText == nil || len(tx.Links) == 0 {
		return nil, image.Rectangle{}
	}
	tpos := tx.Geom.Pos.Content
	ri := tx.pixelToRune(pos)
	for li := range tx.Links {
		lr := &tx.Links[li]
		if !lr.Range.Contains(ri) {
			continue
		}
		gb := tx.paintText.RuneBounds(ri).Translate(tpos).ToRect()
		return lr, gb
	}
	return nil, image.Rectangle{}
}

// HandleTextClick handles click events such that the given function will be called
// on any links that are clicked on.
func (tx *Text) HandleTextClick(openLink func(tl *rich.Hyperlink)) {
	tx.OnClick(func(e events.Event) {
		tl, _ := tx.findLink(e.Pos())
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
	tl, bounds := tx.findLink(pos)
	if tl == nil {
		return tx.Tooltip, tx.DefaultTooltipPos()
	}
	return tl.URL, bounds.Min
}

func (tx *Text) copy() {
	md := mimedata.NewText(tx.Text[tx.selectRange.Start:tx.selectRange.End])
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

func (tx *Text) pixelToRune(pt image.Point) int {
	return tx.paintText.RuneAtPoint(math32.FromPoint(pt), tx.Geom.Pos.Content)
}

// selectUpdate updates selection based on rune index
func (tx *Text) selectUpdate(ri int) {
	if ri >= tx.selectRange.Start {
		tx.selectRange.End = ri
	} else {
		tx.selectRange.Start, tx.selectRange.End = ri, tx.selectRange.Start
	}
	tx.paintText.SelectReset()
	tx.paintText.SelectRegion(tx.selectRange)
}

// selectReset resets any current selection
func (tx *Text) selectReset() {
	tx.selectRange.Start = 0
	tx.selectRange.End = 0
	tx.paintText.SelectReset()
	tx.NeedsRender()
}

// selectAll selects entire set of text
func (tx *Text) selectAll() {
	tx.selectRange.Start = 0
	tx.selectUpdate(len(tx.Text))
	tx.NeedsRender()
}

// selectWord selects word at given rune location
func (tx *Text) selectWord(ri int) {
	// todo: write a general routine for this in rich.Text
}

// configTextSize does the text shaping layout for text,
// using given size to constrain layout.
func (tx *Text) configTextSize(sz math32.Vector2) {
	fs := &tx.Styles.Font
	txs := &tx.Styles.Text
	txs.Color = colors.ToUniform(tx.Styles.Color)
	tx.paintText = tx.Scene.TextShaper.WrapLines(tx.richText, fs, txs, &AppearanceSettings.Text, sz)
}

// configTextAlloc is used for determining how much space the text
// takes, using given size (typically Alloc).
// In this case, alignment factors are turned off,
// because they otherwise can absorb much more space, which should
// instead be controlled by the base Align X,Y factors.
func (tx *Text) configTextAlloc(sz math32.Vector2) math32.Vector2 {
	fs := &tx.Styles.Font
	txs := &tx.Styles.Text
	align, alignV := txs.Align, txs.AlignV
	txs.Align, txs.AlignV = text.Start, text.Start
	tx.paintText = tx.Scene.TextShaper.WrapLines(tx.richText, fs, txs, &AppearanceSettings.Text, sz)

	rsz := tx.paintText.Bounds.Size().Ceil()
	txs.Align, txs.AlignV = align, alignV
	tx.paintText = tx.Scene.TextShaper.WrapLines(tx.richText, fs, txs, &AppearanceSettings.Text, rsz)
	tx.Links = tx.paintText.Source.GetLinks()
	return rsz
}

func (tx *Text) SizeUp() {
	tx.WidgetBase.SizeUp() // sets Actual size based on styles
	sz := &tx.Geom.Size
	if tx.Styles.Text.WhiteSpace.HasWordWrap() {
		est := shaped.WrapSizeEstimate(sz.Actual.Content, len(tx.Text), 1.6, &tx.Styles.Font, &tx.Styles.Text)
		// if DebugSettings.LayoutTrace {
		// 	fmt.Println(tx, "Text SizeUp Estimate:", est)
		// }
		tx.configTextSize(est)
	} else {
		tx.configTextSize(sz.Actual.Content)
	}
	if tx.paintText == nil {
		return
	}
	rsz := tx.paintText.Bounds.Size().Ceil()
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.setTotalFromContent(&sz.Actual)
	if DebugSettings.LayoutTrace {
		fmt.Println(tx, "Text SizeUp:", rsz, "Actual:", sz.Actual.Content)
	}
}

func (tx *Text) SizeDown(iter int) bool {
	if !tx.Styles.Text.WhiteSpace.HasWordWrap() || iter > 1 {
		return false
	}
	sz := &tx.Geom.Size
	rsz := tx.configTextAlloc(sz.Alloc.Content) // use allocation
	prevContent := sz.Actual.Content
	// start over so we don't reflect hysteresis of prior guess
	sz.setInitContentMin(tx.Styles.Min.Dots().Ceil())
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.setTotalFromContent(&sz.Actual)
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
	tx.Scene.Painter.TextLines(tx.paintText, tx.Geom.Pos.Content)
}

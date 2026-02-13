// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"slices"
	"strconv"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
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
	Links []rich.Hyperlink `copier:"-" json:"-" xml:"-" set:"-"`

	// richText is the conversion of the HTML text source.
	// This is what is actually rendered.
	richText rich.Text

	// SelectRange is the selected range, in terms of rune indexes
	// into the richText.Join() runes.
	selectRange textpos.Range

	// highlights are regions that will be highlighted in the text
	// when rendered. Indexes are into richText.Join() runes.
	highlights []textpos.Range

	// paintText is the [shaped.Lines] for the text.
	paintText *shaped.Lines

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

	// TextSupporting is medium-sized supporting text typically used for
	// secondary dialog information below the title. It has a default font
	// size of 14dp and color of [colors.Scheme.OnSurfaceVariant].
	TextSupporting
)

func (tx *Text) WidgetValue() any { return &tx.Text }

// PaintText returns the shaped representation of the text,
// which is needed for some specific special-case rendering
// situations.
func (tx *Text) PaintText() *shaped.Lines {
	return tx.paintText
}

func (tx *Text) Init() {
	tx.WidgetBase.Init()
	tx.AddContextMenu(tx.contextMenu)
	tx.SetType(TextBodyLarge)
	tx.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Selectable, abilities.DoubleClickable, abilities.TripleClickable, abilities.LongPressable, abilities.Slideable)
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
			s.Text.LineHeight = 20.0 / 14
			s.Font.Size.Dp(14)
			s.Font.Weight = rich.Medium
		case TextLabelMedium:
			s.Text.LineHeight = 16.0 / 12
			s.Font.Size.Dp(12)
			s.Font.Weight = rich.Medium
		case TextLabelSmall:
			s.Text.LineHeight = 16.0 / 11
			s.Font.Size.Dp(11)
			s.Font.Weight = rich.Medium
		case TextBodyLarge:
			s.Text.LineHeight = 24.0 / 16
			// s.Font.Size.Dp(16) // already the default so don't specify (to allow for inheritance) (important)
			s.Font.Weight = rich.Normal
		case TextSupporting:
			s.Color = colors.Scheme.OnSurfaceVariant
			fallthrough
		case TextBodyMedium:
			s.Text.LineHeight = 20.0 / 14
			s.Font.Size.Dp(14)
			s.Font.Weight = rich.Normal
		case TextBodySmall:
			s.Text.LineHeight = 16.0 / 12
			s.Font.Size.Dp(12)
			s.Font.Weight = rich.Normal
		case TextTitleLarge:
			s.Text.LineHeight = 28.0 / 22
			s.Font.Size.Dp(22)
			s.Font.Weight = rich.Normal
		case TextTitleMedium:
			s.Text.LineHeight = 24.0 / 16
			s.Font.Size.Dp(16)
			s.Font.Weight = rich.Bold
		case TextTitleSmall:
			s.Text.LineHeight = 20.0 / 14
			s.Font.Size.Dp(14)
			s.Font.Weight = rich.Medium
		case TextHeadlineLarge:
			s.Text.LineHeight = 40.0 / 32
			s.Font.Size.Dp(32)
			s.Font.Weight = rich.Normal
		case TextHeadlineMedium:
			s.Text.LineHeight = 36.0 / 28
			s.Font.Size.Dp(28)
			s.Font.Weight = rich.Normal
		case TextHeadlineSmall:
			s.Text.LineHeight = 32.0 / 24
			s.Font.Size.Dp(24)
			s.Font.Weight = rich.Normal
		case TextDisplayLarge:
			s.Text.LineHeight = 70.0 / 57
			s.Font.Size.Dp(57)
			s.Font.Weight = rich.Normal
		case TextDisplayMedium:
			s.Text.LineHeight = 52.0 / 45
			s.Font.Size.Dp(45)
			s.Font.Weight = rich.Normal
		case TextDisplaySmall:
			s.Text.LineHeight = 44.0 / 36
			s.Font.Size.Dp(36)
			s.Font.Weight = rich.Normal
		}
	})
	tx.FinalStyler(func(s *styles.Style) {
		tx.normalCursor = s.Cursor
		tx.updateRichText() // note: critical to update with final styles
		if tx.paintText != nil && tx.Text != "" {
			_, tsty := s.NewRichText()
			tx.paintText.UpdateStyle(tx.richText, tsty)
		}
	})

	tx.HandleTextClick(func(tl *rich.Hyperlink, e events.Event) {
		system.TheApp.OpenURL(tl.URL)
	})
	tx.OnFocusLost(func(e events.Event) {
		if !(tx.Scene.selectedText != nil && !e.HasAnyModifier(key.Shift, key.Meta)) {
			tx.selectReset()
		}
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
	tx.OnFinal(events.Click, func(e events.Event) {
		if !e.HasAnyModifier(key.Shift, key.Meta) {
			tx.selectReset()
		}
		if !TheApp.SystemPlatform().IsMobile() {
			return
		}
		if tx.StateIs(states.Attended) { // toggle attention with clicking
			// this allows drag scrolling to proceed.
			e.SetHandled()
			em := tx.Events()
			if em != nil {
				em.setAttend(nil)
			}
		}
	})
	tx.On(events.DoubleClick, func(e events.Event) {
		e.SetHandled()
		if !slices.Contains(tx.Scene.selectedText, tx) {
			tx.Scene.selectedText = append(tx.Scene.selectedText, tx)
		}
		tx.selectWord(tx.pixelToRune(e.Pos()))
		tx.SetFocusQuiet()
	})
	tx.On(events.TripleClick, func(e events.Event) {
		e.SetHandled()
		if !slices.Contains(tx.Scene.selectedText, tx) {
			tx.Scene.selectedText = append(tx.Scene.selectedText, tx)
		}
		tx.selectAll()
		tx.SetFocusQuiet()
		if TheApp.SystemPlatform().IsMobile() {
			tx.Send(events.ContextMenu, e)
		}
	})
	tx.On(events.SlideStart, func(e events.Event) {
		e.SetHandled()
		tx.SetState(true, states.Sliding)
		if len(tx.Scene.selectedText) == 0 {
			tx.SetFocusQuiet()
		}
		tx.selectRange.Start = tx.pixelToRune(e.Pos())
		tx.selectRange.End = tx.selectRange.Start
		if !slices.Contains(tx.Scene.selectedText, tx) {
			tx.Scene.selectedText = append(tx.Scene.selectedText, tx)
		}
		tx.NeedsRender()
	})
	tx.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		tx.selectUpdate(tx.pixelToRune(e.Pos()))
		tx.NeedsRender()
	})
	tx.On(events.SlideStop, func(e events.Event) {
		if TheApp.SystemPlatform().IsMobile() {
			tx.Send(events.ContextMenu, e)
		}
	})

	tx.FinalUpdater(func() {
		tx.updateRichText()
		tx.configTextAlloc(tx.Geom.Size.Alloc.Content)
	})
}

// updateRichText gets the richtext from Text, using HTML parsing.
func (tx *Text) updateRichText() {
	sty, tsty := tx.Styles.NewRichText()
	if tsty.WhiteSpace.KeepWhiteSpace() {
		tx.richText, _ = htmltext.HTMLPreToRich([]byte(tx.Text), sty, nil)
	} else {
		tx.richText, _ = htmltext.HTMLToRich([]byte(tx.Text), sty, nil)
	}
	tx.Links = tx.richText.GetLinks()
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
func (tx *Text) HandleTextClick(openLink func(tl *rich.Hyperlink, e events.Event)) {
	tx.OnClick(func(e events.Event) {
		tl, _ := tx.findLink(e.Pos())
		if tl == nil {
			return
		}
		openLink(tl, e)
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

func (tx *Text) contextMenu(m *Scene) {
	NewFuncButton(m).SetFunc(tx.copy).SetIcon(icons.Copy).SetKey(keymap.Copy).SetEnabled(tx.hasSelection())
}

func (tx *Text) copy() { //types:add
	if !tx.hasSelection() {
		return
	}
	var sel string
	if len(tx.Scene.selectedText) > 0 {
		for _, st := range tx.Scene.selectedText {
			if len(sel) > 0 {
				sel += "\n"
			}
			sel += string(st.richText.Join()[st.selectRange.Start:st.selectRange.End])
		}
	} else {
		// note: selectRange is in runes, not string indexes.
		sel = string(tx.richText.Join()[tx.selectRange.Start:tx.selectRange.End])
	}
	md := mimedata.NewText(sel)
	em := tx.Events()
	if em != nil {
		em.Clipboard().Write(md)
	}
	tx.selectReset()
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
}

// hasSelection returns true if there is an active selection.
func (tx *Text) hasSelection() bool {
	return tx.selectRange.Len() > 0
}

// selectReset resets any current selection, including for the multi-text
// Scene selectedText.
func (tx *Text) selectReset() {
	reset := func(t *Text) {
		t.selectRange.Start = 0
		t.selectRange.End = 0
		t.NeedsRender()
	}
	for _, st := range tx.Scene.selectedText {
		reset(st)
	}
	reset(tx)
	tx.Scene.selectedText = nil
}

// selectAll selects entire set of text
func (tx *Text) selectAll() {
	tx.selectRange.Start = 0
	txt := tx.richText.Join()
	tx.selectUpdate(len(txt))
	tx.NeedsRender()
}

// selectWord selects word at given rune location
func (tx *Text) selectWord(ri int) {
	txt := tx.richText.Join()
	wr := textpos.WordAt(txt, ri)
	if wr.Start >= 0 {
		tx.selectRange = wr
		tx.paintText.SelectRegion(tx.selectRange)
	}
	tx.NeedsRender()
}

// configTextSize does the text shaping layout for text,
// using given size to constrain layout.
func (tx *Text) configTextSize(sz math32.Vector2) {
	if tx.Styles.Font.Size.Dots == 0 { // not init
		return
	}
	sty, tsty := tx.Styles.NewRichText()
	tsty.Align, tsty.AlignV = text.Start, text.Start
	tx.paintText = tx.Scene.TextShaper().WrapLines(tx.richText, sty, tsty, sz)
	tx.setAnchorFromProperties()
}

// setAnchorFromProperties sets the Anchor string on paintText based on
// id and tag properties, which is needed for PDF print formatting.
// This is the only point at which we know we have the final
// shaped.Lines for a text item.
func (tx *Text) setAnchorFromProperties() {
	anc := ""
	if id, ok := tx.Properties["id"]; ok {
		anc = id.(string)
	}
	if t, ok := tx.Properties["tag"]; ok {
		tag := t.(string)
		if len(tag) > 1 && tag[0] == 'h' { // header
			level := errors.Log1(strconv.Atoi(tag[1:]))
			anc += fmt.Sprintf(";Header %d:%s", level, tx.Text)
		}
	}
	tx.paintText.Anchor = anc
}

// configTextAlloc is used for determining how much space the text
// takes, using given size (typically Alloc).
// In this case, alignment factors are turned off,
// because they otherwise can absorb much more space, which should
// instead be controlled by the base Align X,Y factors.
func (tx *Text) configTextAlloc(sz math32.Vector2) math32.Vector2 {
	if tx.Scene == nil || tx.Scene.TextShaper() == nil {
		return sz
	}
	if tx.Styles.Font.Size.Dots == 0 {
		return sz // not init
	}
	tsh := tx.Scene.TextShaper()
	sty, tsty := tx.Styles.NewRichText()
	rsz := sz
	if tsty.Align != text.Start && tsty.AlignV != text.Start {
		etxs := *tsty
		etxs.Align, etxs.AlignV = text.Start, text.Start
		tx.paintText = tsh.WrapLines(tx.richText, sty, &etxs, rsz)
		rsz = tx.paintText.Bounds.Size().Ceil()
	}
	tx.paintText = tsh.WrapLines(tx.richText, sty, tsty, rsz)
	tx.setAnchorFromProperties()
	return tx.paintText.Bounds.Size().Ceil()
}

func (tx *Text) SizeUp() {
	tx.WidgetBase.SizeUp() // sets Actual size based on styles
	sz := &tx.Geom.Size
	if tx.Styles.Text.WhiteSpace.HasWordWrap() {
		sty, tsty := tx.Styles.NewRichText()
		est := shaped.WrapSizeEstimate(sz.Actual.Content, len(tx.Text), .5, sty, tsty)
		tx.configTextSize(est)
	} else {
		tx.configTextSize(sz.Actual.Content)
	}
	if tx.paintText == nil {
		return
	}
	rsz := tx.paintText.Bounds.Size().Ceil()
	sz.FitSizeMax(&sz.Actual.Content, rsz)
	sz.SetTotalFromContent(&sz.Actual)
	if DebugSettings.LayoutTrace {
		fmt.Println(tx, "Text SizeUp:", rsz, "Actual:", sz.Actual.Content)
	}
}

func (tx *Text) SizeDown(iter int) bool {
	if !tx.Styles.Text.WhiteSpace.HasWordWrap() || iter > 1 {
		return false
	}
	sz := &tx.Geom.Size
	asz := sz.Alloc.Content
	rsz := tx.configTextAlloc(asz) // use allocation
	prevContent := sz.Actual.Content
	// start over so we don't reflect hysteresis of prior guess
	sz.setInitContentMin(tx.Styles.Min.Dots().Ceil())
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
	tx.paintText.SelectReset()
	tx.paintText.SelectRegion(tx.selectRange)
	tx.paintText.HighlightReset()
	tx.paintText.HighlightRegion(tx.highlights...)
	tx.Scene.Painter.DrawText(tx.paintText, tx.Geom.Pos.Content)
}

//////// Search interface

// TextRunes returns any text content associated with the widget, to be used
// for Search for example. If this is nil, then it is excluded from search.
func (tx *Text) TextRunes() []rune {
	return tx.richText.Join()
}

// Search returns text search results for this widget, searching for
// the find string with given case sensitivity. It is up to each widget
// to define the meaning of the Region line, char values for the matches.
// The bool return value indicates whether this widget handled the search,
// thereby excluding further searching within the elements under it.
func (tx *Text) Search(find string, useCase bool) ([]textpos.Match, bool) {
	return SearchRunes(tx.richText.Join(), find, useCase), true
}

// HighlightMatches does highlighting of the given matches within this widget,
// where the matches are as returned from the Search method.
// Passing a nil causes matches to be reset.
// Any existing highlighting should always be reset first regardless.
// The bool return value indicates whether this widget handled the search,
// thereby excluding further searching within the elements under it.
func (tx *Text) HighlightMatches(matches []textpos.Match) bool {
	tx.highlights = nil
	tx.NeedsRender()
	if matches == nil {
		return true
	}
	for _, m := range matches {
		tx.highlights = append(tx.highlights, textpos.Range{Start: m.Region.Start.Char, End: m.Region.End.Char})
	}
	return true
}

// SelectMatch selects match at given index from among those returned
// from the Search method. scroll = scroll widget into view.
// reset = clear selection instead of selecting (does not scroll).
func (tx *Text) SelectMatch(matches []textpos.Match, index int, scroll, reset bool) {
	if reset {
		tx.selectReset()
		return
	}
	match := matches[index]
	tx.selectRange = textpos.Range{Start: match.Region.Start.Char, End: match.Region.End.Char}
	tx.NeedsRender()
	if scroll {
		tx.ScrollThisToTop()
	}
}

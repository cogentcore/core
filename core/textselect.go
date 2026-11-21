// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"slices"
	"strings"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/text/textpos"
)

// The selection of text by sliding (dragging) the mouse works as in a web
// browser: it is from the anchor, where the mouse was pressed in a [Text],
// to the focus, the closest place in a text to the mouse, and it includes
// all of the texts in between them in the order of the document (the
// widget tree), which is the order of the source of an htmlcore page, such
// as that of the columns of a multi-column layout. The selection is made
// again for each move of the mouse, from the anchor, so that it follows the
// mouse back past the anchor, up or down. The slide events all go to the
// anchor text (see [Text.selectTo]).

// selectionRoot returns the widget whose texts the selection of the given
// text can extend to: the closest frame around it that scrolls (such as
// the page of a [content.Content]), or else its scene, so that the
// selection does not extend into toolbars and side panels.
func (tx *Text) selectionRoot() Widget {
	for p := tx.Parent; p != nil; p = p.AsTree().Parent {
		w, ok := p.(Widget)
		if !ok {
			break
		}
		st := &w.AsWidget().Styles
		if st.Overflow.X >= styles.OverflowAuto || st.Overflow.Y >= styles.OverflowAuto {
			return w
		}
	}
	return tx.Scene
}

// selectableTexts returns the texts in the given widget that can be
// selected, in the order of the document. The texts of buttons and other
// widgets that are activated by clicking (such as the nodes of trees) are
// not part of the document text.
func selectableTexts(root Widget) []*Text {
	var texts []*Text
	root.AsWidget().WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if !cwb.IsDisplayable() {
			return false // hidden
		}
		if t, ok := cw.(*Text); ok {
			if t.AbilityIs(abilities.Selectable) {
				texts = append(texts, t)
			}
			return false
		}
		if cw != root && cwb.AbilityIs(abilities.Activatable) {
			return false
		}
		return true
	})
	return texts
}

// closestText returns the index of the text of the given texts that is
// closest to the given position (in the scene), and the index of the rune
// in it that is closest to it, or -1 if none of the texts is visible.
// The distance is to the visible box of each text, so that the position
// is in the text if it is over it, and otherwise it is the closest one,
// which is the one in the same column of a multi-column layout when it is
// between the paragraphs of a column.
func closestText(texts []*Text, pos image.Point) (int, int) {
	best, bestDist := -1, float32(0)
	pt := math32.FromPoint(pos)
	for i, t := range texts {
		bb := t.Geom.ContentBBox
		if bb.Empty() {
			continue
		}
		box := math32.B2FromRect(bb)
		d := box.ClampPoint(pt).DistanceToSquared(pt)
		if best < 0 || d < bestDist {
			best, bestDist = i, d
			if d == 0 {
				break
			}
		}
	}
	if best < 0 {
		return -1, 0
	}
	return best, texts[best].pixelToRune(pos)
}

// startSelect starts the selection of text by sliding the mouse at the
// given rune index of this text, the anchor, resetting any other selection.
func (tx *Text) startSelect(ri int) {
	tx.selectReset()
	tx.selectAnchor = ri
	tx.selectRange = textpos.Range{Start: ri, End: ri}
	tx.selectTexts = selectableTexts(tx.selectionRoot())
	tx.Scene.selectedText = []*Text{tx}
	tx.NeedsRender()
}

// selectTo selects the text from the anchor of the selection, in this
// text, to the closest place in a text to the given position (in the
// scene), as the mouse slides there (see [Text.startSelect]).
func (tx *Text) selectTo(pos image.Point) {
	texts := tx.selectTexts
	ai := slices.Index(texts, tx)
	fi, fr := -1, 0
	if ai >= 0 {
		fi, fr = closestText(texts, pos)
	}
	if fi < 0 || fi == ai {
		tx.setSelection(textpos.Range{Start: min(tx.selectAnchor, tx.pixelToRune(pos)), End: max(tx.selectAnchor, tx.pixelToRune(pos))})
		if ai >= 0 {
			tx.setSelectedTexts(texts[ai : ai+1])
		}
		return
	}
	lo, hi := min(ai, fi), max(ai, fi)
	for i := lo; i <= hi; i++ {
		t := texts[i]
		r := textpos.Range{Start: 0, End: t.textLen()}
		switch {
		case i == ai && fi > ai: // the selection goes down from the anchor
			r.Start = tx.selectAnchor
		case i == ai: // up from it
			r.End = tx.selectAnchor
		case i == fi && fi > ai:
			r.End = fr
		case i == fi:
			r.Start = fr
		}
		t.setSelection(r)
	}
	tx.setSelectedTexts(texts[lo : hi+1])
}

// setSelection sets the selection of the text to the given range,
// rendering it again if it changed.
func (tx *Text) setSelection(r textpos.Range) {
	if tx.selectRange == r {
		return
	}
	tx.selectRange = r
	tx.NeedsRender()
}

// setSelectedTexts sets the texts of the selection of the scene of the
// text, in the order of the document, resetting the selection of the
// texts that are no longer in it.
func (tx *Text) setSelectedTexts(texts []*Text) {
	for _, st := range tx.Scene.selectedText {
		if !slices.Contains(texts, st) {
			st.setSelection(textpos.Range{})
		}
	}
	tx.Scene.selectedText = slices.Clone(texts)
}

// textLen returns the number of runes of the text.
func (tx *Text) textLen() int {
	if tx.paintText == nil {
		return 0
	}
	return len(tx.richText.Join())
}

// selectedText returns the text that is selected in the texts of the
// selection of the scene of this text (one line for each text with a
// selection, in the order of the document), or in this text.
func (tx *Text) selectedText() string {
	if len(tx.Scene.selectedText) == 0 {
		// note: selectRange is in runes, not string indexes.
		return string(tx.richText.Join()[tx.selectRange.Start:tx.selectRange.End])
	}
	var sel []string
	for _, st := range tx.Scene.selectedText {
		if st.selectRange.Len() > 0 {
			sel = append(sel, string(st.richText.Join()[st.selectRange.Start:st.selectRange.End]))
		}
	}
	return strings.Join(sel, "\n")
}

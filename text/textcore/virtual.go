// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
)

// SetVirtualText sets the virtual text in the given layer that is shown
// only in this editor, such as the composition of an input method or
// a suggested completion (see [lines.Lines.SetVirtualText]).
// Virtual text shown in all editors of the same [lines.Lines], such as
// language server inlay hints, is set on the Lines directly, with a
// view of -1.
func (ed *Base) SetVirtualText(layer string, vts ...lines.VirtualText) {
	if ed.Lines == nil {
		return
	}
	ed.Lines.SetVirtualText(layer, ed.viewId, vts...)
}

// ClearVirtualText removes the virtual text in the given layer
// that is shown only in this editor.
func (ed *Base) ClearVirtualText(layer string) {
	ed.SetVirtualText(layer)
}

// VirtualText returns the virtual text in the given layer that is
// shown only in this editor, with positions updated for any edits.
func (ed *Base) VirtualText(layer string) []lines.VirtualText {
	if ed.Lines == nil {
		return nil
	}
	return ed.Lines.VirtualText(layer, ed.viewId)
}

// VirtualCharStartPos returns the starting (top left) render coords of the
// char at the given offset within the virtual text in the given layer at
// the given source position (see [lines.Lines.VirtualTextView]), in the
// same coordinates as [Base.CharStartPos]. For example, an input method
// places its candidate window at its composition cursor. It returns false
// if there is no such virtual text shown in this editor.
func (ed *Base) VirtualCharStartPos(layer string, pos textpos.Pos, offset int) (math32.Vector2, bool) {
	if ed.Lines == nil {
		return math32.Vector2{}, false
	}
	vp, ok := ed.Lines.VirtualTextView(ed.viewId, layer, pos, offset)
	if !ok {
		return math32.Vector2{}, false
	}
	return ed.viewCharStartPos(vp), true
}

// virtualStyle returns a copy of the font style of the text.
func (ed *Base) virtualStyle() *rich.Style {
	sty := *rich.NewStyle()
	if ed.Lines != nil {
		sty = *ed.Lines.FontStyle()
	}
	return &sty
}

// CompositionStyle returns the style for the composition of an input
// method, which is underlined.
func (ed *Base) CompositionStyle() *rich.Style {
	return ed.virtualStyle().SetDecoration(rich.Underline)
}

// HintStyle returns the style for hints, such as language server
// inlay hints, which are dimmed, on a subtle background.
func (ed *Base) HintStyle() *rich.Style {
	sty := ed.virtualStyle()
	sty.SetFillColor(colors.ToUniform(colors.Scheme.OnSurfaceVariant))
	sty.SetBackground(colors.ToUniform(colors.Scheme.SurfaceContainerHigh))
	return sty
}

// SuggestionStyle returns the style for suggested text, such as
// suggested completions, which is dimmed and italic.
func (ed *Base) SuggestionStyle() *rich.Style {
	sty := ed.virtualStyle()
	sty.Slant = rich.Italic
	sty.SetFillColor(colors.ToUniform(colors.Scheme.OnSurfaceVariant))
	return sty
}

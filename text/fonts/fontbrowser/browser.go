// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate -add-types -setters

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"unicode"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/keylist"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

// Browser is a font browser.
type Browser struct {
	core.Frame

	Filename core.Filename
	Font     *font.Face
	RuneMap  *keylist.List[rune, font.GID]
}

// OpenFile opens a font file.
func (fb *Browser) OpenFile(fname core.Filename) error { //types:add
	return fb.OpenFileIndex(fname, 0)
}

// OpenFileIndex opens a font file.
func (fb *Browser) OpenFileIndex(fname core.Filename, index int) error { //types:add
	b, err := os.ReadFile(string(fname))
	if errors.Log(err) != nil {
		return err
	}
	fb.Filename = fname
	return fb.OpenFontData(b, index)
}

// SelectFont selects a font from among a loaded list.
func (fb *Browser) SelectFont() { //types:add
	d := core.NewBody("Select Font")
	d.SetTitle("Select a font family")
	si := 0
	fl := fb.Scene.TextShaper().FontList()
	fi := fonts.FontFamilies(fl)
	tb := core.NewTable(d)
	tb.SetSlice(&fi).SetSelectedField("Family").
		SetSelectedValue(fb.Font.Describe().Family).BindSelect(&si)
	tb.SetTableStyler(func(w core.Widget, s *styles.Style, row, col int) {
		if col != 1 {
			return
		}
		s.Text.CustomFont = rich.FontName(fi[row].Family)
		s.Font.Family = rich.Custom
		s.Text.FontSize.Dp(24)
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddOK(bar).OnClick(func(e events.Event) {
			fam := fi[si].Family
			idx := 0
			for i := range fl {
				if fl[i].Family == fam && (fl[i].Weight == rich.Medium || fl[i].Weight == rich.Normal) {
					idx = i
					break
				}
			}
			loc := fl[idx].Font.Location
			finfo := fmt.Sprintf("loading font: %s from: %s idx: %d, sel: %d", fam, loc.File, loc.Index, si)
			fmt.Println(finfo)
			core.MessageSnackbar(fb, finfo)
			fb.OpenFileIndex(core.Filename(loc.File), int(loc.Index))
		})
	})
	d.RunWindowDialog(fb)
}

// OpenFontData opens given font data.
func (fb *Browser) OpenFontData(b []byte, index int) error {
	faces, err := font.ParseTTC(bytes.NewReader(b))
	if errors.Log(err) != nil {
		return err
	}
	fb.Font = faces[index]
	fb.UpdateRuneMap()
	fb.Update()
	return nil
}

func (fb *Browser) UpdateRuneMap() {
	fb.DeleteChildren()
	fb.RuneMap = keylist.New[rune, font.GID]()
	if fb.Font == nil {
		return
	}
	for _, pr := range unicode.PrintRanges {
		for _, rv := range pr.R16 {
			for r := rv.Lo; r <= rv.Hi; r += rv.Stride {
				gid, has := fb.Font.NominalGlyph(rune(r))
				if !has {
					continue
				}
				fb.RuneMap.Add(rune(r), gid)
			}
		}
	}
}

// SelectRune selects a rune in current font (first char) of string.
func (fb *Browser) SelectRune(r string) { //types:add
	rs := []rune(r)
	if len(rs) == 0 {
		core.MessageSnackbar(fb, "no runes!")
		return
	}
	ix := fb.RuneMap.IndexByKey(rs[0])
	if ix < 0 {
		core.MessageSnackbar(fb, "rune not found!")
		return
	}
	gi := fb.Child(ix).(core.Widget).AsWidget()
	gi.Styles.State.SetFlag(true, states.Selected, states.Active)
	gi.SetFocus()
	core.MessageSnackbar(fb, fmt.Sprintf("rune %s at index: %d GID: %d", r, ix, fb.RuneMap.Values[ix]))
}

// SelectRuneInt selects a rune in current font by number
func (fb *Browser) SelectRuneInt(r int) { //types:add
	ix := fb.RuneMap.IndexByKey(rune(r))
	if ix < 0 {
		core.MessageSnackbar(fb, "rune not found!")
		return
	}
	gi := fb.Child(ix).(core.Widget).AsWidget()
	gi.Styles.State.SetFlag(true, states.Selected, states.Active)
	gi.SetFocus()
	core.MessageSnackbar(fb, fmt.Sprintf("rune %s at index: %d GID: %d", string(rune(r)), ix, fb.RuneMap.Values[ix]))
}

// SelectGlyphID selects glyphID in current font.
func (fb *Browser) SelectGlyphID(gid opentype.GID) { //types:add
	ix := -1
	for i, g := range fb.RuneMap.Values {
		if gid == g {
			ix = i
			break
		}
	}
	if ix < 0 {
		core.MessageSnackbar(fb, "glyph id not found!")
		return
	}
	r := string(rune(fb.RuneMap.Keys[ix]))
	gi := fb.Child(ix).(core.Widget).AsWidget()
	gi.Styles.State.SetFlag(true, states.Selected, states.Active)
	gi.SetFocus()
	core.MessageSnackbar(fb, fmt.Sprintf("rune %s at index: %d GID: %d", r, ix, fb.RuneMap.Values[ix]))
}

func (fb *Browser) Init() {
	fb.Frame.Init()
	fb.Styler(func(s *styles.Style) {
		// s.Display = styles.Flex
		// s.Wrap = true
		// s.Direction = styles.Row
		s.Display = styles.Grid
		s.Columns = 32
	})
	fb.Maker(func(p *tree.Plan) {
		if fb.Font == nil {
			return
		}
		for i, gid := range fb.RuneMap.Values {
			r := fb.RuneMap.Keys[i]
			nm := string(r) + "_" + strconv.Itoa(int(r))
			tree.AddAt(p, nm, func(w *Glyph) {
				w.SetBrowser(fb).SetRune(r).SetGID(gid)
			})
		}
	})
}

func (fb *Browser) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fb.OpenFile).SetIcon(icons.Open).SetKey(keymap.Open)
		w.Args[0].SetValue(fb.Filename).SetTag(`extension:".ttf"`)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fb.SelectFont).SetIcon(icons.Open)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fb.SelectEmbedded).SetIcon(icons.Open)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fb.SelectRune).SetIcon(icons.Select)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fb.SelectRuneInt).SetIcon(icons.Select)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(fb.SelectGlyphID).SetIcon(icons.Select)
	})
}

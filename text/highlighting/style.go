// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package highlighting provides syntax highlighting styles; it is based on
// github.com/alecthomas/chroma, which in turn was based on the python
// pygments package.  Note that this package depends on core and parse
// and cannot be imported there; is imported in texteditor.
package highlighting

//go:generate core generate -add-types

import (
	"encoding/json"
	"image/color"
	"log/slog"
	"os"
	"strings"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/token"
)

type HighlightingName string

// Trilean value for StyleEntry value inheritance.
type Trilean int32 //enums:enum

const (
	Pass Trilean = iota
	Yes
	No
)

func (t Trilean) Prefix(s string) string {
	if t == Yes {
		return s
	} else if t == No {
		return "no" + s
	}
	return ""
}

// TODO(go1.24): use omitzero instead of omitempty in [StyleEntry]
// once we update to go1.24

// StyleEntry is one value in the map of highlight style values
type StyleEntry struct {

	// Color is the text color.
	Color color.RGBA `json:",omitempty"`

	// Background color.
	// In general it is not good to use this because it obscures highlighting.
	Background color.RGBA `json:",omitempty"`

	// Border color? not sure what this is -- not really used.
	Border color.RGBA `display:"-" json:",omitempty"`

	// Bold font.
	Bold Trilean `json:",omitempty"`

	// Italic font.
	Italic Trilean `json:",omitempty"`

	// Underline.
	Underline Trilean `json:",omitempty"`

	// DottedUnderline
	DottedUnderline Trilean `json:",omitempty"`

	// NoInherit indicates to not inherit these settings from sub-category or category levels.
	// Otherwise everything with a Pass is inherited.
	NoInherit bool `json:",omitempty"`

	// themeColor is the theme-adjusted text color.
	themeColor color.RGBA

	// themeBackground is the theme-adjusted background color.
	themeBackground color.RGBA
}

// // FromChroma copies styles from chroma
//
//	func (he *StyleEntry) FromChroma(ce chroma.StyleEntry) {
//		if ce.Colour.IsSet() {
//			he.Color.SetString(ce.Colour.String(), nil)
//		} else {
//			he.Color.SetToNil()
//		}
//		if ce.Background.IsSet() {
//			he.Background.SetString(ce.Background.String(), nil)
//		} else {
//			he.Background.SetToNil()
//		}
//		if ce.Border.IsSet() {
//			he.Border.SetString(ce.Border.String(), nil)
//		} else {
//			he.Border.SetToNil()
//		}
//		he.Bold = Trilean(ce.Bold)
//		he.Italic = Trilean(ce.Italic)
//		he.Underline = Trilean(ce.Underline)
//		he.NoInherit = ce.NoInherit
//	}
//
// // StyleEntryFromChroma returns a new style entry from corresponding chroma version
//
//	func StyleEntryFromChroma(ce chroma.StyleEntry) StyleEntry {
//		he := StyleEntry{}
//		he.FromChroma(ce)
//		return he
//	}

// UpdateFromTheme normalizes the colors of the style entry such that they have consistent
// chromas and tones that guarantee sufficient text contrast in accordance with the color theme.
func (se *StyleEntry) UpdateFromTheme() {
	hc := hct.FromColor(se.Color)
	ctone := float32(40)
	if matcolor.SchemeIsDark {
		ctone = 80
	}
	se.themeColor = hc.WithChroma(max(hc.Chroma, 48)).WithTone(ctone).AsRGBA()

	if !colors.IsNil(se.Background) {
		hb := hct.FromColor(se.Background)
		btone := max(hb.Tone, 94)
		if matcolor.SchemeIsDark {
			btone = min(hb.Tone, 17)
		}
		se.themeBackground = hb.WithChroma(max(hb.Chroma, 6)).WithTone(btone).AsRGBA()
	}
}

func (se StyleEntry) String() string {
	out := []string{}
	if se.Bold != Pass {
		out = append(out, se.Bold.Prefix("bold"))
	}
	if se.Italic != Pass {
		out = append(out, se.Italic.Prefix("italic"))
	}
	if se.Underline != Pass {
		out = append(out, se.Underline.Prefix("underline"))
	}
	if se.DottedUnderline != Pass {
		out = append(out, se.Underline.Prefix("dotted-underline"))
	}
	if se.NoInherit {
		out = append(out, "noinherit")
	}
	if !colors.IsNil(se.themeColor) {
		out = append(out, colors.AsString(se.themeColor))
	}
	if !colors.IsNil(se.themeBackground) {
		out = append(out, "bg:"+colors.AsString(se.themeBackground))
	}
	if !colors.IsNil(se.Border) {
		out = append(out, "border:"+colors.AsString(se.Border))
	}
	return strings.Join(out, " ")
}

// ToCSS converts StyleEntry to CSS attributes.
func (se StyleEntry) ToCSS() string {
	styles := []string{}
	if !colors.IsNil(se.themeColor) {
		styles = append(styles, "color: "+colors.AsString(se.themeColor))
	}
	if !colors.IsNil(se.themeBackground) {
		styles = append(styles, "background-color: "+colors.AsString(se.themeBackground))
	}
	if se.Bold == Yes {
		styles = append(styles, "font-weight: bold")
	}
	if se.Italic == Yes {
		styles = append(styles, "font-style: italic")
	}
	if se.Underline == Yes {
		styles = append(styles, "text-decoration: underline")
	} else if se.DottedUnderline == Yes {
		styles = append(styles, "text-decoration: dotted-underline")
	}

	return strings.Join(styles, "; ")
}

// ToProperties converts the StyleEntry to key-value properties.
func (se StyleEntry) ToProperties() map[string]any {
	pr := map[string]any{}
	if !colors.IsNil(se.themeColor) {
		pr["color"] = se.themeColor
	}
	if !colors.IsNil(se.themeBackground) {
		pr["background-color"] = se.themeBackground
	}
	if se.Bold == Yes {
		pr["font-weight"] = rich.Bold
	}
	if se.Italic == Yes {
		pr["font-style"] = rich.Italic
	}
	if se.Underline == Yes {
		pr["text-decoration"] = 1 << uint32(rich.Underline)
	} else if se.Underline == Yes {
		pr["text-decoration"] = 1 << uint32(rich.DottedUnderline)
	}
	return pr
}

// ToRichStyle sets the StyleEntry to given [rich.Style].
func (se StyleEntry) ToRichStyle(sty *rich.Style) {
	if !colors.IsNil(se.themeColor) {
		sty.SetFillColor(se.themeColor)
	}
	if !colors.IsNil(se.themeBackground) {
		sty.SetBackground(se.themeBackground)
	}
	if se.Bold == Yes {
		sty.Weight = rich.Bold
	}
	if se.Italic == Yes {
		sty.Slant = rich.Italic
	}
	if se.Underline == Yes {
		sty.Decoration.SetFlag(true, rich.Underline)
	} else if se.DottedUnderline == Yes {
		sty.Decoration.SetFlag(true, rich.DottedUnderline)
	}
}

// Sub subtracts two style entries, returning an entry with only the differences set
func (se StyleEntry) Sub(e StyleEntry) StyleEntry {
	out := StyleEntry{}
	if e.Color != se.Color {
		out.Color = se.Color
		out.themeColor = se.themeColor
	}
	if e.Background != se.Background {
		out.Background = se.Background
		out.themeBackground = se.themeBackground
	}
	if e.Border != se.Border {
		out.Border = se.Border
	}
	if e.Bold != se.Bold {
		out.Bold = se.Bold
	}
	if e.Italic != se.Italic {
		out.Italic = se.Italic
	}
	if e.Underline != se.Underline {
		out.Underline = se.Underline
	}
	if e.DottedUnderline != se.DottedUnderline {
		out.DottedUnderline = se.DottedUnderline
	}
	return out
}

// Inherit styles from ancestors.
//
// Ancestors should be provided from oldest, furthest away to newest, closest.
func (se StyleEntry) Inherit(ancestors ...StyleEntry) StyleEntry {
	out := se
	for i := len(ancestors) - 1; i >= 0; i-- {
		if out.NoInherit {
			return out
		}
		ancestor := ancestors[i]
		if colors.IsNil(out.themeColor) {
			out.Color = ancestor.Color
			out.themeColor = ancestor.themeColor
		}
		if colors.IsNil(out.themeBackground) {
			out.Background = ancestor.Background
			out.themeBackground = ancestor.themeBackground
		}
		if colors.IsNil(out.Border) {
			out.Border = ancestor.Border
		}
		if out.Bold == Pass {
			out.Bold = ancestor.Bold
		}
		if out.Italic == Pass {
			out.Italic = ancestor.Italic
		}
		if out.Underline == Pass {
			out.Underline = ancestor.Underline
		}
		if out.DottedUnderline == Pass {
			out.DottedUnderline = ancestor.DottedUnderline
		}
	}
	return out
}

func (se StyleEntry) IsZero() bool {
	return colors.IsNil(se.Color) && colors.IsNil(se.Background) && colors.IsNil(se.Border) && se.Bold == Pass && se.Italic == Pass && se.Underline == Pass && se.DottedUnderline == Pass && !se.NoInherit
}

///////////////////////////////////////////////////////////////////////////////////
//  Style

// Style is a full style map of styles for different token.Tokens tag values
type Style map[token.Tokens]*StyleEntry

// CopyFrom copies a style from source style
func (hs *Style) CopyFrom(ss *Style) {
	if ss == nil {
		return
	}
	*hs = make(Style, len(*ss))
	for k, v := range *ss {
		(*hs)[k] = v
	}
}

// TagRaw returns a StyleEntry for given tag without any inheritance of anything
// will be IsZero if not defined for this style
func (hs Style) TagRaw(tag token.Tokens) StyleEntry {
	if len(hs) == 0 {
		return StyleEntry{}
	}
	if se, has := hs[tag]; has {
		return *se
	}
	return StyleEntry{}
}

// Tag returns a StyleEntry for given Tag.
// Will try sub-category or category if an exact match is not found.
// does NOT add the background properties -- those are always kept separate.
func (hs Style) Tag(tag token.Tokens) StyleEntry {
	se := hs.TagRaw(tag).Inherit(
		hs.TagRaw(token.Text),
		hs.TagRaw(tag.Cat()),
		hs.TagRaw(tag.SubCat()))
	return se
}

// ToCSS generates a CSS style sheet for this style, by token.Tokens tag
func (hs Style) ToCSS() map[token.Tokens]string {
	css := map[token.Tokens]string{}
	for ht := range token.Names {
		entry := hs.Tag(ht)
		if entry.IsZero() {
			continue
		}
		css[ht] = entry.ToCSS()
	}
	return css
}

// ToProperties generates a list of key-value properties for this style.
func (hs Style) ToProperties() map[string]any {
	pr := map[string]any{}
	for ht, nm := range token.Names {
		entry := hs.Tag(ht)
		if entry.IsZero() {
			if tp, ok := Properties[ht]; ok {
				pr["."+nm] = tp
			}
			continue
		}
		pr["."+nm] = entry.ToProperties()
	}
	return pr
}

// Open hi style from a JSON-formatted file.
func (hs Style) OpenJSON(filename fsx.Filename) error {
	b, err := os.ReadFile(string(filename))
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		slog.Error(err.Error())
		return err
	}
	return json.Unmarshal(b, &hs)
}

// Save hi style to a JSON-formatted file.
func (hs Style) SaveJSON(filename fsx.Filename) error {
	b, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		slog.Error(err.Error()) // unlikely
		return err
	}
	err = os.WriteFile(string(filename), b, 0644)
	if err != nil {
		// PromptDialog(nil, "Could not Save to File", err.Error(), true, false, nil, nil, nil)
		slog.Error(err.Error())
	}
	return err
}

// Properties are default properties for custom tags (tokens); if set in style then used
// there but otherwise we use these as a fallback; typically not overridden
var Properties = map[token.Tokens]map[string]any{
	token.TextSpellErr: {
		"text-decoration": 1 << uint32(rich.DottedUnderline), // bitflag!
	},
}

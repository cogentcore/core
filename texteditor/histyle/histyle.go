// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package histyle provides syntax highlighting styles; it is based on
// github.com/alecthomas/chroma, which in turn was based on the python
// pygments package.  Note that this package depends on goki/gi and goki/pi
// and cannot be imported there; is imported into goki/gi/giv.
package histyle

//go:generate core generate -add-types

import (
	"encoding/json"
	"image/color"
	"log/slog"
	"os"
	"strings"

	"cogentcore.org/core/cam/hct"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/pi/token"
	"cogentcore.org/core/styles"
)

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

// StyleEntry is one value in the map of highlight style values
type StyleEntry struct {

	// text color
	Color color.RGBA

	// background color
	Background color.RGBA

	// border color? not sure what this is -- not really used
	Border color.RGBA `view:"-"`

	// bold font
	Bold Trilean

	// italic font
	Italic Trilean

	// underline
	Underline Trilean

	// don't inherit these settings from sub-category or category levels -- otherwise everything with a Pass is inherited
	NoInherit bool
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

// Norm normalizes the colors of the style entry such that they have consistent
// chromas and tones that guarantee sufficient text contrast.
func (s *StyleEntry) Norm() {
	hc := hct.FromColor(s.Color)
	ctone := float32(40)
	if matcolor.SchemeIsDark {
		ctone = 80
	}
	s.Color = hc.WithChroma(max(hc.Chroma, 48)).WithTone(ctone).AsRGBA()

	if !colors.IsNil(s.Background) {
		hb := hct.FromColor(s.Background)
		btone := max(hb.Tone, 94)
		if matcolor.SchemeIsDark {
			btone = min(hb.Tone, 17)
		}
		s.Background = hb.WithChroma(max(hb.Chroma, 6)).WithTone(btone).AsRGBA()
	}
}

func (s StyleEntry) String() string {
	var out []string
	if s.Bold != Pass {
		out = append(out, s.Bold.Prefix("bold"))
	}
	if s.Italic != Pass {
		out = append(out, s.Italic.Prefix("italic"))
	}
	if s.Underline != Pass {
		out = append(out, s.Underline.Prefix("underline"))
	}
	if s.NoInherit {
		out = append(out, "noinherit")
	}
	if !colors.IsNil(s.Color) {
		out = append(out, colors.AsString(s.Color))
	}
	if !colors.IsNil(s.Background) {
		out = append(out, "bg:"+colors.AsString(s.Background))
	}
	if !colors.IsNil(s.Border) {
		out = append(out, "border:"+colors.AsString(s.Border))
	}
	return strings.Join(out, " ")
}

// ToCSS converts StyleEntry to CSS attributes.
func (s StyleEntry) ToCSS() string {
	var styles []string
	if !colors.IsNil(s.Color) {
		styles = append(styles, "color: "+colors.AsString(s.Color))
	}
	if !colors.IsNil(s.Background) {
		styles = append(styles, "background-color: "+colors.AsString(s.Background))
	}
	if s.Bold == Yes {
		styles = append(styles, "font-weight: bold")
	}
	if s.Italic == Yes {
		styles = append(styles, "font-style: italic")
	}
	if s.Underline == Yes {
		styles = append(styles, "text-decoration: underline")
	}
	return strings.Join(styles, "; ")
}

// ToProps converts StyleEntry to ki.Props attributes.
func (s *StyleEntry) ToProps() *ki.Props {
	pr := ki.NewProps()
	if !colors.IsNil(s.Color) {
		pr.Set("color", s.Color)
	}
	if !colors.IsNil(s.Background) {
		pr.Set("background-color", s.Background)
	}
	if s.Bold == Yes {
		pr.Set("font-weight", styles.WeightBold)
	}
	if s.Italic == Yes {
		pr.Set("font-style", styles.Italic)
	}
	if s.Underline == Yes {
		pr.Set("text-decoration", 1<<uint32(styles.Underline))
	}
	return pr
}

// Sub subtracts two style entries, returning an entry with only the differences set
func (s StyleEntry) Sub(e StyleEntry) StyleEntry {
	out := StyleEntry{}
	if e.Color != s.Color {
		out.Color = s.Color
	}
	if e.Background != s.Background {
		out.Background = s.Background
	}
	if e.Border != s.Border {
		out.Border = s.Border
	}
	if e.Bold != s.Bold {
		out.Bold = s.Bold
	}
	if e.Italic != s.Italic {
		out.Italic = s.Italic
	}
	if e.Underline != s.Underline {
		out.Underline = s.Underline
	}
	return out
}

// Inherit styles from ancestors.
//
// Ancestors should be provided from oldest, furthest away to newest, closest.
func (s StyleEntry) Inherit(ancestors ...StyleEntry) StyleEntry {
	out := s
	for i := len(ancestors) - 1; i >= 0; i-- {
		if out.NoInherit {
			return out
		}
		ancestor := ancestors[i]
		if colors.IsNil(out.Color) {
			out.Color = ancestor.Color
		}
		if colors.IsNil(out.Background) {
			out.Background = ancestor.Background
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
	}
	return out
}

func (s StyleEntry) IsZero() bool {
	return colors.IsNil(s.Color) && colors.IsNil(s.Background) && colors.IsNil(s.Border) && s.Bold == Pass && s.Italic == Pass &&
		s.Underline == Pass && !s.NoInherit
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

// ToProps generates list of ki.Props for this style
func (hs *Style) ToProps() *ki.Props {
	pr := ki.NewProps()
	for ht, nm := range token.Names {
		entry := hs.Tag(ht)
		if entry.IsZero() {
			if tp, ok := Props[ht]; ok {
				pr.Set("."+nm, tp)
			}
			continue
		}
		pr.Set("."+nm, entry.ToProps())
	}
	return pr
}

// OpenJSON hi style from a JSON-formatted file.
func (hs Style) OpenJSON(filename gi.Filename) error {
	b, err := os.ReadFile(string(filename))
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		slog.Error(err.Error())
		return err
	}
	return json.Unmarshal(b, &hs)
}

// SaveJSON hi style to a JSON-formatted file.
func (hs Style) SaveJSON(filename gi.Filename) error {
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

// TagsProps are default properties for custom tags (tokens) -- if set in style then used
// there but otherwise we use these as a fallback -- typically not overridden
var Props = map[token.Tokens]map[string]any{
	token.TextSpellErr: {
		"text-decoration": 1 << uint32(styles.DecoDottedUnderline), // bitflag!
	},
}

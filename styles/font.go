// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"
	"log/slog"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles/units"
)

// IMPORTANT: any changes here must be updated in style_properties.go StyleFontFuncs

// Font contains all font styling information.
// Most of font information is inherited.
// Font does not include all information needed
// for rendering -- see [FontRender] for that.
type Font struct { //types:add

	// size of font to render (inherited); converted to points when getting font to use
	Size units.Value

	// font family (inherited): ordered list of comma-separated names from more general to more specific to use; use split on , to parse
	Family string

	// style (inherited): normal, italic, etc
	Style FontStyles

	// weight (inherited): normal, bold, etc
	Weight FontWeights

	// font stretch / condense options (inherited)
	Stretch FontStretch

	// normal or small caps (inherited)
	Variant FontVariants

	// underline, line-through, etc (not inherited)
	Decoration TextDecorations

	// super / sub script (not inherited)
	Shift BaselineShifts

	// full font information including enhanced metrics and actual font codes for drawing text; this is a pointer into FontLibrary of loaded fonts
	Face *FontFace `display:"-"`
}

func (fs *Font) Defaults() {
	fs.Size = units.Dp(16)
}

// InheritFields from parent
func (fs *Font) InheritFields(parent *Font) {
	// fs.Color = par.Color
	fs.Family = parent.Family
	fs.Style = parent.Style
	if parent.Size.Value != 0 {
		fs.Size = parent.Size
	}
	fs.Weight = parent.Weight
	fs.Stretch = parent.Stretch
	fs.Variant = parent.Variant
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (fs *Font) ToDots(uc *units.Context) {
	if fs.Size.Unit == units.UnitEm || fs.Size.Unit == units.UnitEx || fs.Size.Unit == units.UnitCh {
		slog.Error("girl/styles.Font.Size was set to Em, Ex, or Ch; that is recursive and unstable!", "unit", fs.Size.Unit)
		fs.Size.Dp(16)
	}
	fs.Size.ToDots(uc)
}

// SetDecoration sets text decoration (underline, etc),
// which uses bitflags to allow multiple combinations.
func (fs *Font) SetDecoration(deco ...TextDecorations) {
	for _, d := range deco {
		fs.Decoration.SetFlag(true, d)
	}
}

// SetUnitContext sets the font-specific information in the given
// units.Context, based on the currently loaded face.
func (fs *Font) SetUnitContext(uc *units.Context) {
	if fs.Face != nil {
		uc.SetFont(fs.Face.Metrics.Em, fs.Face.Metrics.Ex, fs.Face.Metrics.Ch, uc.Dp(16))
	}
}

func (fs *Font) StyleFromProperties(parent *Font, properties map[string]any, ctxt colors.Context) {
	for key, val := range properties {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if sfunc, ok := styleFontFuncs[key]; ok {
			sfunc(fs, key, val, parent, ctxt)
		}
	}
}

// SetStyleProperties sets font style values based on given property map (name:
// value pairs), inheriting elements as appropriate from parent, and also
// having a default style for the "initial" setting.
func (fs *Font) SetStyleProperties(parent *Font, properties map[string]any, ctxt colors.Context) {
	// direct font styling is used only for special cases -- don't do this:
	// if !fs.StyleSet && parent != nil { // first time
	// 	fs.InheritFields(parent)
	// }
	fs.StyleFromProperties(parent, properties, ctxt)
}

//////////////////////////////////////////////////////////////////////////////////
// Font Style enums

// TODO: should we keep FontSizePoints?

// FontSizePoints maps standard font names to standard point sizes -- we use
// dpi zoom scaling instead of rescaling "medium" font size, so generally use
// these values as-is.  smaller and larger relative scaling can move in 2pt increments
var FontSizePoints = map[string]float32{
	"xx-small": 7,
	"x-small":  7.5,
	"small":    10, // small is also "smaller"
	"smallf":   10, // smallf = small font size..
	"medium":   12,
	"large":    14,
	"x-large":  18,
	"xx-large": 24,
}

// FontStyles styles of font: normal, italic, etc
type FontStyles int32 //enums:enum -trim-prefix Font -transform kebab

const (
	FontNormal FontStyles = iota

	// Italic indicates to make font italic
	Italic

	// Oblique indicates to make font slanted
	Oblique
)

// FontStyleNames contains the uppercase names of all the valid font styles
// used in the regularized font names.  The first name is the baseline default
// and will be omitted from font names.
var FontStyleNames = []string{"Normal", "Italic", "Oblique"}

// FontWeights are the valid names for different weights of font, with both
// the numeric and standard names given.  The regularized font names in the
// font library use the names, as those are typically found in the font files.
type FontWeights int32 //enums:enum -trim-prefix Weight -transform kebab

const (
	WeightNormal FontWeights = iota
	Weight100
	WeightThin // (Hairline)
	Weight200
	WeightExtraLight // (UltraLight)
	Weight300
	WeightLight
	Weight400
	Weight500
	WeightMedium
	Weight600
	WeightSemiBold // (DemiBold)
	Weight700
	WeightBold
	Weight800
	WeightExtraBold // (UltraBold)
	Weight900
	WeightBlack
	WeightBolder
	WeightLighter
)

// FontWeightNames contains the uppercase names of all the valid font weights
// used in the regularized font names.  The first name is the baseline default
// and will be omitted from font names. Order must have names that are subsets
// of other names at the end so they only match if the more specific one
// hasn't!
var FontWeightNames = []string{"Normal", "Thin", "ExtraLight", "Light", "Medium", "SemiBold", "ExtraBold", "Bold", "Black"}

// FontWeightNameValues is 1-to-1 index map from FontWeightNames to
// corresponding weight value (using more semantic term instead of numerical
// one)
var FontWeightNameValues = []FontWeights{WeightNormal, WeightThin, WeightExtraLight, WeightLight, WeightMedium, WeightSemiBold, WeightExtraBold, WeightBold, WeightBlack}

// FontWeightToNameMap maps all the style enums to canonical regularized font names
var FontWeightToNameMap = map[FontWeights]string{
	Weight100:        "Thin",
	WeightThin:       "Thin",
	Weight200:        "ExtraLight",
	WeightExtraLight: "ExtraLight",
	Weight300:        "Light",
	WeightLight:      "Light",
	Weight400:        "",
	WeightNormal:     "",
	Weight500:        "Medium",
	WeightMedium:     "Medium",
	Weight600:        "SemiBold",
	WeightSemiBold:   "SemiBold",
	Weight700:        "Bold",
	WeightBold:       "Bold",
	Weight800:        "ExtraBold",
	WeightExtraBold:  "ExtraBold",
	Weight900:        "Black",
	WeightBlack:      "Black",
	WeightBolder:     "Medium", // todo: lame but assumes normal and goes one bolder
	WeightLighter:    "Light",  // todo: lame but assumes normal and goes one lighter
}

// FontStretch are different stretch levels of font.  These are less typically
// available on most platforms by default.
type FontStretch int32 //enums:enum -trim-prefix FontStr

const (
	FontStrNormal FontStretch = iota
	FontStrUltraCondensed
	FontStrExtraCondensed
	FontStrSemiCondensed
	FontStrSemiExpanded
	FontStrExtraExpanded
	FontStrUltraExpanded
	FontStrCondensed
	FontStrExpanded
	FontStrNarrower
	FontStrWider
)

// FontStretchNames contains the uppercase names of all the valid font
// stretches used in the regularized font names.  The first name is the
// baseline default and will be omitted from font names.  Order must have
// names that are subsets of other names at the end so they only match if the
// more specific one hasn't!  And also match the FontStretch enum.
var FontStretchNames = []string{"Normal", "UltraCondensed", "ExtraCondensed", "SemiCondensed", "SemiExpanded", "ExtraExpanded", "UltraExpanded", "Condensed", "Expanded", "Condensed", "Expanded"}

// TextDecorations are underline, line-through, etc -- operates as bit flags
// -- also used for additional layout hints for RuneRender
type TextDecorations int64 //enums:bitflag -trim-prefix Deco -transform kebab

const (
	DecoNone TextDecorations = iota

	// Underline indicates to place a line below text
	Underline

	// Overline indicates to place a line above text
	Overline

	// LineThrough indicates to place a line through text
	LineThrough

	// Blink is not currently supported (and probably a bad idea generally ;)
	DecoBlink

	// DottedUnderline is used for abbr tag -- otherwise not a standard text-decoration option afaik
	DecoDottedUnderline

	// following are special case layout hints in RuneRender, to pass
	// information from a styling pass to a subsequent layout pass -- they are
	// NOT processed during final rendering

	// DecoParaStart at start of a SpanRender indicates that it should be
	// styled as the start of a new paragraph and not just the start of a new
	// line
	DecoParaStart
	// DecoSuper indicates super-scripted text
	DecoSuper
	// DecoSub indicates sub-scripted text
	DecoSub
	// DecoBackgroundColor indicates that a bg color has been set -- for use in optimizing rendering
	DecoBackgroundColor
)

// BaselineShifts are for super / sub script
type BaselineShifts int32 //enums:enum -trim-prefix Shift -transform kebab

const (
	ShiftBaseline BaselineShifts = iota
	ShiftSuper
	ShiftSub
)

// FontVariants is just normal vs. small caps. todo: not currently supported
type FontVariants int32 //enums:enum -trim-prefix FontVar -transform kebab

const (
	FontVarNormal FontVariants = iota
	FontVarSmallCaps
)

// FontNameToMods parses the regularized font name and returns the appropriate
// base name and associated font mods.
func FontNameToMods(fn string) (basenm string, str FontStretch, wt FontWeights, sty FontStyles) {
	basenm = fn
	for mi, mod := range FontStretchNames {
		spmod := " " + mod
		if strings.Contains(fn, spmod) {
			str = FontStretch(mi)
			basenm = strings.Replace(basenm, spmod, "", 1)
			break
		}
	}
	for mi, mod := range FontWeightNames {
		spmod := " " + mod
		if strings.Contains(fn, spmod) {
			wt = FontWeightNameValues[mi]
			basenm = strings.Replace(basenm, spmod, "", 1)
			break
		}
	}
	for mi, mod := range FontStyleNames {
		spmod := " " + mod
		if strings.Contains(fn, spmod) {
			sty = FontStyles(mi)
			basenm = strings.Replace(basenm, spmod, "", 1)
			break
		}
	}
	return
}

// FontNameFromMods generates the appropriate regularized file name based on
// base name and modifiers
func FontNameFromMods(basenm string, str FontStretch, wt FontWeights, sty FontStyles) string {
	fn := basenm
	if str != FontStrNormal {
		fn += " " + FontStretchNames[str]
	}
	if wt != WeightNormal && wt != Weight400 {
		fn += " " + FontWeightToNameMap[wt]
	}
	if sty != FontNormal {
		fn += " " + FontStyleNames[sty]
	}
	return fn
}

// FixFontMods ensures that standard font modifiers have a space in front of
// them, and that the default is not in the name -- used for regularizing font
// names.
func FixFontMods(fn string) string {
	for mi, mod := range FontStretchNames {
		if bi := strings.Index(fn, mod); bi > 0 {
			if fn[bi-1] != ' ' {
				fn = strings.Replace(fn, mod, " "+mod, 1)
			}
			if mi == 0 { // default, remove
				fn = strings.Replace(fn, " "+mod, "", 1)
			}
			break // critical to break to prevent subsets from matching
		}
	}
	for mi, mod := range FontWeightNames {
		if bi := strings.Index(fn, mod); bi > 0 {
			if fn[bi-1] != ' ' {
				fn = strings.Replace(fn, mod, " "+mod, 1)
			}
			if mi == 0 { // default, remove
				fn = strings.Replace(fn, " "+mod, "", 1)
			}
			break // critical to break to prevent subsets from matching
		}
	}
	for mi, mod := range FontStyleNames {
		if bi := strings.Index(fn, mod); bi > 0 {
			if fn[bi-1] != ' ' {
				fn = strings.Replace(fn, mod, " "+mod, 1)
			}
			if mi == 0 { // default, remove
				fn = strings.Replace(fn, " "+mod, "", 1)
			}
			break // critical to break to prevent subsets from matching
		}
	}
	// also get rid of Regular!
	fn = strings.TrimSuffix(fn, " Regular")
	fn = strings.TrimSuffix(fn, "Regular")
	return fn
}

// FontRender contains all font styling information
// that is needed for SVG text rendering. It is passed to
// Paint and Style functions. It should typically not be
// used by end-user code -- see [Font] for that.
// It stores all values as pointers so that they correspond
// to the values of the style object it was derived from.
type FontRender struct { //types:add
	Font

	// text color (inherited)
	Color image.Image

	// background color (not inherited, transparent by default)
	Background image.Image

	// alpha value between 0 and 1 to apply to the foreground and background of this element and all of its children
	Opacity float32
}

// FontRender returns the font-rendering-related
// styles of the style object as a FontRender
func (s *Style) FontRender() *FontRender {
	return &FontRender{
		Font:  s.Font,
		Color: s.Color,
		// we do NOT set the BackgroundColor because the label renders its own background color
		// STYTODO(kai): this might cause problems with inline span styles
		Opacity: s.Opacity,
	}
}

func (fr *FontRender) Defaults() {
	fr.Color = colors.Scheme.OnSurface
	fr.Opacity = 1
	fr.Font.Defaults()
}

// InheritFields from parent
func (fr *FontRender) InheritFields(parent *FontRender) {
	fr.Color = parent.Color
	fr.Opacity = parent.Opacity
	fr.Font.InheritFields(&parent.Font)
}

// SetStyleProperties sets font style values based on given property map (name:
// value pairs), inheriting elements as appropriate from parent, and also
// having a default style for the "initial" setting.
func (fr *FontRender) SetStyleProperties(parent *FontRender, properties map[string]any, ctxt colors.Context) {
	var pfont *Font
	if parent != nil {
		pfont = &parent.Font
	}
	fr.Font.StyleFromProperties(pfont, properties, ctxt)
	fr.StyleRenderFromProperties(parent, properties, ctxt)
}

func (fs *FontRender) StyleRenderFromProperties(parent *FontRender, properties map[string]any, ctxt colors.Context) {
	for key, val := range properties {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if sfunc, ok := styleFontRenderFuncs[key]; ok {
			sfunc(fs, key, val, parent, ctxt)
		}
	}
}

// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"strings"

	"github.com/goki/gi/units"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// IMPORTANT: any changes here must be updated in style_props.go StyleFontFuncs

// Font contains all font styling information, including everything that
// is used in SVG text rendering -- used in Paint and in Style. Most of font
// information is inherited.
type Font struct {
	Color   Color           `xml:"color" inherit:"true" desc:"prop: color (inherited) = text color -- also defines the currentColor variable value"`
	BgColor ColorSpec       `xml:"background-color" desc:"prop: background-color = background color -- not inherited, transparent by default"`
	Opacity float32         `xml:"opacity" desc:"prop: opacity = alpha value to apply to all elements"`
	Size    units.Value     `xml:"font-size" inherit:"true" desc:"prop: font-size (inherited)= size of font to render -- convert to points when getting font to use"`
	Family  string          `xml:"font-family" inherit:"true" desc:"prop: font-family = font family -- ordered list of comma-separated names from more general to more specific to use -- use split on , to parse"`
	Style   FontStyles      `xml:"font-style" inherit:"true" desc:"prop: font-style = style -- normal, italic, etc"`
	Weight  FontWeights     `xml:"font-weight" inherit:"true" desc:"prop: font-weight = weight: normal, bold, etc"`
	Stretch FontStretch     `xml:"font-stretch" inherit:"true" desc:"prop: font-stretch = font stretch / condense options"`
	Variant FontVariants    `xml:"font-variant" inherit:"true" desc:"prop: font-variant = normal or small caps"`
	Deco    TextDecorations `xml:"text-decoration" desc:"prop: text-decoration = underline, line-through, etc -- not inherited"`
	Shift   BaselineShifts  `xml:"baseline-shift" desc:"prop: baseline-shift = super / sub script -- not inherited"`
	Face    *FontFace       `view:"-" desc:"full font information including enhanced metrics and actual font codes for drawing text -- this is a pointer into FontLibrary of loaded fonts"`
	Rem     float32         `desc:"Rem size of font -- 12pt converted to same effective DPI as above measurements"`
	// todo: kerning
	// todo: stretch -- css 3 -- not supported
}

func (fs *Font) Defaults() {
	fs.Color = Black
	fs.Opacity = 1.0
	fs.Size = units.NewPt(12)
}

// SetStylePost does any updates after generic xml-tag property setting -- use
// for anything that also has non-standard values that might not be processed
// properly by default
func (fs *Font) SetStylePost(props ki.Props) {
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (fs *Font) InheritFields(par *Font) {
	fs.Color = par.Color
	fs.Family = par.Family
	fs.Style = par.Style
	if par.Size.Val != 0 {
		fs.Size = par.Size
	}
	fs.Weight = par.Weight
	fs.Stretch = par.Stretch
	fs.Variant = par.Variant
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (fs *Font) ToDots(uc *units.Context) {
	fs.Size.ToDots(uc)
}

// SetDeco sets decoration (underline, etc), which uses bitflag to allow multiple combinations
func (fs *Font) SetDeco(deco TextDecorations) {
	bitflag.Set32((*int32)(&fs.Deco), int(deco))
}

// ClearDeco clears decoration (underline, etc), which uses bitflag to allow
// multiple combinations
func (fs *Font) ClearDeco(deco TextDecorations) {
	bitflag.Clear32((*int32)(&fs.Deco), int(deco))
}

// SetUnitContext sets the font-specific information in the given
// units.Context, based on the currently-loaded face.
func (fs *Font) SetUnitContext(ctxt *units.Context) {
	if fs.Face != nil {
		ctxt.SetFont(fs.Face.Metrics.Em, fs.Face.Metrics.Ex, fs.Face.Metrics.Ch, fs.Rem)
	}
}

func (fs *Font) StyleFromProps(par *Font, props ki.Props, ctxt Context) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if sfunc, ok := StyleFontFuncs[key]; ok {
			sfunc(fs, key, val, par, ctxt)
		}
	}
}

// SetStyleProps sets font style values based on given property map (name:
// value pairs), inheriting elements as appropriate from parent, and also
// having a default style for the "initial" setting.
func (fs *Font) SetStyleProps(parent *Font, props ki.Props, ctxt Context) {
	// direct font styling is used only for special cases -- don't do this:
	// if !fs.StyleSet && parent != nil { // first time
	// 	fs.InheritFields(parent)
	// }
	fs.StyleFromProps(parent, props, ctxt)
	fs.SetStylePost(props)
}

// CopyNonDefaultProps does SetProp on given node for all of the style settings
// that are not at their default values.
func (fs *Font) CopyNonDefaultProps(node ki.Ki) {
	prefclr := ThePrefs.PrefColor("font")
	preffont := ThePrefs.PrefFontFamily()
	if fs.Color != *prefclr {
		node.SetProp("color", fs.Color)
	}
	if !fs.BgColor.Color.IsNil() {
		node.SetProp("background-color", fs.BgColor.Color)
	}
	if fs.Opacity != 1 {
		node.SetProp("opacity", fs.Opacity)
	}
	if fs.Family != "" && fs.Family != preffont {
		node.SetProp("font-family", fs.Family)
	}
	if fs.Style != FontNormal {
		node.SetProp("font-style", fs.Style)
	}
	if fs.Weight != WeightNormal {
		node.SetProp("font-weight", fs.Weight)
	}
	if fs.Stretch != FontStrNormal {
		node.SetProp("font-stretch", fs.Stretch)
	}
	if fs.Variant != FontVarNormal {
		node.SetProp("font-variant", fs.Variant)
	}
	if fs.Deco != DecoNone {
		node.SetProp("font-decoration", fs.Deco)
	}
	if fs.Shift != ShiftBaseline {
		node.SetProp("baseline-shift", fs.Shift)
	}
}

//////////////////////////////////////////////////////////////////////////////////
// Font Style enums

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
type FontStyles int32

const (
	FontNormal FontStyles = iota
	FontItalic
	FontOblique
	FontStylesN
)

//go:generate stringer -type=FontStyles

var KiT_FontStyles = kit.Enums.AddEnumAltLower(FontStylesN, kit.NotBitFlag, StylePropProps, "Font")

func (ev FontStyles) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FontStyles) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// FontStyleNames contains the uppercase names of all the valid font styles
// used in the regularized font names.  The first name is the baseline default
// and will be omitted from font names.
var FontStyleNames = []string{"Normal", "Italic", "Oblique"}

// FontWeights are the valid names for different weights of font, with both
// the numeric and standard names given.  The regularized font names in the
// font library use the names, as those are typically found in the font files.
type FontWeights int32

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
	FontWeightsN
)

//go:generate stringer -type=FontWeights

var KiT_FontWeights = kit.Enums.AddEnumAltLower(FontWeightsN, kit.NotBitFlag, StylePropProps, "Weight")

func (ev FontWeights) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FontWeights) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// FontWeightNames contains the uppercase names of all the valid font weights
// used in the regularized font names.  The first name is the baseline default
// and will be omitted from font names. Order must have names that are subsets
// of other names at the end so they only match if the more specific one
// hasn't!
var FontWeightNames = []string{"Normal", "Thin", "ExtraLight", "Light", "Medium", "SemiBold", "ExtraBold", "Bold", "Black"}

// FontWeightNameVals is 1-to-1 index map from FontWeightNames to
// corresponding weight value (using more semantic term instead of numerical
// one)
var FontWeightNameVals = []FontWeights{WeightNormal, WeightThin, WeightExtraLight, WeightLight, WeightMedium, WeightSemiBold, WeightExtraBold, WeightBold, WeightBlack}

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
type FontStretch int32

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
	FontStretchN
)

//go:generate stringer -type=FontStretch

var KiT_FontStretch = kit.Enums.AddEnumAltLower(FontStretchN, kit.NotBitFlag, StylePropProps, "FontStr")

func (ev FontStretch) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FontStretch) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// FontStretchNames contains the uppercase names of all the valid font
// stretches used in the regularized font names.  The first name is the
// baseline default and will be omitted from font names.  Order must have
// names that are subsets of other names at the end so they only match if the
// more specific one hasn't!  And also match the FontStretch enum.
var FontStretchNames = []string{"Normal", "UltraCondensed", "ExtraCondensed", "SemiCondensed", "SemiExpanded", "ExtraExpanded", "UltraExpanded", "Condensed", "Expanded", "Condensed", "Expanded"}

// TextDecorations are underline, line-through, etc -- operates as bit flags
// -- also used for additional layout hints for RuneRender
type TextDecorations int32

const (
	DecoNone TextDecorations = iota
	DecoUnderline
	DecoOverline
	DecoLineThrough
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
	// DecoBgColor indicates that a bg color has been set -- for use in optimizing rendering
	DecoBgColor
	TextDecorationsN
)

//go:generate stringer -type=TextDecorations

var KiT_TextDecorations = kit.Enums.AddEnumAltLower(TextDecorationsN, kit.BitFlag, StylePropProps, "Deco")

func (ev TextDecorations) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *TextDecorations) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// BaselineShifts are for super / sub script
type BaselineShifts int32

const (
	ShiftBaseline BaselineShifts = iota
	ShiftSuper
	ShiftSub
	BaselineShiftsN
)

//go:generate stringer -type=BaselineShifts

var KiT_BaselineShifts = kit.Enums.AddEnumAltLower(BaselineShiftsN, kit.NotBitFlag, StylePropProps, "Shift")

func (ev BaselineShifts) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *BaselineShifts) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// FontVariants is just normal vs. small caps. todo: not currently supported
type FontVariants int32

const (
	FontVarNormal FontVariants = iota
	FontVarSmallCaps
	FontVariantsN
)

//go:generate stringer -type=FontVariants

var KiT_FontVariants = kit.Enums.AddEnumAltLower(FontVariantsN, kit.NotBitFlag, StylePropProps, "FontVar")

func (ev FontVariants) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FontVariants) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

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
			wt = FontWeightNameVals[mi]
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

// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"strings"
	"sync"

	"github.com/fatih/camelcase"
	"github.com/goki/gi/units"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/iancoleman/strcase"
)

// IMPORTANT: any changes here must be updated in stylefuncs.go StyleFontFuncs

// FontStyle contains all font styling information, including everything that
// is used in SVG text rendering -- used in Paint and in Style. Most of font
// information is inherited.
type FontStyle struct {
	Color   Color           `xml:"color" inherit:"true" desc:"prop: color = text color -- also defines the currentColor variable value"`
	BgColor ColorSpec       `xml:"background-color" desc:"prop: background-color = background color -- not inherited, transparent by default"`
	Opacity float32         `xml:"opacity" desc:"prop: opacity = alpha value to apply to all elements"`
	Size    units.Value     `xml:"font-size" inherit:"true" desc:"prop: font-size = size of font to render -- convert to points when getting font to use"`
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

func (fs *FontStyle) Defaults() {
	fs.Color.SetColor(color.Black)
	fs.Opacity = 1.0
	fs.Size = units.NewPt(12)
}

// SetStylePost does any updates after generic xml-tag property setting -- use
// for anything that also has non-standard values that might not be processed
// properly by default
func (fs *FontStyle) SetStylePost(props ki.Props) {
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (fs *FontStyle) InheritFields(par *FontStyle) {
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

// SetDeco sets decoration (underline, etc), which uses bitflag to allow multiple combinations
func (fs *FontStyle) SetDeco(deco TextDecorations) {
	bitflag.Set32((*int32)(&fs.Deco), int(deco))
}

// ClearDeco clears decoration (underline, etc), which uses bitflag to allow
// multiple combinations
func (fs *FontStyle) ClearDeco(deco TextDecorations) {
	bitflag.Clear32((*int32)(&fs.Deco), int(deco))
}

// FontFallbacks are a list of fallback fonts to try, at the basename level.
// Make sure there are no loops!  Include Noto versions of everything in this
// because they have the most stretch options, so they should be in the mix if
// they have been installed, and include "Go" options last.
var FontFallbacks = map[string]string{
	"serif":            "Times New Roman",
	"times":            "Times New Roman",
	"Times New Roman":  "Liberation Serif",
	"Liberation Serif": "NotoSerif",
	"sans-serif":       "NotoSans",
	"NotoSans":         "Go",
	"courier":          "Courier",
	"Courier":          "Courier New",
	"Courier New":      "NotoSansMono",
	"NotoSansMono":     "Go Mono",
	"monospace":        "NotoSansMono",
	"cursive":          "Comic Sans", // todo: look up more of these
	"Comic Sans":       "Comic Sans MS",
	"fantasy":          "Impact",
	"Impact":           "Impac",
}

func addUniqueFont(fns *[]string, fn string) bool {
	sz := len(*fns)
	for i := 0; i < sz; i++ {
		if (*fns)[i] == fn {
			return false
		}
	}
	*fns = append(*fns, fn)
	return true
}

func addUniqueFontRobust(fns *[]string, fn string) bool {
	if FontLibrary.FontAvail(fn) {
		return addUniqueFont(fns, fn)
	}
	camel := strcase.ToCamel(fn)
	if FontLibrary.FontAvail(camel) {
		return addUniqueFont(fns, camel)
	}
	spc := strings.Join(camelcase.Split(camel), " ")
	if FontLibrary.FontAvail(spc) {
		return addUniqueFont(fns, spc)
	}
	return false
}

// FontSerifMonoGuess looks at a list of alternative font names and tires to
// guess if the font is a serif (vs sans) or monospaced (vs proportional)
// font.
func FontSerifMonoGuess(fns []string) (serif, mono bool) {
	for _, fn := range fns {
		lfn := strings.ToLower(fn)
		if strings.Contains(lfn, "serif") {
			serif = true
		}
		if strings.Contains(lfn, "mono") || lfn == "menlo" || lfn == "courier" || lfn == "courier new" || strings.Contains(lfn, "typewriter") {
			mono = true
		}
	}
	return
}

// FontAlts generates a list of all possible alternative fonts that actually
// exist in font library for a list of font families, and a guess as to
// whether the font is a serif (vs sans) or monospaced (vs proportional) font.
// Only deals with base names.
func FontAlts(fams string) (fns []string, serif, mono bool) {
	nms := strings.Split(fams, ",")
	if len(nms) == 0 {
		fn := Prefs.FontFamily
		if fn == "" {
			fns = []string{"Go"}
			return
		}
	}
	fns = make([]string, 0, 20)
	for _, fn := range nms {
		fn = strings.TrimSpace(fn)
		basenm, _, _, _ := FontNameToMods(fn)
		addUniqueFontRobust(&fns, basenm)
	altsloop:
		for {
			altfn, ok := FontFallbacks[basenm]
			if !ok {
				break altsloop
			}
			addUniqueFontRobust(&fns, altfn)
			basenm = altfn
		}
	}

	serif, mono = FontSerifMonoGuess(fns)

	// final baseline backups
	if mono {
		addUniqueFont(&fns, "NotoSansMono") // has more options
		addUniqueFont(&fns, "Go Mono")      // just as good as liberation mono..
	} else if serif {
		addUniqueFont(&fns, "Liberation Serif")
		addUniqueFont(&fns, "NotoSerif")
		addUniqueFont(&fns, "Go") // not serif but drop dead backup
	} else {
		addUniqueFont(&fns, "NotoSans")
		addUniqueFont(&fns, "Go") // good as anything
	}

	return
}

// FaceName returns the full FaceName to use for the current FontStyle spec, robustly
func (fs *FontStyle) FaceName() string {
	fnm := FontFaceName(fs.Family, fs.Stretch, fs.Weight, fs.Style)
	return fnm
}

// faceNameCache is a cache for fast lookup of valid font face names given style specs
var faceNameCache map[string]string

// faceNameCacheMu protects access to faceNameCache
var faceNameCacheMu sync.RWMutex

// FontFaceName returns the best full FaceName to use for the given font
// family(ies) (comma separated) and modifier parameters
func FontFaceName(fam string, str FontStretch, wt FontWeights, sty FontStyles) string {
	if fam == "" {
		fam = string(Prefs.FontFamily)
	}

	cacheNm := fam + "|" + str.String() + "|" + wt.String() + "|" + sty.String()
	faceNameCacheMu.RLock()
	if fc, has := faceNameCache[cacheNm]; has {
		faceNameCacheMu.RUnlock()
		return fc
	}
	faceNameCacheMu.RUnlock()

	nms := strings.Split(fam, ",")
	basenm := ""
	if len(nms) > 0 { // start off with any styles implicit in font name
		_, fstr, fwt, fsty := FontNameToMods(strings.TrimSpace(nms[0]))
		if fstr != FontStrNormal {
			str = fstr
		}
		if fwt != WeightNormal {
			wt = fwt
		}
		if fsty != FontNormal {
			sty = fsty
		}
	}

	nms, _, _ = FontAlts(fam) // nms are all base names now

	// we try multiple iterations, going through list of alternatives (which
	// should be from most specific to least, all of which have an existing
	// base name) -- first iter we look for an exact match for given
	// modifiers, then we start relaxing things in terms of most likely
	// issues..
	didItalic := false
	didOblique := false
iterloop:
	for iter := 0; iter < 10; iter++ {
		for _, basenm = range nms {
			fn := FontNameFromMods(basenm, str, wt, sty)
			if FontLibrary.FontAvail(fn) {
				break iterloop
			}
		}
		if str != FontStrNormal {
			hasStr := false
			for _, basenm = range nms {
				fn := FontNameFromMods(basenm, str, WeightNormal, FontNormal)
				if FontLibrary.FontAvail(fn) {
					hasStr = true
					break
				}
			}
			if !hasStr { // if even basic stretch not avail, move on
				str = FontStrNormal
				continue
			}
			continue
		}
		if sty == FontItalic { // italic is more common, but maybe oblique exists
			didItalic = true
			if !didOblique {
				sty = FontOblique
				continue
			}
			sty = FontNormal
			continue
		}
		if sty == FontOblique { // by now we've tried both, try nothing
			didOblique = true
			if !didItalic {
				sty = FontItalic
				continue
			}
			sty = FontNormal
			continue
		}
		if wt != WeightNormal {
			if wt < Weight400 {
				if wt != WeightLight {
					wt = WeightLight
					continue
				}
			} else {
				if wt != WeightBold {
					wt = WeightBold
					continue
				}
			}
			wt = WeightNormal
			continue
		}
		if str != FontStrNormal { // time to give up
			str = FontStrNormal
			continue
		}
		break // tried everything
	}
	fnm := FontNameFromMods(basenm, str, wt, sty)

	faceNameCacheMu.Lock()
	if faceNameCache == nil {
		faceNameCache = make(map[string]string)
	}
	faceNameCache[cacheNm] = fnm
	faceNameCacheMu.Unlock()

	return fnm
}

// OpenFont loads the font specified by the font style from the font library.
// This is the primary method to use for loading fonts, as it uses a robust
// fallback method to finding an appropriate font, and falls back on the
// builtin Go font as a last resort.  The Face field will have the resulting
// font.  The font size is always rounded to nearest integer, to produce
// better-looking results (presumably).  The current metrics and given
// unit.Context are updated based on the properties of the font.
func (fs *FontStyle) OpenFont(ctxt *units.Context) {
	facenm := fs.FaceName()
	if fs.Size.Dots == 0 {
		fs.Size.ToDots(ctxt)
	}
	intDots := int(math.Round(float64(fs.Size.Dots)))
	if intDots == 0 {
		fmt.Printf("FontStyle Error: bad font size: %v or units context: %v\n", fs.Size, *ctxt)
		intDots = 12
	}
	face, err := FontLibrary.Font(facenm, intDots)
	if err != nil {
		log.Printf("%v\n", err)
		if fs.Face == nil {
			face, err = FontLibrary.Font("Go", intDots) // guaranteed to exist
			fs.Face = face
		}
	} else {
		fs.Face = face
	}
	fs.Rem = ctxt.ToDots(12, units.Pt)
	fs.SetUnitContext(ctxt)
}

// SetUnitContext sets the font-specific information in the given
// units.Context, based on the currently-loaded face.
func (fs *FontStyle) SetUnitContext(ctxt *units.Context) {
	if fs.Face != nil {
		ctxt.SetFont(fs.Face.Metrics.Em, fs.Face.Metrics.Ex, fs.Face.Metrics.Ch, fs.Rem)
	}
}

// Style CSS looks for "tag" name props in cssAgg props, and applies those to
// style if found, and returns true -- false if no such tag found
func (fs *FontStyle) StyleCSS(tag string, cssAgg ki.Props, ctxt *units.Context, vp *Viewport2D) bool {
	if cssAgg == nil {
		return false
	}
	tp, ok := cssAgg[tag]
	if !ok {
		return false
	}
	pmap, ok := tp.(ki.Props) // must be a props map
	if !ok {
		return false
	}
	fs.SetStyleProps(nil, pmap, vp)
	fs.OpenFont(ctxt)
	return true
}

func (fs *FontStyle) StyleFromProps(par *FontStyle, props ki.Props, vp *Viewport2D) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if sfunc, ok := StyleFontFuncs[key]; ok {
			sfunc(fs, key, val, par, vp)
		}
	}
}

// SetStyleProps sets font style values based on given property map (name:
// value pairs), inheriting elements as appropriate from parent, and also
// having a default style for the "initial" setting.
func (fs *FontStyle) SetStyleProps(parent *FontStyle, props ki.Props, vp *Viewport2D) {
	// direct font styling is used only for special cases -- don't do this:
	// if !fs.StyleSet && parent != nil { // first time
	// 	fs.InheritFields(parent)
	// }
	fs.StyleFromProps(parent, props, vp)
	fs.SetStylePost(props)
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

var KiT_FontStyles = kit.Enums.AddEnumAltLower(FontStylesN, false, StylePropProps, "Font")

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

var KiT_FontWeights = kit.Enums.AddEnumAltLower(FontWeightsN, false, StylePropProps, "Weight")

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

var KiT_FontStretch = kit.Enums.AddEnumAltLower(FontStretchN, false, StylePropProps, "FontStr")

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

var KiT_TextDecorations = kit.Enums.AddEnumAltLower(TextDecorationsN, true, StylePropProps, "Deco") // true = bit flag

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

var KiT_BaselineShifts = kit.Enums.AddEnumAltLower(BaselineShiftsN, false, StylePropProps, "Shift")

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

var KiT_FontVariants = kit.Enums.AddEnumAltLower(FontVariantsN, false, StylePropProps, "FontVar")

func (ev FontVariants) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FontVariants) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

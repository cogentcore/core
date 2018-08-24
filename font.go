// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/chewxy/math32"
	"github.com/fatih/camelcase"
	"github.com/goki/freetype/truetype"
	"github.com/iancoleman/strcase"
	// "github.com/golang/freetype/truetype"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomediumitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gosmallcaps"
	"golang.org/x/image/font/gofont/gosmallcapsitalic"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// font.go contains all font and basic SVG-level text rendering styles, and the
// font library.  see text.go for rendering code

// FontName is used to specify an font -- just the unique name of the font
// family -- automtically provides a chooser menu for fonts using ValueView
// system
type FontName string

// FontStyle contains all font styling information, including everything that
// is used in SVG text rendering -- used in Paint and in Style. Most of font
// information is inherited.
type FontStyle struct {
	Color    Color           `xml:"color" inherit:"true" desc:"text color -- also defines the currentColor variable value"`
	BgColor  ColorSpec       `xml:"background-color" desc:"background color -- not inherited, transparent by default"`
	Opacity  float32         `xml:"opacity" desc:"alpha value to apply to all elements"`
	Size     units.Value     `xml:"font-size" desc:"size of font to render -- convert to points when getting font to use"`
	Family   string          `xml:"font-family" inherit:"true" desc:"font family -- ordered list of comma-separated names from more general to more specific to use -- use split on , to parse"`
	Style    FontStyles      `xml:"font-style" inherit:"true" desc:"style -- normal, italic, etc"`
	Weight   FontWeights     `xml:"font-weight" inherit:"true" desc:"weight: normal, bold, etc"`
	Stretch  FontStretch     `xml:"font-stretch" inherit:"true" desc:"font stretch / condense options"`
	Variant  FontVariants    `xml:"font-variant" inherit:"true" desc:"normal or small caps"`
	Deco     TextDecorations `xml:"text-decoration" desc:"underline, line-through, etc -- not inherited"`
	Shift    BaselineShifts  `xml:"baseline-shift" desc:"super / sub script -- not inherited"`
	Face     font.Face       `view:"-" desc:"actual font codes for drawing text -- just a pointer into FontLibrary of loaded fonts"`
	Height   float32         `desc:"reference 1.0 spacing line height of font in dots -- computed from font as ascent + descent + lineGap, where lineGap is specified by the font as the recommended line spacing"`
	Em       float32         `desc:"Em size of font -- this is NOT actually the width of the letter M, but rather the specified point size of the font (in actual display dots, not points) -- it does NOT include the descender and will not fit the entire height of the font"`
	Ex       float32         `desc:"Ex size of font -- this is the actual height of the letter x in the font"`
	Ch       float32         `desc:"Ch size of font -- this is the actual width of the 0 glyph in the font"`
	Rem      float32         `desc:"Rem size of font -- 12pt converted to same effective DPI as above measurements"`
	FaceName string          `desc:"full name of font face as loaded -- computed based on Family, Style, Weight, etc"`
	// todo: kerning
	// todo: stretch -- css 3 -- not supported
}

func (fs *FontStyle) Defaults() {
	fs.Color.SetColor(color.Black)
	fs.Opacity = 1.0
	fs.Size = units.NewValue(12, units.Pt)
}

// SetStylePost does any updates after generic xml-tag property setting -- use
// for anything that also has non-standard values that might not be processed
// properly by default
func (fs *FontStyle) SetStylePost(props ki.Props) {
	if pfs, ok := props["font-size"]; ok {
		if fsz, ok := pfs.(string); ok {
			if psz, ok := FontSizePoints[fsz]; ok {
				fs.Size = units.NewValue(psz, units.Pt)
			}
		}
	}
	if tds, ok := props["text-decoration"]; ok {
		if td, ok := tds.(string); ok {
			if td == "none" {
				fs.Deco = DecoNone // otherwise get a bit flag set
			}
		}
	}
}

// InheritFields from parent: Manual inheriting of values is much faster than
// automatic version!
func (fs *FontStyle) InheritFields(par *FontStyle) {
	fs.Color = par.Color
	fs.Family = par.Family
	fs.Style = par.Style
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

// FaceNm returns the full FaceName to use for the current FontStyle spec, robustly
func (fs *FontStyle) FaceNm() string {
	fnm := FontFaceName(fs.Family, fs.Stretch, fs.Weight, fs.Style)
	return fnm
}

// FontFaceName returns the best full FaceName to use for the given font
// family(ies) (comma separated) and modifier parameters
func FontFaceName(fam string, str FontStretch, wt FontWeights, sty FontStyles) string {
	if fam == "" {
		fam = string(Prefs.FontFamily)
	}
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
	return fnm
}

// LoadFont loads the font specified by the font style from the font library.
// This is the primary method to use for loading fonts, as it uses a robust
// fallback method to finding an appropriate font, and falls back on the
// builtin Go font as a last resort.  The Face field will have the resulting
// font.  The font size is always rounded to nearest integer, to produce
// better-looking results (presumably).  The current metrics and given
// unit.Context are updated based on the properties of the font.
func (fs *FontStyle) LoadFont(ctxt *units.Context) {
	fs.FaceName = fs.FaceNm()
	intDots := int(math.Round(float64(fs.Size.Dots)))
	if intDots == 0 {
		intDots = 12
	}
	face, err := FontLibrary.Font(fs.FaceName, intDots)
	if err != nil {
		log.Printf("%v\n", err)
		if fs.Face == nil {
			face, err = FontLibrary.Font("Go", intDots) // guaranteed to exist
			fs.Face = face
		}
	} else {
		fs.Face = face
	}
	fs.ComputeMetrics(ctxt)
	fs.SetUnitContext(ctxt)
}

// ComputeMetrics computes the Height, Em, Ex, Ch and Rem metrics associated
// with current font and overall units context
func (fs *FontStyle) ComputeMetrics(ctxt *units.Context) {
	if fs.Face == nil {
		return
	}
	intDots := float32(math.Round(float64(fs.Size.Dots)))
	if intDots == 0 {
		intDots = 12
	}
	// apd := fs.Face.Metrics().Ascent + fs.Face.Metrics().Descent
	fs.Height = math32.Ceil(FixedToFloat32(fs.Face.Metrics().Height))
	fs.Em = intDots // conventional definition
	xb, _, ok := fs.Face.GlyphBounds('x')
	if ok {
		fs.Ex = FixedToFloat32(xb.Max.Y - xb.Min.Y)
	} else {
		fs.Ex = 0.5 * fs.Em
	}
	xb, _, ok = fs.Face.GlyphBounds('0')
	if ok {
		fs.Ch = FixedToFloat32(xb.Max.X - xb.Min.X)
	} else {
		fs.Ch = 0.5 * fs.Em
	}
	fs.Rem = ctxt.ToDots(12, units.Pt)
	// fmt.Printf("fs: %v sz: %v\t\tHt: %v\tEm: %v\tEx: %v\tCh: %v\n", fs.FaceName, intDots, fs.Height, fs.Em, fs.Ex, fs.Ch)
}

// SetUnitContext sets the font-specific information in the given
// units.Context, based on the currently-loaded face.
func (fs *FontStyle) SetUnitContext(ctxt *units.Context) {
	if fs.Face != nil {
		ctxt.SetFont(fs.Em, fs.Ex, fs.Ch, fs.Rem)
	}
}

// Style CSS looks for "tag" name props in cssAgg props, and applies those to
// style if found, and returns true -- false if no such tag found
func (fs *FontStyle) StyleCSS(tag string, cssAgg ki.Props, ctxt *units.Context) bool {
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
	fs.SetStyleProps(nil, pmap)
	fs.LoadFont(ctxt)
	return true
}

// SetStyleProps sets font style values based on given property map (name:
// value pairs), inheriting elements as appropriate from parent, and also
// having a default style for the "initial" setting
func (fs *FontStyle) SetStyleProps(parent *FontStyle, props ki.Props) {
	// direct font styling is used only for special cases -- don't do this:
	// if !fs.StyleSet && parent != nil { // first time
	// 	FontStyleFields.Inherit(fs, parent)
	// }
	FontStyleFields.Style(fs, parent, props)
	fs.SetStylePost(props)
}

// ToDots calls ToDots on all units.Value fields in the style (recursively)
func (fs *FontStyle) ToDots(ctxt *units.Context) {
	FontStyleFields.ToDots(fs, ctxt)
}

// FontStyleDefault is default style can be used when property specifies "default"
var FontStyleDefault FontStyle

// FontStyleFields contain the StyledFields for FontStyle type
var FontStyleFields = initFontStyle()

func initFontStyle() *StyledFields {
	FontStyleDefault.Defaults()
	sf := &StyledFields{}
	sf.Init(&FontStyleDefault)
	return sf
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

//////////////////////////////////////////////////////////////////////////////////
// Font library

// FontInfo contains basic font information for choosing a given font --
// displayed in the font chooser dialog.
type FontInfo struct {
	Name    string      `desc:"official regularized name of font"`
	Stretch FontStretch `xml:"stretch" desc:"stretch: normal, expanded, condensed, etc"`
	Weight  FontWeights `xml:"weight" desc:"weight: normal, bold, etc"`
	Style   FontStyles  `xml:"style" desc:"style -- normal, italic, etc"`
	Example string      `desc:"example text -- styled according to font params in chooser"`
}

// FontLib holds the fonts available in a font library.  The font name is
// regularized so that the base "Regular" font is the root term of a sequence
// of other font names that describe the stretch, weight, and style, e.g.,
// "Arial" as the base name, "Arial Bold", "Arial Bold Italic" etc.  Thus,
// each font name specifies a particular font weight and style.  When fonts
// are loaded into the library, the names are appropriately regularized.
type FontLib struct {
	FontPaths  []string                     `desc:"list of font paths to search for fonts"`
	FontsAvail map[string]string            `desc:"map of font name to path to file"`
	FontInfo   []FontInfo                   `desc:"information about each font -- this list should be used for selecting valid regularized font names"`
	Faces      map[string]map[int]font.Face `desc:"double-map of cached fonts, by font name and then integer font size within that"`
	initMu     sync.Mutex
	loadMu     sync.Mutex
}

// FontLibrary is the gi font library, initialized from fonts available on font paths
var FontLibrary FontLib

// FontAvail determines if a given font name is available (case insensitive)
func (fl *FontLib) FontAvail(fontnm string) bool {
	fontnm = strings.ToLower(fontnm)
	_, ok := FontLibrary.FontsAvail[fontnm]
	return ok
}

// FontInfoExample is example text to demonstrate fonts -- from Inkscape plus extra
var FontInfoExample = "AaBbCcIiPpQq12369$€¢?.:/()àáâãäåæç日本中国⇧⌘"

func (fl *FontLib) Init() {
	fl.initMu.Lock()
	if fl.FontPaths == nil {
		// fmt.Printf("Initializing font lib\n")
		fl.FontPaths = make([]string, 0, 1000)
		fl.FontsAvail = make(map[string]string)
		fl.FontInfo = make([]FontInfo, 0, 1000)
		fl.Faces = make(map[string]map[int]font.Face)
	} else if len(fl.FontsAvail) == 0 {
		fmt.Printf("updating fonts avail in %v\n", fl.FontPaths)
		fl.UpdateFontsAvail()
	}
	fl.initMu.Unlock()
}

// Font gets a particular font, specified by the official regularized font
// name (see FontsAvail list), at given dots size (integer), using a cache of
// loaded fonts.
func (fl *FontLib) Font(fontnm string, size int) (font.Face, error) {
	fontnm = strings.ToLower(fontnm)
	fl.Init()
	if facemap := fl.Faces[fontnm]; facemap != nil {
		if face := facemap[size]; face != nil {
			// fmt.Printf("Got font face from cache: %v %v\n", fontnm, size)
			return face, nil
		}
	}
	if path := fl.FontsAvail[fontnm]; path != "" {
		face, err := LoadFontFace(path, size, 0)
		if err != nil {
			log.Printf("gi.FontLib: error loading font %v, removed from list\n", fontnm)
			fl.DeleteFont(fontnm)
			return nil, err
		}
		fl.loadMu.Lock()
		facemap := fl.Faces[fontnm]
		if facemap == nil {
			facemap = make(map[int]font.Face)
			fl.Faces[fontnm] = facemap
		}
		facemap[size] = face
		// fmt.Printf("Loaded font face: %v %v\n", fontnm, size)
		fl.loadMu.Unlock()
		return face, nil
	}
	return nil, fmt.Errorf("gi.FontLib: Font named: %v not found in list of available fonts, try adding to FontPaths in gi.FontLibrary, searched paths: %v\n", fontnm, fl.FontPaths)
}

func (fl *FontLib) DeleteFont(fontnm string) {
	delete(fl.FontsAvail, fontnm)
	for i, fi := range fl.FontInfo {
		if strings.ToLower(fi.Name) == fontnm {
			sz := len(fl.FontInfo)
			copy(fl.FontInfo[i:], fl.FontInfo[i+1:])
			fl.FontInfo = fl.FontInfo[:sz-1]
			break
		}
	}
}

// LoadAllFonts attempts to load all fonts that were found -- call this before
// displaying the font chooser to eliminate any bad fonts.
func (fl *FontLib) LoadAllFonts(size int) {
	sz := len(fl.FontInfo)
	for i := sz - 1; i > 0; i-- {
		fi := fl.FontInfo[i]
		fl.Font(strings.ToLower(fi.Name), size)
	}
}

// InitFontPaths initializes font paths to system defaults, only if no paths
// have yet been set
func (fl *FontLib) InitFontPaths(paths ...string) {
	if len(fl.FontPaths) > 0 {
		return
	}
	fl.AddFontPaths(paths...)
}

func (fl *FontLib) AddFontPaths(paths ...string) bool {
	fl.Init()
	for _, p := range paths {
		fl.FontPaths = append(fl.FontPaths, p)
	}
	return fl.UpdateFontsAvail()
}

// UpdateFontsAvail scans for all fonts we can use on the FontPaths
func (fl *FontLib) UpdateFontsAvail() bool {
	if len(fl.FontPaths) == 0 {
		log.Print("gi.FontLib: no font paths -- need to add some\n")
		return false
	}
	if len(fl.FontsAvail) > 0 {
		fl.FontsAvail = make(map[string]string)
	}
	fl.GoFontsAvail()
	for _, p := range fl.FontPaths {
		fl.FontsAvailFromPath(p)
	}
	sort.Slice(fl.FontInfo, func(i, j int) bool {
		return fl.FontInfo[i].Name < fl.FontInfo[j].Name
	})

	return len(fl.FontsAvail) > 0
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

var FontExts = map[string]struct{}{
	".ttf": struct{}{},
	".ttc": struct{}{}, // note: unpack to raw .ttf to use -- otherwise only getting first font
	//	".otf": struct{}{},  // not yet supported
}

// FontsAvailFromPath scans for all fonts we can use on a given path,
// gathering info into FontsAvail and FontInfo.
func (fl *FontLib) FontsAvailFromPath(path string) error {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("gi.FontLib: error accessing path %q: %v\n", path, err)
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		_, ok := FontExts[ext]
		if !ok {
			return nil
		}
		_, fn := filepath.Split(path)
		fn = fn[:len(fn)-len(ext)]
		bfn := fn
		bfn = strings.TrimSuffix(fn, "bd")
		bfn = strings.TrimSuffix(bfn, "bi")
		bfn = strings.TrimSuffix(bfn, "z")
		bfn = strings.TrimSuffix(bfn, "b")
		if bfn != "calibri" && bfn != "gadugui" && bfn != "segoeui" && bfn != "segui" {
			bfn = strings.TrimSuffix(bfn, "i")
		}
		if afn, ok := altFontMap[bfn]; ok {
			sfx := ""
			if strings.HasSuffix(fn, "bd") || strings.HasSuffix(fn, "b") {
				sfx = " Bold"
			} else if strings.HasSuffix(fn, "bi") || strings.HasSuffix(fn, "z") {
				sfx = " Bold Italic"
			} else if strings.HasSuffix(fn, "i") {
				sfx = " Italic"
			}
			fn = afn + sfx
		} else {
			fn = strings.Replace(fn, "_", " ", -1)
			fn = strings.Replace(fn, "-", " ", -1)
			// fn = strings.Title(fn)
			for sc, rp := range shortFontMods {
				if strings.HasSuffix(fn, sc) {
					fn = strings.TrimSuffix(fn, sc)
					fn += rp
					break
				}
			}
		}
		fn = FixFontMods(fn)
		basefn := strings.ToLower(fn)
		if _, ok := fl.FontsAvail[basefn]; !ok {
			fl.FontsAvail[basefn] = path
			fi := FontInfo{Name: fn, Example: FontInfoExample}
			_, fi.Stretch, fi.Weight, fi.Style = FontNameToMods(fn)
			fl.FontInfo = append(fl.FontInfo, fi)
			// fmt.Printf("added font: %v at path %q\n", basefn, path)

		}
		return nil
	})
	if err != nil {
		log.Printf("gi.FontLib: error walking the path %q: %v\n", path, err)
	}
	return err
}

// altFontMap is an alternative font map that maps file names to more standard
// full names (e.g., Times -> Times New Roman) -- also looks for b,i suffixes
// for these cases -- some are added here just to pick up those suffixes.
// This is needed for Windows only.
var altFontMap = map[string]string{
	"arial":   "Arial",
	"ariblk":  "Arial Black",
	"candara": "Candara",
	"calibri": "Calibri",
	"cambria": "Cambria",
	"cour":    "Courier New",
	"constan": "Constantia",
	"consola": "Console",
	"comic":   "Comic Sans MS",
	"corbel":  "Corbel",
	"framd":   "Franklin Gothic Medium",
	"georgia": "Georgia",
	"gadugi":  "Gadugi",
	"malgun":  "Malgun Gothic",
	"mmrtex":  "Myanmar Text",
	"pala":    "Palatino",
	"segoepr": "Segoe Print",
	"segoesc": "Segoe Script",
	"segoeui": "Segoe UI",
	"segui":   "Segoe UI Historic",
	"tahoma":  "Tahoma",
	"taile":   "Traditional Arabic",
	"times":   "Times New Roman",
	"trebuc":  "Trebuchet",
	"verdana": "Verdana",
}

// shortFontMods corrects annoying short font mod names, found in Unity font
// on linux -- needs space and uppercase to avoid confusion -- checked with
// HasSuffix
var shortFontMods = map[string]string{
	" B":  " Bold",
	" I":  " Italic",
	" C":  " Condensed",
	" L":  " Light",
	" LI": " Light Italic",
	" M":  " Medium",
	" MI": " Medium Italic",
	" R":  " Regular",
	" RI": " Italic",
	" BI": " Bold Italic",
}

// LoadFontFace loads a font file at given path, with given raw size in
// display dots, and if strokeWidth is > 0, the font is drawn in outline form
// (stroked) instead of filled (supported in SVG).
func LoadFontFace(path string, size int, strokeWidth int) (font.Face, error) {
	if strings.HasPrefix(path, "gofont") {
		return LoadGoFont(path, size, strokeWidth)
	}
	fontBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".otf" {
		// note: this compiles but otf fonts are NOT yet supported apparently
		f, err := sfnt.Parse(fontBytes)
		if err != nil {
			return nil, err
		}
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: float64(size),
			// Hinting: font.HintingFull,
		})
		return face, err
	} else {
		f, err := truetype.Parse(fontBytes)
		if err != nil {
			return nil, err
		}
		face := truetype.NewFace(f, &truetype.Options{
			Size:   float64(size),
			Stroke: strokeWidth,
			// Hinting: font.HintingFull,
			// GlyphCacheEntries: 1024, // default is 512 -- todo benchmark
		})
		return face, nil
	}
}

// see: https://blog.golang.org/go-fonts

type goFontInfo struct {
	name string
	ttf  []byte
}

var GoFonts = map[string]goFontInfo{
	"gofont/goregular":         {"Go", goregular.TTF},
	"gofont/gobold":            {"Go Bold", gobold.TTF},
	"gofont/gobolditalic":      {"Go Bold Italic", gobolditalic.TTF},
	"gofont/goitalic":          {"Go Italic", goitalic.TTF},
	"gofont/gomedium":          {"Go Medium", gomedium.TTF},
	"gofont/gomediumitalic":    {"Go Medium Italic", gomediumitalic.TTF},
	"gofont/gomono":            {"Go Mono", gomono.TTF},
	"gofont/gomonobold":        {"Go Mono Bold", gomonobold.TTF},
	"gofont/gomonobolditalic":  {"Go Mono Bold Italic", gomonobolditalic.TTF},
	"gofont/gomonoitalic":      {"Go Mono Italic", gomonoitalic.TTF},
	"gofont/gosmallcaps":       {"Go Small Caps", gosmallcaps.TTF},
	"gofont/gosmallcapsitalic": {"Go Small Caps Italic", gosmallcapsitalic.TTF},
}

func LoadGoFont(path string, size int, strokeWidth int) (font.Face, error) {
	gf, ok := GoFonts[path]
	if !ok {
		return nil, fmt.Errorf("Go Font Path not found: %v", path)
	}
	f, _ := truetype.Parse(gf.ttf)
	face := truetype.NewFace(f, &truetype.Options{
		Size:   float64(size),
		Stroke: strokeWidth,
		// Hinting: font.HintingFull,
		// GlyphCacheEntries: 1024, // default is 512 -- todo benchmark

	})
	return face, nil
}

func (fl *FontLib) GoFontsAvail() {
	for path, gf := range GoFonts {
		basefn := strings.ToLower(gf.name)
		fl.FontsAvail[basefn] = path
		fi := FontInfo{Name: gf.name, Example: FontInfoExample}
		_, fi.Stretch, fi.Weight, fi.Style = FontNameToMods(gf.name)
		fl.FontInfo = append(fl.FontInfo, fi)
	}
}

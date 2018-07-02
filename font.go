// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/goki/gi/units"
	"github.com/goki/ki/kit"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// font.go contains all font and basic SVG-level text rendering styles, and the
// font library.  see text.go for rendering code

// FontStyle contains all font styling information, including everything that
// is used in SVG text rendering -- used in Paint and in Style -- see style.go
// -- most of font information is inherited
type FontStyle struct {
	Size             units.Value     `xml:"font-size" desc:"size of font to render -- convert to points when getting font to use"`
	Family           string          `xml:"font-family" inherit:"true" desc:"font family -- ordered list of names from more general to more specific to use -- use split on , to parse"`
	Style            FontStyles      `xml:"font-style" inherit:"true" desc:"style -- normal, italic, etc"`
	Weight           FontWeights     `xml:"font-weight" inherit:"true" desc:"weight: normal, bold, etc"`
	Stretch          FontStretch     `xml:"font-stretch" inherit:"true" desc:"font stretch / condense options"`
	Variant          FontVariants    `xml:"font-variant" inherit:"true" desc:"normal or small caps"`
	Decoration       TextDecorations `xml:"text-decoration" desc:"underline, line-through, etc -- not inherited"`
	Shift            BaselineShifts  `xml:"baseline-shift" desc:"super / sub script -- not inherited"`
	LetterSpacing    units.Value     `xml:"letter-spacing" desc:"spacing between characters and lines"`
	WordSpacing      units.Value     `xml:"word-spacing" inherit:"true" desc:"extra space to add between words"`
	Anchor           TextAnchors     `xml:"text-anchor" inherit:"true" desc:"for svg rendering only: determines the alignment relative to text position coordinate: for RTL start is right, not left, and start is top for TB"`
	UnicodeBidi      UnicodeBidi     `xml:"unicode-bidi" inherit:"true" desc:"determines how to treat unicode bidirectional information"`
	Direction        TextDirections  `xml:"direction" inherit:"true" desc:"direction of text -- only applicable for unicode-bidi = bidi-override or embed -- applies to all text elements"`
	WritingMode      TextDirections  `xml:"writing-mode" inherit:"true" desc:"overall writing mode -- only for text elements, not tspan"`
	OrientationVert  float32         `xml:"glyph-orientation-vertical" inherit:"true" desc:"for TBRL writing mode (only), determines orientation of alphabetic characters -- 90 is default (rotated) -- 0 means keep upright"`
	OrientationHoriz float32         `xml:"glyph-orientation-horizontal" inherit:"true" desc:"for horizontal LR/RL writing mode (only), determines orientation of all characters -- 0 is default (upright)"`
	Face             font.Face       `desc:"actual font codes for drawing text -- just a pointer into FontLibrary of loaded fonts"`
	Height           float32         `desc:"reference 1.0 spacing line height of font in dots -- computed from font"`
	FaceName         string          `desc:"name corresponding to Face"`
	// todo: size also includes things like: medium, xx-small...xx-large, smaller, larger, etc
	// todo: kerning
	// todo: stretch -- css 3 -- not supported
}

func (fs *FontStyle) Defaults() {
	fs.FaceName = "Arial"
	fs.Size = units.NewValue(12, units.Pt)
	fs.Direction = LTR
	fs.OrientationVert = 90
}

// any updates after generic xml-tag property setting?
func (fs *FontStyle) SetStylePost() {
}

// FaceNm returns the full FaceName to use for the current FontStyle spec
func (fs *FontStyle) FaceNm() string {
	if fs.Family == "" {
		return "Arial" // built-in default
	}
	fnm := "Arial"
	nms := strings.Split(fs.Family, ",")
	for _, fn := range nms {
		fn = strings.TrimSpace(fn)
		if FontLibrary.FontAvail(fn) {
			fnm = fn
			break
		}
		switch fn {
		case "times":
			fnm = "Times New Roman"
			break
		case "serif":
			fnm = "Times New Roman"
			break
		case "sans-serif":
			fnm = "Arial"
			break
		case "courier":
			fnm = "Courier New" // this is the tt name
			break
		case "monospace":
			if FontLibrary.FontAvail("Andale Mono") {
				fnm = "Andale Mono"
			} else {
				fnm = "Courier New"
			}
			break
		case "cursive":
			if FontLibrary.FontAvail("Comic Sans") {
				fnm = "Comic Sans"
			} else if FontLibrary.FontAvail("Comic Sans MS") {
				fnm = "Comic Sans MS"
			}
			break
		case "fantasy":
			if FontLibrary.FontAvail("Impact") {
				fnm = "Impact"
			} else if FontLibrary.FontAvail("Impac") {
				fnm = "Impac"
			}
			break
		}
	}
	mods := ""
	if fs.Style == FontItalic && fs.Weight == WeightBold {
		mods = "Bold Italic"
	} else if fs.Style == FontItalic {
		mods = "Italic"
	} else if fs.Weight == WeightBold {
		mods = "Bold"
	}
	if mods != "" {
		if FontLibrary.FontAvail(fnm + " " + mods) {
			fnm += " " + mods
		}
		// todo: use similar style font fallback to get mods
	}
	return fnm
}

var lastDots = 0.0

func (fs *FontStyle) LoadFont(ctxt *units.Context, fallback string) {
	fs.FaceName = fs.FaceNm()
	intDots := math.Round(float64(fs.Size.Dots))
	face, err := FontLibrary.Font(fs.FaceName, intDots)
	if err != nil {
		// log.Printf("%v\n", err)
		if fs.Face == nil {
			if fallback != "" {
				fs.FaceName = fallback
				fs.LoadFont(ctxt, "") // try again
			} else {
				//				log.Printf("FontStyle LoadFont() -- Falling back on basicfont\n")
				fs.Face = basicfont.Face7x13
			}
		}
	} else {
		fs.Face = face
	}
	fs.Height = FixedToFloat32(fs.Face.Metrics().Height)
	// if lastDots != fs.Size.Dots {
	// 	pts := fs.Size.Convert(units.Pt, ctxt)
	// 	fmt.Printf("LoadFont points: %v intDots: %v, origDots: %v, height %v, ctxt dpi: %v\n", pts.Val, intDots, pts.Dots, fs.Height, ctxt.DPI)
	// 	lastDots = fs.Size.Dots
	// }
	fs.SetUnitContext(ctxt)
	// em := float32(fs.Face.Metrics().Ascent+fs.Face.Metrics().Descent) / 64.0
	// fmt.Printf("requested font size: %v got height: %v, em: %v\n", pts.Val, fs.Height, em)
}

func (fs *FontStyle) SetUnitContext(ctxt *units.Context) {
	// todo: could measure actual chars but just use defaults right now
	if fs.Face != nil {
		em := FixedToFloat32(fs.Face.Metrics().Ascent + fs.Face.Metrics().Descent)
		ctxt.SetFont(em, 0.5*em, .9*em, 12.0) // todo: rem!?  just using 12
		// fmt.Printf("em %v ex %v ch %v\n", em, 0.5*em, 0.9*em)
		// order is ex, ch, rem -- using .75 for ch
	}
}

//////////////////////////////////////////////////////////////////////////////////
// Font Style enums

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

// FontWeights styles of font: normal, italic, etc
type FontWeights int32

const (
	WeightNormal FontWeights = iota
	WeightBold
	WeightBolder
	WeightLighter
	Weight100
	Weight200
	Weight300
	Weight400 // normal
	Weight500
	Weight600
	Weight700
	Weight800
	Weight900 // bold
	FontWeightsN
)

//go:generate stringer -type=FontWeights

var KiT_FontWeights = kit.Enums.AddEnumAltLower(FontWeightsN, false, StylePropProps, "Weight")

func (ev FontWeights) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FontWeights) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

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

// FontStretch are different stretch levels of font.  todo: not currently supported
type FontStretch int32

const (
	FontStrNormal FontStretch = iota
	FontStrWider
	FontStrNarrower
	FontStrUltraCondensed
	FontStrExtraCondensed
	FontStrCondensed
	FontStrSemiCondensed
	FontStrSemiExpanded
	FontStrExpanded
	FontStrExtraExpanded
	FontStrUltraExpanded
	FontStretchN
)

//go:generate stringer -type=FontStretch

var KiT_FontStretch = kit.Enums.AddEnumAltLower(FontStretchN, false, StylePropProps, "FontStr")

func (ev FontStretch) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FontStretch) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// TextDecorations are underline, line-through, etc -- operates as bit flags
// -- also used for additional layout hints for RuneRender
type TextDecorations int32

const (
	DecoNone TextDecorations = iota
	DecoUnderline
	DecoOverline
	DecoLineThrough
	DecoBlink

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
	DecoSuper
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

// https://godoc.org/golang.org/x/text/unicode/bidi
// UnicodeBidi determines how
type UnicodeBidi int32

const (
	BidiNormal UnicodeBidi = iota
	BidiEmbed
	BidiBidiOverride
	UnicodeBidiN
)

//go:generate stringer -type=UnicodeBidi

var KiT_UnicodeBidi = kit.Enums.AddEnumAltLower(UnicodeBidiN, false, StylePropProps, "Bidi")

func (ev UnicodeBidi) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *UnicodeBidi) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// TextDirections are for direction of text writing, used in direction and writing-mode styles
type TextDirections int32

const (
	LRTB TextDirections = iota
	RLTB
	TBRL
	LR
	RL
	TB
	LTR
	RTL
	TextDirectionsN
)

//go:generate stringer -type=TextDirections

var KiT_TextDirections = kit.Enums.AddEnumAltLower(TextDirectionsN, false, StylePropProps, "")

func (ev TextDirections) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *TextDirections) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// TextAnchors are for direction of text writing, used in direction and writing-mode styles
type TextAnchors int32

const (
	AnchorStart TextAnchors = iota
	AnchorMiddle
	AnchorEnd
	TextAnchorsN
)

//go:generate stringer -type=TextAnchors

var KiT_TextAnchors = kit.Enums.AddEnumAltLower(TextAnchorsN, false, StylePropProps, "Anchor")

func (ev TextAnchors) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *TextAnchors) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

//////////////////////////////////////////////////////////////////////////////////
// Font library

func LoadFontFace(path string, points float64) (font.Face, error) {
	fontBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size: points,
		// Hinting: font.HintingFull,
	})
	return face, nil
}

type FontInfo struct {
	Name    string      `desc:"name of font"`
	Style   FontStyles  `xml:"style" inherit:"true" desc:"style -- normal, italic, etc"`
	Weight  FontWeights `xml:"weight" inherit:"true" desc:"weight: normal, bold, etc"`
	Example string      `desc:"example text -- styled according to font params in chooser"`
}

type FontLib struct {
	FontPaths  []string
	FontsAvail map[string]string `desc:"map of font name to path to file"`
	FontInfo   []FontInfo        `desc:"information about each font"`
	Faces      map[string]map[float64]font.Face
	initMu     sync.Mutex
	loadMu     sync.Mutex
}

// FontLibrary is the gi font library, initialized from fonts available on font paths
var FontLibrary FontLib

// AltFontMap is an alternative font map that maps file names to more standard
// full names (e.g., Times -> Times New Roman) -- also looks for b,i suffixes
// for these cases -- some are added here just to pick up those suffixes
var AltFontMap = map[string]string{
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

func (fl *FontLib) Init() {
	fl.initMu.Lock()
	if fl.FontPaths == nil {
		// fmt.Printf("Initializing font lib\n")
		fl.FontPaths = make([]string, 0, 1000)
		fl.FontsAvail = make(map[string]string)
		fl.FontInfo = make([]FontInfo, 0, 1000)
		fl.Faces = make(map[string]map[float64]font.Face)
	} else if len(fl.FontsAvail) == 0 {
		fmt.Printf("updating fonts avail in %v\n", fl.FontPaths)
		fl.UpdateFontsAvail()
	}
	fl.initMu.Unlock()
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
	for _, p := range fl.FontPaths {
		fl.FontsAvailFromPath(p)
	}
	sort.Slice(fl.FontInfo, func(i, j int) bool {
		return fl.FontInfo[i].Name < fl.FontInfo[j].Name
	})

	return len(fl.FontsAvail) > 0
}

// FontsAvailFromPath scans for all fonts we can use on a given path,
// gathering info into FontsAvail and FontInfo
func (fl *FontLib) FontsAvailFromPath(path string) error {
	ext := ".ttf" // for now -- might need more

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("gi.FontLib: error accessing path %q: %v\n", path, err)
			return err
		}
		if filepath.Ext(path) == ext {
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
			if afn, ok := AltFontMap[bfn]; ok {
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
			}
			basefn := strings.ToLower(fn)
			if _, ok := fl.FontsAvail[basefn]; !ok {
				fl.FontsAvail[basefn] = path
				fi := FontInfo{Name: fn, Style: FontNormal, Weight: WeightNormal, Example: FontInfoExample}
				if strings.Contains(basefn, "bold") {
					fi.Weight = WeightBold
				}
				if strings.Contains(basefn, "italic") {
					fi.Style = FontItalic
				} else if strings.Contains(basefn, "oblique") {
					fi.Style = FontOblique
				}
				fl.FontInfo = append(fl.FontInfo, fi)
				// fmt.Printf("added font: %v at path %q\n", basefn, path)

			}
		}
		return nil
	})
	if err != nil {
		log.Printf("gi.FontLib: error walking the path %q: %v\n", path, err)
	}
	return err
}

// Font gets a particular font
func (fl *FontLib) Font(fontnm string, points float64) (font.Face, error) {
	fontnm = strings.ToLower(fontnm)
	fl.Init()
	if facemap := fl.Faces[fontnm]; facemap != nil {
		if face := facemap[points]; face != nil {
			// fmt.Printf("Got font face from cache: %v %v\n", fontnm, points)
			return face, nil
		}
	}
	if path := fl.FontsAvail[fontnm]; path != "" {
		face, err := LoadFontFace(path, points)
		if err != nil {
			log.Printf("gi.FontLib: error loading font %v\n", err)
			return nil, err
		}
		fl.loadMu.Lock()
		facemap := fl.Faces[fontnm]
		if facemap == nil {
			facemap = make(map[float64]font.Face)
			fl.Faces[fontnm] = facemap
		}
		facemap[points] = face
		// fmt.Printf("Loaded font face: %v %v\n", fontnm, points)
		fl.loadMu.Unlock()
		return face, nil
	}
	return nil, fmt.Errorf("gi.FontLib: Font named: %v not found in list of available fonts, try adding to FontPaths in gi.FontLibrary, searched paths: %v\n", fontnm, fl.FontPaths)
}

// FontAvail determines if a given font name is available (case insensitive)
func (fl *FontLib) FontAvail(fontnm string) bool {
	fontnm = strings.ToLower(fontnm)
	_, ok := FontLibrary.FontsAvail[fontnm]
	return ok
}

// FontInfoExample is example text to demonstrate fonts -- from Inkscape plus extra
var FontInfoExample = "AaBbCcIiPpQq12369$€¢?.:/()àáâãäåæç日本中国"

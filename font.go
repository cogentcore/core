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
	"strings"
	"sync"

	"github.com/goki/gi/units"
	"github.com/goki/ki/kit"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// styles of font: normal, italic, etc
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

// styles of font: normal, italic, etc
type FontWeights int32

const (
	WeightNormal FontWeights = iota
	WeightBold
	WeightBolder
	WeightLighter
	//	Weight100...900  todo: seriously?  400 = normal, 700 = bold
	FontWeightsN
)

//go:generate stringer -type=FontWeights

var KiT_FontWeights = kit.Enums.AddEnumAltLower(FontWeightsN, false, StylePropProps, "Weight")

func (ev FontWeights) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *FontWeights) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// todo: Variant = normal / small-caps

// note: most of font information is inherited

// font style information -- used in Paint and in Style -- see style.go
type FontStyle struct {
	Face     font.Face   `desc:"actual font codes for drawing text -- just a pointer into FontLibrary of loaded fonts"`
	Height   float32     `desc:"recommended line hieight of font in dots"`
	FaceName string      `desc:"name corresponding to Face"`
	Size     units.Value `xml:"size" desc:"size of font to render -- convert to points when getting font to use"`
	Family   string      `xml:"family" inherit:"true" desc:"font family -- ordered list of names from more general to more specific to use -- use split on , to parse"`
	Style    FontStyles  `xml:"style" inherit:"true" desc:"style -- normal, italic, etc"`
	Weight   FontWeights `xml:"weight" inherit:"true" desc:"weight: normal, bold, etc"`
	// todo: size also includes things like: medium, xx-small...xx-large, smaller, larger, etc
	// todo: kerning
	// todo: stretch -- css 3 -- not supported
}

func (p *FontStyle) Defaults() {
	p.FaceName = "Arial"
	p.Size = units.NewValue(12, units.Pt)
}

// any updates after generic xml-tag property setting?
func (p *FontStyle) SetStylePost() {
}

// FaceNm returns the full FaceName to use for the current FontStyle spec
func (p *FontStyle) FaceNm() string {
	if p.Family == "" {
		return "Arial" // built-in default
	}
	fnm := "Arial"
	nms := strings.Split(p.Family, ",")
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
	if p.Style == FontItalic && p.Weight == WeightBold {
		mods = "Bold Italic"
	} else if p.Style == FontItalic {
		mods = "Italic"
	} else if p.Weight == WeightBold {
		mods = "Bold"
	}
	if mods != "" {
		if FontLibrary.FontAvail(fnm + " " + mods) {
			fnm += " " + mods
		}
	}
	return fnm
}

var lastDots = 0.0

func (p *FontStyle) LoadFont(ctxt *units.Context, fallback string) {
	p.FaceName = p.FaceNm()
	intDots := math.Round(float64(p.Size.Dots))
	face, err := FontLibrary.Font(p.FaceName, intDots)
	if err != nil {
		// log.Printf("%v\n", err)
		if p.Face == nil {
			if fallback != "" {
				p.FaceName = fallback
				p.LoadFont(ctxt, "") // try again
			} else {
				//				log.Printf("FontStyle LoadFont() -- Falling back on basicfont\n")
				p.Face = basicfont.Face7x13
			}
		}
	} else {
		p.Face = face
	}
	p.Height = float32(p.Face.Metrics().Height) / 64.0
	// if lastDots != p.Size.Dots {
	// 	pts := p.Size.Convert(units.Pt, ctxt)
	// 	fmt.Printf("LoadFont points: %v intDots: %v, origDots: %v, height %v, ctxt dpi: %v\n", pts.Val, intDots, pts.Dots, p.Height, ctxt.DPI)
	// 	lastDots = p.Size.Dots
	// }
	p.SetUnitContext(ctxt)
	// em := float32(p.Face.Metrics().Ascent+p.Face.Metrics().Descent) / 64.0
	// fmt.Printf("requested font size: %v got height: %v, em: %v\n", pts.Val, p.Height, em)
}

func (p *FontStyle) SetUnitContext(ctxt *units.Context) {
	// todo: could measure actual chars but just use defaults right now
	if p.Face != nil {
		em := float32(p.Face.Metrics().Ascent+p.Face.Metrics().Descent) / 64.0
		ctxt.SetFont(em, 0.5*em, .9*em, 12.0) // todo: rem!?  just using 12
		// fmt.Printf("em %v ex %v ch %v\n", em, 0.5*em, 0.9*em)
		// order is ex, ch, rem -- using .75 for ch
	}
}

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

// we export this font library
var FontLibrary FontLib

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

func (fl *FontLib) UpdateFontsAvail() bool {
	if len(fl.FontPaths) == 0 {
		log.Print("FontLib: no font paths -- need to add some\n")
		return false
	}
	if len(fl.FontsAvail) > 0 {
		fl.FontsAvail = make(map[string]string)
	}

	ext := ".ttf" // for now -- might need more

	for _, p := range fl.FontPaths {
		err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("FontLib: error accessing path %q: %v\n", p, err)
				return err
			}
			if filepath.Ext(path) == ext {
				_, fn := filepath.Split(path)
				fn = strings.TrimRight(fn, ext)
				fn = strings.Replace(fn, "_", " ", -1)
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
			fmt.Printf("FontLib: error walking the path %q: %v\n", p, err)
		}
	}
	return len(fl.FontsAvail) > 0
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
			log.Printf("FontLib: error loading font %v\n", err)
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
	return nil, fmt.Errorf("FontLib: Font named: %v not found in list of available fonts, try adding to FontPaths in gi.FontLibrary, searched paths: %v\n", fontnm, fl.FontPaths)
}

// FontAvail determines if a given font name is available (case insensitive)
func (fl *FontLib) FontAvail(fontnm string) bool {
	fontnm = strings.ToLower(fontnm)
	_, ok := FontLibrary.FontsAvail[fontnm]
	return ok
}

// FontInfoExample is example text to demonstrate fonts -- from Inkscape
var FontInfoExample = "AaBbCcIiPpQq12369$€¢?.:/()"

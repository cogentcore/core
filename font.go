// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
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

	"github.com/golang/freetype/truetype"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki/kit"
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

// todo: Variant = normal / small-caps

// note: most of font information is inherited

// font style information -- used in Paint and in Style -- see style.go
type FontStyle struct {
	Face     font.Face   `desc:"actual font codes for drawing text -- just a pointer into FontLibrary of loaded fonts"`
	Height   float64     `desc:"recommended line hieight of font in dots"`
	FaceName string      `desc:"name corresponding to Face"`
	Size     units.Value `xml:"size" desc:"size of font to render -- convert to points when getting font to use"`
	Family   string      `xml:"family" inherit:"true" desc:"font family -- ordered list of names from more general to more specific to use -- use split on , to parse"`
	Style    FontStyles  `xml:"style" inherit:"true","desc:"style -- normal, italic, etc"`
	Weight   FontWeights `xml:"weight" inherit:"true","desc:"weight: normal, bold, etc"`
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

var lastDots = 0.0

func (p *FontStyle) LoadFont(ctxt *units.Context, fallback string) {
	intDots := math.Round(p.Size.Dots)
	face, err := FontLibrary.Font(p.FaceName, intDots)
	if err != nil {
		log.Printf("%v\n", err)
		if p.Face == nil {
			if fallback != "" {
				p.FaceName = fallback
				p.LoadFont(ctxt, "") // try again
			} else {
				log.Printf("FontStyle LoadFont() -- Falling back on basicfont\n")
				p.Face = basicfont.Face7x13
			}
		}
	} else {
		p.Face = face
	}
	p.Height = float64(p.Face.Metrics().Height) / 64.0
	// if lastDots != p.Size.Dots {
	// 	pts := p.Size.Convert(units.Pt, ctxt)
	// 	fmt.Printf("LoadFont points: %v intDots: %v, origDots: %v, height %v, ctxt dpi: %v\n", pts.Val, intDots, pts.Dots, p.Height, ctxt.DPI)
	// 	lastDots = p.Size.Dots
	// }
	p.SetUnitContext(ctxt)
	// em := float64(p.Face.Metrics().Ascent+p.Face.Metrics().Descent) / 64.0
	// fmt.Printf("requested font size: %v got height: %v, em: %v\n", pts.Val, p.Height, em)
}

func (p *FontStyle) SetUnitContext(ctxt *units.Context) {
	// todo: could measure actual chars but just use defaults right now
	if p.Face != nil {
		em := float64(p.Face.Metrics().Ascent+p.Face.Metrics().Descent) / 64.0
		ctxt.SetFont(em, 0.5*em, .9*em, 12.0) // todo: rem!?  just using 12
		// fmt.Printf("em %v ex %v ch %v\n", em, 0.5*em, 0.9*em)
		// order is ex, ch, rem -- using .75 for ch
	}
}

// update the font settings from the style info on the node
// func (pf *FontStyle) SetFromNode(g *Node2DBase) {
// 	// always check if property has been set before setting -- otherwise defaults to empty -- true = inherit props

// 	loadFont := false

// 	prevFaceName := pf.FaceName

// 	if sz, got := g.PropNumber("font-size"); got {
// 		if pf.Points != sz {
// 			loadFont = true
// 		}
// 		pf.Points = sz
// 	}
// 	if nm, got := g.PropEnum("font-face"); got {
// 		if len(nm) != 0 {
// 			if pf.FaceName != nm {
// 				pf.FaceName = nm
// 				loadFont = true
// 			}
// 		}
// 	}
// 	if nm, got := g.PropEnum("font-family"); got {
// 		if len(nm) != 0 {
// 			if pf.FaceName != nm {
// 				pf.FaceName = nm
// 				loadFont = true
// 			}
// 		}
// 	}
// 	if loadFont {
// 		pf.LoadFont(prevFaceName)
// 	}
// }

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

type FontLib struct {
	FontPaths  []string
	FontsAvail map[string]string `desc:"map of font name to path to file"`
	Faces      map[string]map[float64]font.Face
	initMu     sync.Mutex
	loadMu     sync.Mutex
}

// we export this font library
var FontLibrary FontLib

func (fl *FontLib) Init() {
	fl.initMu.Lock()
	if fl.FontPaths == nil {
		fl.FontPaths = make([]string, 0, 100)
		fl.FontsAvail = make(map[string]string)
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
				basefn := strings.TrimRight(fn, ext)
				fl.FontsAvail[basefn] = path
				// fmt.Printf("added font: %v at path %q\n", basefn, path)
			}
			return nil
		})

		if err != nil {
			fmt.Printf("FontLib: error walking the path %q: %v\n", p, err)
		}
	}
	return len(fl.FontsAvail) > 0
}

// get a particular font
func (fl *FontLib) Font(fontnm string, points float64) (font.Face, error) {
	fl.Init()
	if facemap := fl.Faces[fontnm]; facemap != nil {
		if face := facemap[points]; face != nil {
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
		}
		facemap[points] = face
		fl.loadMu.Unlock()
		return face, nil
	}
	return nil, fmt.Errorf("FontLib: Font named: %v not found in list of available fonts, try adding to FontPaths in gi.FontLibrary, searched paths: %v\n", fontnm, fl.FontPaths)
}

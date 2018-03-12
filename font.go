// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// font information for painter
type PaintFont struct {
	Face     font.Face
	Height   float64
	FaceName string
	Points   float64 // target points to use
}

func (p *PaintFont) Defaults() {
	p.FaceName = "Arial"
	p.Points = 24
}

func (p *PaintFont) LoadFont(fallback string) {
	face, err := FontLibrary.Font(p.FaceName, p.Points)
	if err != nil {
		log.Printf("%v\n", err)
		if p.Face == nil {
			if fallback != "" {
				p.FaceName = fallback
				p.LoadFont("") // try again
			} else {
				log.Printf("PaintFont LoadFont() -- Falling back on basicfont\n")
				p.Face = basicfont.Face7x13
			}
		}
	} else {
		p.Face = face
	}
	p.Height = float64(p.Face.Metrics().Height) / 64.0
}

// update the font settings from the style info on the node
func (pf *PaintFont) SetFromNode(g *Node2DBase) {
	// always check if property has been set before setting -- otherwise defaults to empty -- true = inherit props

	loadFont := false

	prevFaceName := pf.FaceName

	if sz, got := g.PropNumber("font-size"); got {
		if pf.Points != sz {
			loadFont = true
		}
		pf.Points = sz
	}
	if nm, got := g.PropEnum("font-face"); got {
		if len(nm) != 0 {
			if pf.FaceName != nm {
				pf.FaceName = nm
				loadFont = true
			}
		}
	}
	if loadFont {
		pf.LoadFont(prevFaceName)
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

type FontLib struct {
	FontPaths  []string
	FontsAvail map[string]string `desc:"map of font name to path to file"`
	Faces      map[string]map[float64]font.Face
}

// we export this font library
var FontLibrary FontLib

func (fl *FontLib) Init() {
	if fl.FontPaths == nil {
		fl.FontPaths = make([]string, 0, 100)
		fl.FontsAvail = make(map[string]string)
		fl.Faces = make(map[string]map[float64]font.Face)
	} else if len(fl.FontsAvail) == 0 {
		fl.UpdateFontsAvail()
	}
}

func (fl *FontLib) AddFontPaths(paths ...string) {
	fl.Init()
	for _, p := range paths {
		fl.FontPaths = append(fl.FontPaths, p)
	}
	fl.UpdateFontsAvail()
}

func (fl *FontLib) UpdateFontsAvail() {
	if len(fl.FontPaths) == 0 {
		log.Print("FontLib: no font paths -- need to add some\n")
		return
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
			// if info.IsDir() && info.Name() == subDirToSkip {
			// 	fmt.Printf("skipping a dir without errors: %+v \n", info.Name())
			// 	return filepath.SkipDir
			// }
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
		facemap := fl.Faces[fontnm]
		if facemap == nil {
			facemap = make(map[float64]font.Face)
		}
		facemap[points] = face
		return face, nil
	}
	return nil, fmt.Errorf("FontLib: Font named: %v not found in list of available fonts, try adding to FontPaths in gi.FontLibrary, searched paths: %v\n", fontnm, fl.FontPaths)
}

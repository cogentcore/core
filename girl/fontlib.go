// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/camelcase"
	"github.com/goki/freetype/truetype"
	"github.com/goki/gi/gist"
	"github.com/iancoleman/strcase"
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
)

// loadFontMu protects the font loading calls, which are not concurrent-safe
var loadFontMu sync.RWMutex

// FontInfo contains basic font information for choosing a given font --
// displayed in the font chooser dialog.
type FontInfo struct {
	Name    string           `desc:"official regularized name of font"`
	Stretch gist.FontStretch `xml:"stretch" desc:"stretch: normal, expanded, condensed, etc"`
	Weight  gist.FontWeights `xml:"weight" desc:"weight: normal, bold, etc"`
	Style   gist.FontStyles  `xml:"style" desc:"style -- normal, italic, etc"`
	Example string           `desc:"example text -- styled according to font params in chooser"`
}

// Label satisfies the Labeler interface
func (fi FontInfo) Label() string {
	return fi.Name
}

// FontLib holds the fonts available in a font library.  The font name is
// regularized so that the base "Regular" font is the root term of a sequence
// of other font names that describe the stretch, weight, and style, e.g.,
// "Arial" as the base name, "Arial Bold", "Arial Bold Italic" etc.  Thus,
// each font name specifies a particular font weight and style.  When fonts
// are loaded into the library, the names are appropriately regularized.
type FontLib struct {
	FontPaths  []string                          `desc:"list of font paths to search for fonts"`
	FontsAvail map[string]string                 `desc:"map of font name to path to file"`
	FontInfo   []FontInfo                        `desc:"information about each font -- this list should be used for selecting valid regularized font names"`
	Faces      map[string]map[int]*gist.FontFace `desc:"double-map of cached fonts, by font name and then integer font size within that"`
}

// FontLibrary is the gi font library, initialized from fonts available on font paths
var FontLibrary FontLib

// FontAvail determines if a given font name is available (case insensitive)
func (fl *FontLib) FontAvail(fontnm string) bool {
	loadFontMu.RLock()
	defer loadFontMu.RUnlock()

	fontnm = strings.ToLower(fontnm)
	_, ok := FontLibrary.FontsAvail[fontnm]
	return ok
}

// FontInfoExample is example text to demonstrate fonts -- from Inkscape plus extra
var FontInfoExample = "AaBbCcIiPpQq12369$€¢?.:/()àáâãäåæç日本中国⇧⌘"

// Init initializes the font library if it hasn't been yet
func (fl *FontLib) Init() {
	if fl.FontPaths == nil {
		loadFontMu.Lock()
		// fmt.Printf("Initializing font lib\n")
		fl.FontPaths = make([]string, 0, 1000)
		fl.FontsAvail = make(map[string]string)
		fl.FontInfo = make([]FontInfo, 0, 1000)
		fl.Faces = make(map[string]map[int]*gist.FontFace)
		loadFontMu.Unlock()
		return // no paths to load from yet
	}
	loadFontMu.RLock()
	sz := len(fl.FontsAvail)
	loadFontMu.RUnlock()
	if sz == 0 {
		// fmt.Printf("updating fonts avail in %v\n", fl.FontPaths)
		fl.UpdateFontsAvail()
	}
}

// Font gets a particular font, specified by the official regularized font
// name (see FontsAvail list), at given dots size (integer), using a cache of
// loaded fonts.
func (fl *FontLib) Font(fontnm string, size int) (*gist.FontFace, error) {
	fontnm = strings.ToLower(fontnm)
	fl.Init()
	loadFontMu.RLock()
	if facemap := fl.Faces[fontnm]; facemap != nil {
		if face := facemap[size]; face != nil {
			// fmt.Printf("Got font face from cache: %v %v\n", fontnm, size)
			loadFontMu.RUnlock()
			return face, nil
		}
	}
	if path := fl.FontsAvail[fontnm]; path != "" {
		loadFontMu.RUnlock()
		loadFontMu.Lock()
		face, err := OpenFontFace(fontnm, path, size, 0)
		if err != nil || face == nil {
			if err == nil {
				err = fmt.Errorf("gi.FontLib: nil face with no error for: %v", fontnm)
			}
			log.Printf("gi.FontLib: error loading font %v, removed from list\n", fontnm)
			loadFontMu.Unlock()
			fl.DeleteFont(fontnm)
			return nil, err
		}
		facemap := fl.Faces[fontnm]
		if facemap == nil {
			facemap = make(map[int]*gist.FontFace)
			fl.Faces[fontnm] = facemap
		}
		facemap[size] = face
		// fmt.Printf("Opened font face: %v %v\n", fontnm, size)
		loadFontMu.Unlock()
		return face, nil
	}
	loadFontMu.RUnlock()
	return nil, fmt.Errorf("gi.FontLib: Font named: %v not found in list of available fonts, try adding to FontPaths in gi.FontLibrary, searched paths: %v\n", fontnm, fl.FontPaths)
}

// DeleteFont removes given font from list of available fonts -- if not supported etc
func (fl *FontLib) DeleteFont(fontnm string) {
	loadFontMu.Lock()
	defer loadFontMu.Unlock()
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

// OpenAllFonts attempts to load all fonts that were found -- call this before
// displaying the font chooser to eliminate any bad fonts.
func (fl *FontLib) OpenAllFonts(size int) {
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
	loadFontMu.Lock()
	defer loadFontMu.Unlock()
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
		fn = gist.FixFontMods(fn)
		basefn := strings.ToLower(fn)
		if _, ok := fl.FontsAvail[basefn]; !ok {
			fl.FontsAvail[basefn] = path
			fi := FontInfo{Name: fn, Example: FontInfoExample}
			_, fi.Stretch, fi.Weight, fi.Style = gist.FontNameToMods(fn)
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

var FontExts = map[string]struct{}{
	".ttf": {},
	".ttc": {}, // note: unpack to raw .ttf to use -- otherwise only getting first font
	//	".otf": struct{}{},  // not yet supported
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

// see: https://blog.golang.org/go-fonts

type GoFontInfo struct {
	name string
	ttf  []byte
}

var GoFonts = map[string]GoFontInfo{
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

func OpenGoFont(name, path string, size int, strokeWidth int) (*gist.FontFace, error) {
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
	ff := gist.NewFontFace(name, size, face)
	return ff, nil
}

func (fl *FontLib) GoFontsAvail() {
	for path, gf := range GoFonts {
		basefn := strings.ToLower(gf.name)
		fl.FontsAvail[basefn] = path
		fi := FontInfo{Name: gf.name, Example: FontInfoExample}
		_, fi.Stretch, fi.Weight, fi.Style = gist.FontNameToMods(gf.name)
		fl.FontInfo = append(fl.FontInfo, fi)
	}
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

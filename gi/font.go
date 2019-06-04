// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/chewxy/math32"
	"github.com/goki/freetype/truetype"
	// "github.com/golang/freetype/truetype"

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

// font.go contains core font handling code interfacing with std font lib
// and providing a FontLibrary
// fontstyles.go has all the font styling parameters.
// text.go has rendering code for formatted text

// FontName is used to specify a font, as the unique name of the font family.
// This automatically provides a chooser menu for fonts using giv ValueView.
type FontName string

// FontFace is our enhanced Font Face structure which contains the enhanced computed
// metrics in addition to the font.Face face
type FontFace struct {
	Name    string      `desc:"The full FaceName that the font is accessed by"`
	Size    int         `desc:"The integer font size in raw dots"`
	Face    font.Face   `desc:"The system image.Font font rendering interface"`
	Metrics FontMetrics `desc:"enhanced metric information for the font"`
}

// NewFontFace returns a new font face
func NewFontFace(nm string, sz int, face font.Face) *FontFace {
	ff := &FontFace{Name: nm, Size: sz, Face: face}
	ff.ComputeMetrics()
	return ff
}

// FontMetrics are our enhanced dot-scale font metrics compared to what is available in
// the standard font.Metrics lib, including Ex and Ch being defined in terms of
// the actual letter x and 0
type FontMetrics struct {
	Height float32 `desc:"reference 1.0 spacing line height of font in dots -- computed from font as ascent + descent + lineGap, where lineGap is specified by the font as the recommended line spacing"`
	Em     float32 `desc:"Em size of font -- this is NOT actually the width of the letter M, but rather the specified point size of the font (in actual display dots, not points) -- it does NOT include the descender and will not fit the entire height of the font"`
	Ex     float32 `desc:"Ex size of font -- this is the actual height of the letter x in the font"`
	Ch     float32 `desc:"Ch size of font -- this is the actual width of the 0 glyph in the font"`
}

// ComputeMetrics computes the Height, Em, Ex, Ch and Rem metrics associated
// with current font and overall units context
func (fs *FontFace) ComputeMetrics() {
	// apd := fs.Face.Metrics().Ascent + fs.Face.Metrics().Descent
	fmet := fs.Face.Metrics()
	fs.Metrics.Height = math32.Ceil(FixedToFloat32(fmet.Height))
	fs.Metrics.Em = float32(fs.Size) // conventional definition
	xb, _, ok := fs.Face.GlyphBounds('x')
	if ok {
		fs.Metrics.Ex = FixedToFloat32(xb.Max.Y - xb.Min.Y)
		// note: metric.Ex is typically 0?
		// if fs.Metrics.Ex != metex {
		// 	fmt.Printf("computed Ex: %v  metric ex: %v\n", fs.Metrics.Ex, metex)
		// }
	} else {
		metex := FixedToFloat32(fmet.XHeight)
		if metex != 0 {
			fs.Metrics.Ex = metex
		} else {
			fs.Metrics.Ex = 0.5 * fs.Metrics.Em
		}
	}
	xb, _, ok = fs.Face.GlyphBounds('0')
	if ok {
		fs.Metrics.Ch = FixedToFloat32(xb.Max.X - xb.Min.X)
	} else {
		fs.Metrics.Ch = 0.5 * fs.Metrics.Em
	}
}

// loadFontMu protects the font loading calls, which are not concurrent-safe
var loadFontMu sync.RWMutex

// FontInfo contains basic font information for choosing a given font --
// displayed in the font chooser dialog.
type FontInfo struct {
	Name    string      `desc:"official regularized name of font"`
	Stretch FontStretch `xml:"stretch" desc:"stretch: normal, expanded, condensed, etc"`
	Weight  FontWeights `xml:"weight" desc:"weight: normal, bold, etc"`
	Style   FontStyles  `xml:"style" desc:"style -- normal, italic, etc"`
	Example string      `desc:"example text -- styled according to font params in chooser"`
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
	FontPaths  []string                     `desc:"list of font paths to search for fonts"`
	FontsAvail map[string]string            `desc:"map of font name to path to file"`
	FontInfo   []FontInfo                   `desc:"information about each font -- this list should be used for selecting valid regularized font names"`
	Faces      map[string]map[int]*FontFace `desc:"double-map of cached fonts, by font name and then integer font size within that"`
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
		fl.Faces = make(map[string]map[int]*FontFace)
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
func (fl *FontLib) Font(fontnm string, size int) (*FontFace, error) {
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
			facemap = make(map[int]*FontFace)
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

// OpenFontFace loads a font file at given path, with given raw size in
// display dots, and if strokeWidth is > 0, the font is drawn in outline form
// (stroked) instead of filled (supported in SVG).
// loadFontMu must be locked prior to calling
func OpenFontFace(name, path string, size int, strokeWidth int) (*FontFace, error) {
	if strings.HasPrefix(path, "gofont") {
		return OpenGoFont(name, path, size, strokeWidth)
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
		ff := NewFontFace(name, size, face)
		return ff, err
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
		ff := NewFontFace(name, size, face)
		return ff, nil
	}
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

func OpenGoFont(name, path string, size int, strokeWidth int) (*FontFace, error) {
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
	ff := NewFontFace(name, size, face)
	return ff, nil
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

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

	"github.com/fatih/camelcase"
	"github.com/goki/freetype/truetype"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/iancoleman/strcase"

	// "github.com/golang/freetype/truetype"

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

// loadFontMu protects the font loading calls, which are not concurrent-safe
var loadFontMu sync.RWMutex

// FaceName returns the full FaceName to use for the current FontStyle spec, robustly
func (fs *FontStyle) FaceName() string {
	fnm := FontFaceName(fs.Family, fs.Stretch, fs.Weight, fs.Style)
	return fnm
}

// Style CSS looks for "tag" name props in cssAgg props, and applies those to
// style if found, and returns true -- false if no such tag found
func (fs *FontStyle) StyleCSS(tag string, cssAgg ki.Props, unit *units.Context, ctxt Context) bool {
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
	fs.SetStyleProps(nil, pmap, ctxt)
	fs.OpenFont(unit)
	return true
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
		// fmt.Printf("FontStyle Error: bad font size: %v or units context: %v\n", fs.Size, *ctxt)
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
	".ttf": {},
	".ttc": {}, // note: unpack to raw .ttf to use -- otherwise only getting first font
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

// faceNameCache is a cache for fast lookup of valid font face names given style specs
var faceNameCache map[string]string

// faceNameCacheMu protects access to faceNameCache
var faceNameCacheMu sync.RWMutex

// FontFaceName returns the best full FaceName to use for the given font
// family(ies) (comma separated) and modifier parameters
func FontFaceName(fam string, str FontStretch, wt FontWeights, sty FontStyles) string {
	if fam == "" {
		fam = string(ThePrefs.PrefFontFamily())
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

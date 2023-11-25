// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/fatih/camelcase"
	"github.com/iancoleman/strcase"
	"goki.dev/girl/styles"
	"goki.dev/grr"
)

// loadFontMu protects the font loading calls, which are not concurrent-safe
var loadFontMu sync.RWMutex

// FontInfo contains basic font information for choosing a given font --
// displayed in the font chooser dialog.
type FontInfo struct {

	// official regularized name of font
	Name string

	// stretch: normal, expanded, condensed, etc
	Stretch styles.FontStretch `xml:"stretch"`

	// weight: normal, bold, etc
	Weight styles.FontWeights `xml:"weight"`

	// style -- normal, italic, etc
	Style styles.FontStyles `xml:"style"`

	// example text -- styled according to font params in chooser
	Example string
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

	// An fs containing available fonts, which are typically embedded through go:embed.
	// It is initialized to contain of the default fonts located in the fonts directory
	// (https://github.com/goki/girl/tree/main/paint/fonts), but it can be extended by
	// any packages by using a merged fs package.
	FontsFS fs.FS

	// list of font paths to search for fonts
	FontPaths []string

	// Map of font name to path to file. If the path starts
	// with "fs://", it indicates that it is located in
	// [FontLib.FontsFS].
	FontsAvail map[string]string

	// information about each font -- this list should be used for selecting valid regularized font names
	FontInfo []FontInfo

	// double-map of cached fonts, by font name and then integer font size within that
	Faces map[string]map[int]*styles.FontFace
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
		fl.FontsFS = grr.Log(fs.Sub(defaultFonts, "fonts"))
		fl.FontPaths = make([]string, 0)
		fl.FontsAvail = make(map[string]string)
		fl.FontInfo = make([]FontInfo, 0)
		fl.Faces = make(map[string]map[int]*styles.FontFace)
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
func (fl *FontLib) Font(fontnm string, size int) (*styles.FontFace, error) {
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

	path := fl.FontsAvail[fontnm]
	if path == "" {
		loadFontMu.RUnlock()
		return nil, fmt.Errorf("gi.FontLib: Font named: %v not found in list of available fonts; try adding to FontPaths in gi.FontLibrary; searched FontLib.FontsFS and paths: %v", fontnm, fl.FontPaths)
	}

	var bytes []byte

	if strings.HasPrefix(path, "fs://") {
		b, err := fs.ReadFile(fl.FontsFS, strings.TrimPrefix(path, "fs://"))
		if err != nil {
			err = fmt.Errorf("error opening font file for font %q in FontsFS: %w", fontnm, err)
			slog.Error(err.Error())
			return nil, err
		}
		bytes = b
	} else {
		b, err := os.ReadFile(path)
		if err != nil {
			err = fmt.Errorf("error opening font file for font %q with path %q: %w", fontnm, path, err)
			slog.Error(err.Error())
			return nil, err
		}
		bytes = b
	}

	loadFontMu.RUnlock()
	loadFontMu.Lock()
	face, err := OpenFontFace(bytes, fontnm, path, size, 0)
	if err != nil || face == nil {
		if err == nil {
			err = fmt.Errorf("gi.FontLib: nil face with no error for: %v", fontnm)
		}
		slog.Error("gi.FontLib: error loading font, removed from list", "fontName", fontnm)
		loadFontMu.Unlock()
		fl.DeleteFont(fontnm)
		return nil, err
	}
	facemap := fl.Faces[fontnm]
	if facemap == nil {
		facemap = make(map[int]*styles.FontFace)
		fl.Faces[fontnm] = facemap
	}
	facemap[size] = face
	// fmt.Printf("Opened font face: %v %v\n", fontnm, size)
	loadFontMu.Unlock()
	return face, nil

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
	fl.FontPaths = append(fl.FontPaths, paths...)
	return fl.UpdateFontsAvail()
}

// UpdateFontsAvail scans for all fonts we can use on the FontPaths
func (fl *FontLib) UpdateFontsAvail() bool {
	if len(fl.FontPaths) == 0 {
		slog.Error("gi.FontLib: no font paths; need to add some")
		return false
	}
	loadFontMu.Lock()
	defer loadFontMu.Unlock()
	if len(fl.FontsAvail) > 0 {
		fl.FontsAvail = make(map[string]string)
	}
	err := fl.FontsAvailFromFS(fl.FontsFS, "fs://")
	if err != nil {
		slog.Error("gi.FontLib: error walking FontLib.FontsFS", "err", err)
	}
	for _, p := range fl.FontPaths {
		err := fl.FontsAvailFromFS(os.DirFS(p), p+string(filepath.Separator))
		if err != nil {
			slog.Error("gi.FontLib: error walking path", "path", p, "err", err)
		}
	}
	sort.Slice(fl.FontInfo, func(i, j int) bool {
		return fl.FontInfo[i].Name < fl.FontInfo[j].Name
	})

	return len(fl.FontsAvail) > 0
}

// FontsAvailFromPath scans for all fonts we can use on a given fs,
// gathering info into FontsAvail and FontInfo. It adds the given root
// path string to all paths.
func (fl *FontLib) FontsAvailFromFS(fsys fs.FS, root string) error {
	return fs.WalkDir(fsys, ".", func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("gi.FontLib: error accessing path", "path", path, "err", err)
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
		fn = styles.FixFontMods(fn)
		basefn := strings.ToLower(fn)
		if _, ok := fl.FontsAvail[basefn]; !ok {
			fl.FontsAvail[basefn] = root + path
			fi := FontInfo{Name: fn, Example: FontInfoExample}
			_, fi.Stretch, fi.Weight, fi.Style = styles.FontNameToMods(fn)
			fl.FontInfo = append(fl.FontInfo, fi)
			// fmt.Printf("added font %q at path %q\n", basefn, root+path)

		}
		return nil
	})
}

var FontExts = map[string]struct{}{
	".ttf": {},
	".ttc": {}, // note: unpack to raw .ttf to use -- otherwise only getting first font
	".otf": {}, // not yet supported
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

//go:embed fonts/*.ttf
var defaultFonts embed.FS

// FontFallbacks are a list of fallback fonts to try, at the basename level.
// Make sure there are no loops!  Include Noto versions of everything in this
// because they have the most stretch options, so they should be in the mix if
// they have been installed, and include "Roboto" options last.
var FontFallbacks = map[string]string{
	"serif":            "Times New Roman",
	"times":            "Times New Roman",
	"Times New Roman":  "Liberation Serif",
	"Liberation Serif": "NotoSerif",
	"sans-serif":       "NotoSans",
	"NotoSans":         "Roboto",
	"courier":          "Courier",
	"Courier":          "Courier New",
	"Courier New":      "NotoSansMono",
	"NotoSansMono":     "Roboto Mono",
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

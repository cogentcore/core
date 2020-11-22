// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"strings"
	"sync"

	"github.com/goki/gi/gist"
)

var (
	// faceNameCache is a cache for fast lookup of valid font face names given style specs
	faceNameCache map[string]string

	// faceNameCacheMu protects access to faceNameCache
	faceNameCacheMu sync.RWMutex
)

// FontFaceName returns the best full FaceName to use for the given font
// family(ies) (comma separated) and modifier parameters
func FontFaceName(fam string, str gist.FontStretch, wt gist.FontWeights, sty gist.FontStyles) string {
	if fam == "" {
		fam = string(gist.ThePrefs.PrefFontFamily())
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
		_, fstr, fwt, fsty := gist.FontNameToMods(strings.TrimSpace(nms[0]))
		if fstr != gist.FontStrNormal {
			str = fstr
		}
		if fwt != gist.WeightNormal {
			wt = fwt
		}
		if fsty != gist.FontNormal {
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
			fn := gist.FontNameFromMods(basenm, str, wt, sty)
			if FontLibrary.FontAvail(fn) {
				break iterloop
			}
		}
		if str != gist.FontStrNormal {
			hasStr := false
			for _, basenm = range nms {
				fn := gist.FontNameFromMods(basenm, str, gist.WeightNormal, gist.FontNormal)
				if FontLibrary.FontAvail(fn) {
					hasStr = true
					break
				}
			}
			if !hasStr { // if even basic stretch not avail, move on
				str = gist.FontStrNormal
				continue
			}
			continue
		}
		if sty == gist.FontItalic { // italic is more common, but maybe oblique exists
			didItalic = true
			if !didOblique {
				sty = gist.FontOblique
				continue
			}
			sty = gist.FontNormal
			continue
		}
		if sty == gist.FontOblique { // by now we've tried both, try nothing
			didOblique = true
			if !didItalic {
				sty = gist.FontItalic
				continue
			}
			sty = gist.FontNormal
			continue
		}
		if wt != gist.WeightNormal {
			if wt < gist.Weight400 {
				if wt != gist.WeightLight {
					wt = gist.WeightLight
					continue
				}
			} else {
				if wt != gist.WeightBold {
					wt = gist.WeightBold
					continue
				}
			}
			wt = gist.WeightNormal
			continue
		}
		if str != gist.FontStrNormal { // time to give up
			str = gist.FontStrNormal
			continue
		}
		break // tried everything
	}
	fnm := gist.FontNameFromMods(basenm, str, wt, sty)

	faceNameCacheMu.Lock()
	if faceNameCache == nil {
		faceNameCache = make(map[string]string)
	}
	faceNameCache[cacheNm] = fnm
	faceNameCacheMu.Unlock()

	return fnm
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
		fn := gist.ThePrefs.PrefFontFamily()
		if fn == "" {
			fns = []string{"Go"}
			return
		}
	}
	fns = make([]string, 0, 20)
	for _, fn := range nms {
		fn = strings.TrimSpace(fn)
		basenm, _, _, _ := gist.FontNameToMods(fn)
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

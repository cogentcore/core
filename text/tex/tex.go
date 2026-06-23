// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/text/shaped"
	"star-tex.org/x/tex"
)

var (
	texEngine *tex.LaTeXEngine
	texFonts  *dviFonts
	texMu     sync.Mutex

	// note: must be standalone to work properly for inline paths.
	// standalone cannot use standard \begin{equation} so using $\displaymath
	preamble = `\documentclass{standalone}
\begin{document}
`
	postamble = `
\end{document}
`
)

func init() {
	shaped.ShapeMath = LaTeXMath
}

type cacheKey struct {
	fontSizeDots float32
	formula      string
}

// cache results of TeX processing because it is rather slow.
var cache = map[cacheKey]*ppath.Path{}

// LaTeXMath parses a LaTeX math expression and returns a path
// rendering that expression. To activate display math mode, add an additional $
// surrounding the expression: one set of $ $ is automatically included to produce
// inline math mode rendering by default.
// The additional $ activates displaystyle math.
// fontSizeDots specifies the actual font size in dots (actual pixels)
// for a 10pt font in the DVI system.
func LaTeXMath(formula string, fontSizeDots float32) (*ppath.Path, error) {
	texMu.Lock()
	defer texMu.Unlock()

	formula = strings.TrimSpace(formula)
	if len(formula) == 0 {
		return nil, fmt.Errorf("LaTeXMath: empty formula")
	}
	ckey := cacheKey{fontSizeDots: fontSizeDots, formula: formula}
	if p, ok := cache[ckey]; ok {
		return p, nil
	}
	txt := preamble
	if formula[0] == '$' {
		txt += "$\\displaystyle " + formula[1:len(formula)-1] + " $"
	} else {
		txt += "$" + formula + "$"
	}
	txt += postamble
	if Debug {
		fmt.Println("Input:")
		fmt.Println(txt)
	}
	r := strings.NewReader(txt)
	w := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	if texEngine == nil {
		texEngine = tex.NewLaTeX()
	}
	texEngine.Stdout = stdout
	if err := texEngine.Process(w, r); err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}

	if texFonts == nil {
		texFonts = newFonts()
	}
	p, err := DVIToPath(w.Bytes(), texFonts, fontSizeDots)
	if err != nil {
		fmt.Println("got DVIToPath error:", err)
		fmt.Println(stdout.String())
		cache[ckey] = nil
		return nil, err
	}
	cache[ckey] = p
	return p, nil
}

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
	texEngine *tex.Engine
	texFonts  *dviFonts
	texMu     sync.Mutex

	preamble = `\nopagenumbers

\def\frac#1#2{{{#1}\over{#2}}}
`
)

func init() {
	shaped.ShapeMath = TeXMath
}

type cacheKey struct {
	fontSizeDots float32
	formula      string
}

// cache results of TeX processing because it is rather slow.
var cache = map[cacheKey]*ppath.Path{}

// TeXMath parses a plain TeX math expression and returns a path
// rendering that expression. This is NOT LaTeX and only \frac is defined
// as an additional math utility function, for fractions.
// To activate display math mode, add an additional $ $ surrounding the
// expression: one set of $ $ is automatically included to produce inline
// math mode rendering.
// fontSizeDots specifies the actual font size in dots (actual pixels)
// for a 10pt font in the DVI system.
func TeXMath(formula string, fontSizeDots float32) (*ppath.Path, error) {
	texMu.Lock()
	defer texMu.Unlock()

	ckey := cacheKey{fontSizeDots: fontSizeDots, formula: formula}
	if p, ok := cache[ckey]; ok {
		return p, nil
	}

	r := strings.NewReader(fmt.Sprintf(`%s $%s$
\bye
`, preamble, formula))
	w := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	if texEngine == nil {
		texEngine = tex.New()
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
		fmt.Println(stdout.String())
		cache[ckey] = nil
		return nil, err
	}
	cache[ckey] = p
	return p, nil
}

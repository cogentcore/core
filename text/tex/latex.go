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
	"star-tex.org/x/tex"
)

var (
	// theDVIFonts are the DVI fonts, as a shared resource.
	theDVIFonts   *dviFonts
	theDVIFontsMu sync.Mutex

	preamble = `\nopagenumbers

\def\frac#1#2{{{#1}\over{#2}}}
`
)

// ParseLaTeX parse a LaTeX formula (that what is between $...$) and returns a path.
// fontSizeDots specifies the actual font size in dots (actual pixels)
// for a 10pt font in the DVI system.
func ParseLaTeX(formula string, fontSizeDots float32) (*ppath.Path, error) {
	r := strings.NewReader(fmt.Sprintf(`%s $%s$`, preamble, formula))
	w := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	engine := tex.NewEngine(stdout, bytes.NewReader([]byte{}))
	if err := engine.Process(w, r); err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}

	theDVIFontsMu.Lock()
	defer theDVIFontsMu.Unlock()
	if theDVIFonts == nil {
		theDVIFonts = newFonts()
	}
	p, err := DVIToPath(w.Bytes(), theDVIFonts, fontSizeDots)
	if err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}
	return p, nil
}

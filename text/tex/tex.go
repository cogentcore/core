// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/tex/texcache"
	tex "github.com/cogentcore/star-tex"
)

//go:embed texmf
var texmf embed.FS

var (
	texEngine *tex.LaTeXEngine
	texFonts  *dviFonts
	texMu     sync.Mutex
	// this is set to path where texmf files were installed
	texmfAt string

	// note: must be standalone to work properly for inline paths.
	// standalone cannot use standard \begin{equation} so using $\displaymath
	preamble = `\documentclass{standalone}
\usepackage{amsmath}
\begin{document}
`
	postamble = `
\end{document}
`
)

func init() {
	shaped.ShapeMath = LaTeXMath
}

// LaTeXMath parses a LaTeX math expression and returns a path
// rendering that expression. To activate display math mode, add an additional $
// surrounding the expression: one set of $ $ is automatically included to produce
// inline math mode rendering by default.
// The additional $ activates displaystyle math.
// fontSizeDots specifies the actual font size in dots (actual pixels)
// for a 10pt font in the DVI system.
func LaTeXMath(expr string, fontSizeDots float32) (ppath.Path, error) {
	texMu.Lock()
	defer texMu.Unlock()

	expr = strings.TrimSpace(expr)
	if len(expr) == 0 {
		return nil, fmt.Errorf("LaTeXMath: empty expr")
	}

	p := texcache.Get(expr, fontSizeDots)
	if p != nil {
		return p, nil
	}

	InstallTexMF()

	txt := preamble
	if expr[0] == '$' {
		txt += "$\\displaystyle " + expr[1:len(expr)-1] + " $"
	} else {
		txt += "$" + expr + "$"
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
		return nil, err
	}
	texcache.Add(expr, fontSizeDots, p)
	return p, nil
}

// InstallTexMF installs the specific TeX class and style files that
// we depend on, if not yet installed. Returns true if we just did
// the install. Must be called under texMu lock.
func InstallTexMF() bool {
	if texmfAt != "" {
		return false
	}
	trgDir := ""
	if system.TheApp != nil {
		trgDir = system.TheApp.CogentCoreDataDir()
	} else {
		// presumably testing, just use local files and be done!
		dir := "./texmf"
		texmfAt = dir
		os.Setenv("TEXMF", dir)
		return true
	}

	dir := filepath.Join(trgDir, "texmf")
	err := os.MkdirAll(dir, 0x777)
	if errors.Log(err) != nil {
		return false
	}
	files, err := texmf.ReadDir("texmf")
	if errors.Log(err) != nil {
		return false
	}
	for _, de := range files {
		fn := de.Name()
		fmt.Println(fn)
		fsp := filepath.Join("texmf", fn)
		b, err := fs.ReadFile(texmf, fsp)
		if errors.Log(err) != nil {
			return false
		}
		op := filepath.Join(dir, fn)
		os.WriteFile(op, b, 0666)
	}
	texmfAt = dir
	os.Setenv("TEXMF", dir)
	return true
}

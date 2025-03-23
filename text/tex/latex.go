// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"bytes"
	"fmt"
	"strings"

	"cogentcore.org/core/paint/ppath"
	"star-tex.org/x/tex"
)

var preamble = `\nopagenumbers

\def\frac#1#2{{{#1}\over{#2}}}
`

// ParseLaTeX parse a LaTeX formula (that what is between $...$) and returns a path.
func ParseLaTeX(formula string) (*ppath.Path, error) {
	r := strings.NewReader(fmt.Sprintf(`%s $%s$`, preamble, formula))
	w := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	engine := tex.NewEngine(stdout, bytes.NewReader([]byte{}))
	if err := engine.Process(w, r); err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}

	p, err := DVI2Path(w.Bytes(), newFonts())
	if err != nil {
		fmt.Println(stdout.String())
		return nil, err
	}
	return p, nil
}

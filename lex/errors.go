// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Much of this is directly copied from Go's go/scanner package:

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"goki.dev/ki/v2/ints"
	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
)

// In an ErrorList, an error is represented by an *Error.
// The position Pos, if valid, points to the beginning of
// the offending token, and the error condition is described
// by Msg.
type Error struct {

	// position where the error occurred in the source
	Pos Pos `desc:"position where the error occurred in the source"`

	// full filename with path
	Filename string `desc:"full filename with path"`

	// brief error message
	Msg string `desc:"brief error message"`

	// line of source where error was
	Src string `desc:"line of source where error was"`

	// lexer or parser rule that emitted the error
	Rule ki.Ki `desc:"lexer or parser rule that emitted the error"`
}

// Error implements the error interface -- gives the minimal version of error string
func (e Error) Error() string {
	if e.Filename != "" {
		_, fn := filepath.Split(e.Filename)
		return fn + ":" + e.Pos.String() + ": " + e.Msg
	}
	return e.Pos.String() + ": " + e.Msg
}

// Report provides customizable output options for viewing errors:
// - basepath if non-empty shows filename relative to that path.
// - showSrc shows the source line on a second line -- truncated to 30 chars around err
// - showRule prints the rule name
func (e Error) Report(basepath string, showSrc, showRule bool) string {
	var err error
	fnm := ""
	if e.Filename != "" {
		if basepath != "" {
			fnm, err = filepath.Rel(basepath, e.Filename)
		}
		if basepath == "" || err != nil {
			_, fnm = filepath.Split(e.Filename)
		}
	}
	str := fnm + ":" + e.Pos.String() + ": " + e.Msg
	if showRule && !kit.IfaceIsNil(e.Rule) {
		str += fmt.Sprintf(" (rule: %v)", e.Rule.Name())
	}
	ssz := len(e.Src)
	if showSrc && ssz > 0 && ssz >= e.Pos.Ch {
		str += "<br>\n\t> "
		if ssz > e.Pos.Ch+30 {
			str += e.Src[e.Pos.Ch : e.Pos.Ch+30]
		} else if ssz > e.Pos.Ch {
			str += e.Src[e.Pos.Ch:]
		}
	}
	return str
}

// ErrorList is a list of *Errors.
// The zero value for an ErrorList is an empty ErrorList ready to use.
type ErrorList []*Error

// Add adds an Error with given position and error message to an ErrorList.
func (p *ErrorList) Add(pos Pos, fname, msg string, srcln string, rule ki.Ki) *Error {
	e := &Error{pos, fname, msg, srcln, rule}
	*p = append(*p, e)
	return e
}

// Reset resets an ErrorList to no errors.
func (p *ErrorList) Reset() { *p = (*p)[0:0] }

// ErrorList implements the sort Interface.
func (p ErrorList) Len() int      { return len(p) }
func (p ErrorList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p ErrorList) Less(i, j int) bool {
	e := p[i]
	f := p[j]
	if e.Filename != f.Filename {
		return e.Filename < f.Filename
	}
	if e.Pos.Ln != f.Pos.Ln {
		return e.Pos.Ln < f.Pos.Ln
	}
	if e.Pos.Ch != f.Pos.Ch {
		return e.Pos.Ch < f.Pos.Ch
	}
	return e.Msg < e.Msg
}

// Sort sorts an ErrorList. *Error entries are sorted by position,
// other errors are sorted by error message, and before any *Error
// entry.
func (p ErrorList) Sort() {
	sort.Sort(p)
}

// RemoveMultiples sorts an ErrorList and removes all but the first error per line.
func (p *ErrorList) RemoveMultiples() {
	sort.Sort(p)
	var last Pos // initial last.Ln is != any legal error line
	var lastfn string
	i := 0
	for _, e := range *p {
		if e.Filename != lastfn || e.Pos.Ln != last.Ln {
			last = e.Pos
			lastfn = e.Filename
			(*p)[i] = e
			i++
		}
	}
	(*p) = (*p)[0:i]
}

// An ErrorList implements the error interface.
func (p ErrorList) Error() string {
	switch len(p) {
	case 0:
		return "no errors"
	case 1:
		return p[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", p[0], len(p)-1)
}

// Err returns an error equivalent to this error list.
// If the list is empty, Err returns nil.
func (p ErrorList) Err() error {
	if len(p) == 0 {
		return nil
	}
	return p
}

// Report returns all (or up to maxN if > 0) errors in the list in one string
// with customizable output options for viewing errors:
// - basepath if non-empty shows filename relative to that path.
// - showSrc shows the source line on a second line -- truncated to 30 chars around err
// - showRule prints the rule name
func (p ErrorList) Report(maxN int, basepath string, showSrc, showRule bool) string {
	ne := len(p)
	if ne == 0 {
		return ""
	}
	str := ""
	if maxN == 0 {
		maxN = ne
	} else {
		maxN = ints.MinInt(ne, maxN)
	}
	cnt := 0
	lstln := -1
	for ei := 0; ei < ne; ei++ {
		er := p[ei]
		if er.Pos.Ln == lstln {
			continue
		}
		str += p[ei].Report(basepath, showSrc, showRule) + "<br>\n"
		lstln = er.Pos.Ln
		cnt++
		if cnt > maxN {
			break
		}
	}
	if ne > maxN {
		str += fmt.Sprintf("... and %v more errors<br>\n", ne-maxN)
	}
	return str
}

// PrintError is a utility function that prints a list of errors to w,
// one error per line, if the err parameter is an ErrorList. Otherwise
// it prints the err string.
func PrintError(w io.Writer, err error) {
	if list, ok := err.(ErrorList); ok {
		for _, e := range list {
			fmt.Fprintf(w, "%s\n", e)
		}
	} else if err != nil {
		fmt.Fprintf(w, "%s\n", err)
	}
}

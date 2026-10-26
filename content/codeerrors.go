// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"errors"
	"fmt"
	"unicode"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
)

// positionedError is an error of running code at a position in the code,
// such as those of cogentcore.org/core/mvmcore and of Goal.
type positionedError interface {
	error

	// CodePos returns the 1-based line and column (in bytes) of the error
	// in the code, 0 if not known.
	CodePos() (line, col int)

	// CodeMessage returns the message of the error, without its position.
	CodeMessage() string
}

// ShowCodeErrors shows the given error of running the code of the given
// text editor, if non-nil, and otherwise clears the errors of the editor.
// The errors at a position in the code are marked in the editor (see
// [textcore.Base.SetCodeErrors]), whose tooltips are their messages, and
// the messages of all of the errors are shown in the given parent, below
// the output of the code.
func ShowCodeErrors(ed *textcore.Editor, parent *core.Frame, err error) {
	if err == nil {
		ed.SetCodeErrors()
		return
	}
	var marks []textcore.CodeError
	var msgs []string
	for _, e := range splitErrors(err) {
		pe, ok := e.(positionedError)
		if !ok {
			msgs = append(msgs, e.Error())
			continue
		}
		line, col := pe.CodePos()
		if line <= 0 {
			msgs = append(msgs, pe.CodeMessage())
			continue
		}
		msgs = append(msgs, fmt.Sprintf("%d:%d: %s", line, col, pe.CodeMessage()))
		if reg, ok := errorRegion(ed, line, col); ok {
			marks = append(marks, textcore.CodeError{Region: reg, Msg: pe.CodeMessage()})
		}
	}
	ed.SetCodeErrors(marks...)
	fr := core.NewFrame(parent)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 0)
	})
	for _, msg := range msgs {
		tx := core.NewText(fr).SetText(msg)
		tx.Styler(func(s *styles.Style) {
			s.Color = colors.Scheme.Error.Base
			s.Font.Family = rich.Monospace
			s.Grow.Set(1, 0)
		})
	}
}

// splitErrors returns the errors of the given error, which are the
// errors that it joins, if it joins errors, with any of those that are
// at a position in the code.
func splitErrors(err error) []error {
	if _, ok := err.(positionedError); ok {
		return []error{err}
	}
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		var errs []error
		for _, e := range j.Unwrap() {
			errs = append(errs, splitErrors(e)...)
		}
		return errs
	}
	var pe positionedError
	if errors.As(err, &pe) {
		return []error{pe}
	}
	return []error{err}
}

// errorRegion returns the region of the text of the given editor of an
// error at the given 1-based line and column (in bytes): that of the word
// or other character at the position.
func errorRegion(ed *textcore.Editor, line, col int) (textpos.Region, bool) {
	ln := line - 1
	if ed.Lines == nil || ln >= ed.Lines.NumLines() {
		return textpos.Region{}, false
	}
	rs := ed.Lines.Line(ln)
	st := len(byteRunes(rs, col-1)) // the column is in bytes; the region is in runes
	end := st
	isWord := func(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }
	for end < len(rs) && isWord(rs[end]) {
		end++
	}
	if end == st {
		end = min(st+1, len(rs))
	}
	if end == st { // an error at the end of the line
		st = max(st-1, 0)
	}
	return textpos.Region{Start: textpos.Pos{Line: ln, Char: st}, End: textpos.Pos{Line: ln, Char: end}}, end > st
}

// byteRunes returns the runes of the given line that are in its first n
// bytes, in UTF-8.
func byteRunes(rs []rune, n int) []rune {
	b := 0
	for i, r := range rs {
		if b >= n {
			return rs[:i]
		}
		b += len(string(r))
	}
	return rs
}

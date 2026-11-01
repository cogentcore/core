// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"go/format"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/text/textpos"
)

// FormatCode are the functions that format the code of the code blocks
// of each language (such as "Go") when it is edited, as gofmt does for
// Go code, which is the default. Other packages can add other languages,
// as cogentcore.org/lab/mvmlab does Goal. A function returns an error
// for code that it cannot format, which is left as it is.
var FormatCode = map[string]func(src string) (string, error){
	"Go": formatGo,
}

// formatGo formats the given Go code, which can be a list of
// declarations or statements, as gofmt does.
func formatGo(src string) (string, error) {
	b, err := format.Source([]byte(src))
	return string(b), err
}

// bindFormat formats the code of the given editor of a code block of the
// given language when it is edited (see [FormatCode]), before it runs.
func bindFormat(ed *textcore.Editor, language string) {
	if FormatCode[language] == nil {
		return
	}
	ed.OnFirst(events.Change, func(e events.Event) {
		formatEditor(ed, language)
	})
}

// formatEditor formats the code of the given editor of a code block of
// the given language, as an edit that can be undone, keeping the cursor
// on its line. The editor is not marked as changed, so that formatting
// does not run the code again.
func formatEditor(ed *textcore.Editor, language string) {
	f := FormatCode[language]
	ln := ed.Lines
	if f == nil || ln == nil {
		return
	}
	src := ln.String()
	res, err := f(src)
	if err != nil || res == src { // errors are shown when the code runs
		return
	}
	// the text ends with a newline after the end of its last line
	if strings.HasSuffix(src, "\n") {
		res = strings.TrimSuffix(res, "\n")
	}
	cur := ed.CursorPos
	ln.ReplaceText(textpos.Pos{}, ln.EndPos(), textpos.Pos{}, res, lines.ReplaceNoMatchCase)
	ln.SetChanged(false)
	cur.Line = min(cur.Line, ln.NumLines()-1)
	cur.Char = min(cur.Char, ln.LineLen(cur.Line))
	ed.SetCursorShow(cur)
}

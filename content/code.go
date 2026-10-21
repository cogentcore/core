// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textcore"
)

// CodeFunc is a Go code block compiled into the app at build time
// (see cogentcore.org/core/content/cmd/contentcode). It runs the code
// with the given parent widget, which the code knows as b, and writes
// what the code prints to out.
type CodeFunc func(b *core.Frame, out io.Writer)

// CompiledCode maps the [CodeKey] of the source of a code block to the
// function compiled from it at build time. A code block whose source has
// an entry here runs without an interpreter.
var CompiledCode = map[string]CodeFunc{}

// AddCompiledCode adds the given compiled code blocks to [CompiledCode].
// Generated code calls it.
func AddCompiledCode(code map[string]CodeFunc) {
	for k, f := range code {
		CompiledCode[k] = f
	}
}

// CodeKey returns the key of the given code block source in [CompiledCode]:
// a hash of the source without any whitespace, so that indentation (tabs
// or spaces) and blank lines do not matter. The generator in
// cogentcore.org/core/content/cmd/contentcode computes the same key.
func CodeKey(src string) string {
	s := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, src)
	h := uint32(2166136261) // FNV-1a
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return fmt.Sprintf("%08x-%d", h, len(s))
}

// BindCode runs the code of the given text editor in the given parent,
// and again whenever the code is edited. Code that was compiled into the
// app (see [CompiledCode]) runs directly. Other code, including edited
// code, runs in the interpreter of [BindTextEditor] if there is one
// (yaegi, through cogentcore.org/core/yaegicore); otherwise an edit shows
// a message that this app can not run edited code.
func BindCode(ed *textcore.Editor, parent *core.Frame, language string) {
	if language == "Go" {
		if f := CompiledCode[CodeKey(ed.Lines.String())]; f != nil {
			bindCompiledCode(ed, parent, language, f)
			return
		}
	}
	if BindTextEditor != nil {
		BindTextEditor(ed, parent, language)
		return
	}
	// The code was not compiled (the generator could not compile it) and
	// there is no interpreter: it is shown, but does not run.
	errors.Log(fmt.Errorf("content: %s code block was not compiled into the app and there is no interpreter to run it", language))
}

// bindCompiledCode runs f now, and hands the editor to the interpreter
// once the code is edited.
func bindCompiledCode(ed *textcore.Editor, parent *core.Frame, language string, f CodeFunc) {
	key := CodeKey(ed.Lines.String())
	interpreted := false
	ed.OnChange(func(e events.Event) {
		if interpreted {
			return
		}
		src := ed.Lines.String()
		if g := CompiledCode[CodeKey(src)]; g != nil {
			if CodeKey(src) != key {
				key = CodeKey(src)
				f = g
				runCompiledCode(ed, parent, f)
			}
			return
		}
		if BindTextEditor == nil {
			core.ErrorSnackbar(ed, fmt.Errorf("this app has no interpreter to run edited code (only the code compiled into it)"), "Error running "+language+" code")
			return
		}
		// BindTextEditor runs the code now and on later changes.
		interpreted = true
		BindTextEditor(ed, parent, language)
	})
	runCompiledCode(ed, parent, f)
}

// runCompiledCode runs f in parent, like the interpreter would.
func runCompiledCode(ed *textcore.Editor, parent *core.Frame, f CodeFunc) {
	parent.DeleteChildren()
	core.ExternalParent = parent
	var out bytes.Buffer
	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		f(parent, &out)
		return nil
	}()
	if ostr := out.String(); len(strings.TrimSpace(ostr)) > 0 {
		oe := textcore.NewEditor(parent)
		oe.SetReadOnly(true)
		oe.Lines.Settings.LineNumbers = false
		oe.Lines.SetString(ostr)
	}
	if err != nil {
		core.ErrorSnackbar(ed, err, "Error running Go code")
		return
	}
	parent.Update()
}

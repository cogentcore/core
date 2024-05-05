// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"reflect"
	"strings"

	"cogentcore.org/core/shell"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Interpreter represents one running shell context
type Interpreter struct {
	Shell *shell.Shell

	Interp *interp.Interpreter
}

func NewInterpreter() *Interpreter {
	in := &Interpreter{}
	in.Interp = interp.New(interp.Options{})
	in.Interp.Use(stdlib.Symbols) // this causes symbols to crash
	in.Shell = shell.NewShell()
	return in
}

// SymbolByName returns the reflect.Value for given symbol name
// from the current Globals, Symbols (must call GetSymbols first)
func (in *Shell) SymbolByName(name string) (bool, reflect.Value) {
	globs := sh.Interp.Globals()
	syms := sh.Interp.Symbols("main") // note: cannot use ""

	nmorig := name
	nmpath := ""
	dotIdx := strings.Index(name, ".")
	if dotIdx > 0 {
		nmpath = name[:dotIdx]
		name = name[dotIdx+1:]
	}
	for path, sy := range syms {
		in.DebugPrintln(2, "	searching symbol path:", path)
		if nmpath != "" && path != nmpath {
			continue
		}
		for nm, v := range sy {
			in.DebugPrintln(2, "	checking symbol name:", nm)
			if nm == name {
				in.DebugPrintln(1, "	got symbol:", path+"."+nm)
				return true, v
			}
		}
	}

	for nm, v := range globs {
		in.DebugPrintln(2, "	searching global name:", nm)
		if nm == name {
			in.DebugPrintln(1, "	got global:", nm)
			return true, v
		}
	}
	in.DebugPrintln(1, "	symbol not found:", nmorig)
	return false, reflect.Value{}
}

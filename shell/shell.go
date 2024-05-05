// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"reflect"
	"strings"

	"cogentcore.org/core/base/exec"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Shell represents one running shell context
type Shell struct {
	// debug levels: 2 = full detail, 1 = summary, 0 = none
	Debug int
}

func NewShell() *Shell {
	sh := &Shell{}
	sh.Interp = interp.New(interp.Options{})
	sh.Interp.Use(stdlib.Symbols) // this causes symbols to crash
	sh.Exec = exec.Major()
	sh.Exec.Commands = nil
	return sh
}

// GetSymbols gets the current symbols from the interpreter
func (sh *Shell) GetSymbols() {
	sh.Globals = sh.Interp.Globals()
	sh.Symbols = sh.Interp.Symbols("main") // note: cannot use ""
}

// SymbolByName returns the reflect.Value for given symbol name
// from the current Globals, Symbols (must call GetSymbols first)
func (sh *Shell) SymbolByName(name string) (bool, reflect.Value) {
	nmorig := name
	nmpath := ""
	dotIdx := strings.Index(name, ".")
	if dotIdx > 0 {
		nmpath = name[:dotIdx]
		name = name[dotIdx+1:]
	}
	for path, sy := range sh.Symbols {
		sh.DebugPrintln(2, "	searching symbol path:", path)
		if nmpath != "" && path != nmpath {
			continue
		}
		for nm, v := range sy {
			sh.DebugPrintln(2, "	checking symbol name:", nm)
			if nm == name {
				sh.DebugPrintln(1, "	got symbol:", path+"."+nm)
				return true, v
			}
		}
	}

	for nm, v := range sh.Globals {
		sh.DebugPrintln(2, "	searching global name:", nm)
		if nm == name {
			sh.DebugPrintln(1, "	got global:", nm)
			return true, v
		}
	}
	sh.DebugPrintln(1, "	symbol not found:", nmorig)
	return false, reflect.Value{}
}

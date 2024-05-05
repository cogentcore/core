// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package interpreter

import (
	"cogentcore.org/core/shell"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Interp represents one running shell context
type Interp struct {
	Shell *shell.Shell

	Interp *interp.Interpreter
}

func NewInterp() *Interp {
	sh := &Interp{}
	sh.Interp = interp.New(interp.Options{})
	sh.Interp.Use(stdlib.Symbols) // this causes symbols to crash
	sh.Shell = shell.NewShell()
	return sh
}

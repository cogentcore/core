// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"context"

	"cogentcore.org/core/base/exec"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Context has the current parsing context
type Context struct {
	Interp *interp.Interpreter

	YContext context.Context

	Config *exec.Config

	// Globals map[string]reflect.Value
	//
	// Symbols yaegi.Exports
}

func NewContext() *Context {
	ctx := &Context{}
	ctx.Interp = interp.New(interp.Options{})
	ctx.Interp.Use(stdlib.Symbols)
	ctx.Config = exec.Major()
	return ctx
}

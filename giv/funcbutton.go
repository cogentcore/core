// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/glop/sentencecase"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
)

// FuncButton is a button that is set up to call a function when it
// is pressed, using a dialog to prompt the user for any arguments.
// Also, it automatically sets various properties of the button like
// the text, tooltip, and icon based on the properties of the
// function, using reflect and gti. The function must be registered
// with gti; add a `//gti:add` comment directive and run `goki generate`
// if you get errors. If the function is a method, both the method and
// its receiver type must be added to gti.
type FuncButton struct {
	gi.Button
	// Func is the [gti.Func] associated with this button.
	// This function can also be a method, but it must be
	// converted to a [gti.Func] first. It should typically
	// be set using [FuncButton.SetFunc].
	Func *gti.Func
}

// SetFunc sets the function associated with the FuncButton to the
// given function or method value, which must be added to gti.
func (fb *FuncButton) SetFunc(fun any) *FuncButton {
	fnm := gti.FuncName(fun)
	// the "-fm" suffix indicates that it is a method
	if !strings.HasSuffix(fnm, "-fm") {
		f := gti.FuncByName(fnm)
		if f == nil {
			slog.Error("programmer error: cannot use giv.NewFuncButton with a function that has not been added to gti; see the documentation for giv.NewFuncButton", "function", fnm)
			return nil
		}
		return fb.SetFuncImpl(f, reflect.ValueOf(fun))
	}

	fnm = strings.TrimSuffix(fnm, "-fm")
	// the last dot separates the function name
	li := strings.LastIndex(fnm, ".")
	metnm := fnm[li+1:]
	typnm := fnm[:li]
	// get rid of any parentheses and pointer receivers
	// that may surround the type name
	typnm = strings.ReplaceAll(typnm, "(*", "")
	typnm = strings.TrimSuffix(typnm, ")")
	gtyp := gti.TypeByName(typnm)
	if gtyp == nil {
		slog.Error("programmer error: cannot use giv.NewFuncButton with a method whose receiver type has not been added to gti; see the documentation for giv.NewFuncButton", "type", typnm, "method", metnm, "fullPath", fnm)
		return nil
	}
	met := gtyp.Methods.ValByKey(metnm)
	if met == nil {
		slog.Error("programmer error: cannot use giv.NewFuncButton with a method that has not been added to gti (even though the receiver type was); see the documentation for giv.NewFuncButton", "type", typnm, "method", metnm, "fullPath", fnm)
		return nil
	}
	return fb.SetMethodImpl(met, reflect.ValueOf(fun))
}

// SetFuncImpl is the underlying implementation of [FuncButton.SetFunc].
// It should typically not be used by end-user code.
func (fb *FuncButton) SetFuncImpl(gfun *gti.Func, rfun reflect.Value) *FuncButton {
	fb.Func = gfun
	// get name without package
	snm := gfun.Name
	li := strings.LastIndex(snm, ".")
	if li >= 0 {
		snm = snm[li+1:] // must also get rid of "."
	}
	fb.SetText(sentencecase.Of(snm))
	fb.SetTooltip(gfun.Doc)
	// we default to the icon with the same name as
	// the function, if it exists
	ic := icons.Icon(strcase.ToSnake(snm))
	if ic.IsValid() {
		fb.SetIcon(ic)
	}
	fb.OnClick(func(e events.Event) {
		fmt.Println("calling", fb.Func.Name)
	})
	return fb
}

// SetMethodImpl is the underlying implementation of [FuncButton.SetFunc] for methods.
// It should typically not be used by end-user code.
func (fb *FuncButton) SetMethodImpl(gmet *gti.Method, rmet reflect.Value) *FuncButton {
	return fb.SetFuncImpl(&gti.Func{
		Name:       gmet.Name,
		Doc:        gmet.Doc,
		Directives: gmet.Directives,
		Args:       gmet.Args,
		Returns:    gmet.Returns,
	}, rmet)
}

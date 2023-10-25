// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"log/slog"
	"reflect"
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/glop/sentencecase"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
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
	Func *gti.Func `set:"-"`

	// ReflectFunc is the [reflect.Value] of the function or
	// method associated with this button. It should typically
	// bet set using [FuncButton.SetFunc].
	ReflectFunc reflect.Value `set:"-"`

	// Args are the [ArgConfig] objects associated with
	// the function button. They are automatically set in
	// [SetFunc], but they can be customized to configure
	// default values and other options.
	Args []ArgConfig `set:"-"`

	// Confirm is whether to prompt the user for confirmation
	// before calling the function.
	Confirm bool

	// ShowReturn is whether to display the return values of
	// the function (and a success message if there are none).
	// The way that the return values are shown is determined
	// by ShowReturnAsDialog. ShowReturn is on by default.
	ShowReturn bool `def:"true"`

	// ShowReturnAsDialog, if and only if ShowReturn is true,
	// indicates to show the return values of the function in
	// a dialog, instead of in a snackbar, as they are by default.
	// If there is a return value from the function of a complex
	// type (struct, slice, map), then ShowReturnAsDialog will
	// automatically be set to true.
	ShowReturnAsDialog bool
}

func (fb *FuncButton) OnInit() {
	fb.Button.OnInit()
	fb.ShowReturn = true
}

// SetFunc sets the function associated with the FuncButton to the
// given function or method value. For documentation information for
// the function to be obtained, it must be added to gti.
func (fb *FuncButton) SetFunc(fun any) *FuncButton {
	fnm := gti.FuncName(fun)
	// the "-fm" suffix indicates that it is a method
	if strings.HasSuffix(fnm, "-fm") {
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
		var met *gti.Method
		if gtyp == nil {
			slog.Info("warning for programmer: giv.FuncButton.SetFunc called with a method whose receiver type has not been added to gti, meaning documentation information can not be obtained; see the documentation for giv.FuncButton for more information", "function", fnm)
			met = &gti.Method{Name: metnm}
		} else {
			met = gtyp.Methods.ValByKey(metnm)
			if met == nil {
				slog.Info("warning for programmer: giv.FuncButton.SetFunc called with a method that has not been added to gti (even though the receiver type was, you still need to add the method itself), meaning documentation information can not be obtained; see the documentation for giv.FuncButton for more information", "function", fnm)
				met = &gti.Method{Name: metnm}
			}
		}
		return fb.SetMethodImpl(met, reflect.ValueOf(fun))
	}

	rs := []rune(fnm)
	// FuncName.funcN indicates that a function was defined anonymously
	if len(rs) > 0 && unicode.IsDigit(rs[len(rs)-1]) && strings.Contains(fnm, ".func") {
		fnm = strings.TrimRightFunc(fnm, func(r rune) bool {
			return unicode.IsDigit(r)
		})
		fnm = strings.TrimSuffix(fnm, ".func")
		f := &gti.Func{Name: fnm, Doc: "Anonymous function defined in " + fnm}
		return fb.SetFuncImpl(f, reflect.ValueOf(fun))
	}

	f := gti.FuncByName(fnm)
	if f == nil {
		slog.Info("warning for programmer: giv.FuncButton.SetFunc called with a function that has not been added to gti, meaning documentation information can not be obtained; see the documentation for giv.FuncButton for more information", "function", fnm)
		f = &gti.Func{Name: fnm}
	}
	return fb.SetFuncImpl(f, reflect.ValueOf(fun))
}

// SetFuncImpl is the underlying implementation of [FuncButton.SetFunc].
// It should typically not be used by end-user code.
func (fb *FuncButton) SetFuncImpl(gfun *gti.Func, rfun reflect.Value) *FuncButton {
	fb.Func = gfun
	fb.ReflectFunc = rfun
	fb.SetArgs()
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
		fb.CallFunc()
	})
	return fb
}

// CallFunc calls the function or method associated with this button,
// prompting the user for any arguments.
func (fb *FuncButton) CallFunc() {
	if len(fb.Args) == 0 {
		if !fb.Confirm {
			rets := fb.ReflectFunc.Call(nil)
			fb.ShowReturnsDialog(rets)
			return
		}
		gi.NewStdDialog(fb.This().(gi.Widget), gi.DlgOpts{Title: fb.Text + "?", Prompt: "Are you sure you want to run " + fb.Text + "? " + fb.Tooltip, Ok: true, Cancel: true},
			func(dlg *gi.Dialog) {
				if !dlg.Accepted {
					return
				}
				rets := fb.ReflectFunc.Call(nil)
				fb.ShowReturnsDialog(rets)
			}).Run()
		return
	}
	ArgViewDialog(
		fb.This().(gi.Widget),
		DlgOpts{Title: fb.Text, Prompt: fb.Tooltip, Ok: true, Cancel: true},
		fb.Args,
		func(dlg *gi.Dialog) {
			if !dlg.Accepted {
				return
			}
			rargs := make([]reflect.Value, len(fb.Args))
			for i, arg := range fb.Args {
				rargs[i] = laser.NonPtrValue(arg.Val)
			}

			if !fb.Confirm {
				rets := fb.ReflectFunc.Call(rargs)
				fb.ShowReturnsDialog(rets)
			}
			gi.NewStdDialog(fb.This().(gi.Widget), gi.DlgOpts{Title: fb.Text + "?", Prompt: "Are you sure you want to run " + fb.Text + "? " + fb.Tooltip, Ok: true, Cancel: true},
				func(dlg *gi.Dialog) {
					if !dlg.Accepted {
						return
					}
					rets := fb.ReflectFunc.Call(rargs)
					fb.ShowReturnsDialog(rets)
				}).Run()
		},
	).Run()
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

// ShowReturnsDialog runs a dialog displaying the given function return
// values for the function associated with the function button. It does
// nothing if [FuncButton.ShowReturn] is dialog
func (fb *FuncButton) ShowReturnsDialog(rets []reflect.Value) {
	if !fb.ShowReturn {
		return
	}
	// TODO: handle error return values
	main := "Result of " + fb.Text
	if len(rets) == 0 {
		main = fb.Text + " succeeded"
	}
	if !fb.ShowReturnAsDialog {
		gi.NewSnackbar(fb.This().(gi.Widget), gi.SnackbarOpts{Text: main}).Run()
		return
	}
	ac := fb.Returns(rets)
	ArgViewDialog(fb.This().(gi.Widget), DlgOpts{Title: main, Prompt: fb.Tooltip, Ok: true}, ac, nil).Run()
}

// ArgsForFunc sets the appropriate [ArgConfig] objects for the
// arguments of the function associated with the function button.
// It is called in [FuncButton.SetFunc] and should typically not
// be called by end-user code.
func (fb *FuncButton) SetArgs() {
	narg := fb.ReflectFunc.Type().NumIn()
	fb.Args = make([]ArgConfig, narg)
	for i := range fb.Args {
		ra := ArgConfig{}
		atyp := fb.ReflectFunc.Type().In(i)

		if fb.Func.Args != nil {
			ga := fb.Func.Args.ValByIdx(i)
			if ga != nil {
				ra.Name = ga.Name
				ra.Doc = ga.Doc
			} else {
				ra.Name = laser.NonPtrType(atyp).Name()
				ra.Doc = "Unnamed argument of type " + laser.LongTypeName(atyp)
			}
		} else {
			ra.Name = laser.NonPtrType(atyp).Name()
			ra.Doc = "Unnamed argument of type " + laser.LongTypeName(atyp)
		}

		ra.Label = sentencecase.Of(ra.Name)
		ra.Val = reflect.New(atyp)

		ra.View = ToValue(ra.Val.Interface(), "")
		ra.View.SetSoloValue(ra.Val)
		ra.View.SetName(ra.Name)
		fb.Args[i] = ra
	}
}

// ReturnsForFunc returns the appropriate [ArgConfig] objects for the given return values
// of the function associated with the function button.
func (fb *FuncButton) Returns(rets []reflect.Value) []ArgConfig {
	res := make([]ArgConfig, fb.Func.Returns.Len())
	for i, kv := range fb.Func.Returns.Order {
		ret := kv.Val
		ra := ArgConfig{
			Name:  ret.Name,
			Label: sentencecase.Of(ret.Name),
			Doc:   ret.Doc,
		}

		ra.Val = rets[i]

		ra.View = ToValue(ra.Val.Interface(), "")
		ra.View.SetSoloValue(ra.Val)
		ra.View.SetName(ra.Name)
		ra.View.SetFlag(true, states.ReadOnly)
		res[i] = ra
	}
	return res
}

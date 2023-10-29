// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/glop/sentencecase"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// FuncButton is a button that is set up to call a function when it
// is pressed, using a dialog to prompt the user for any arguments.
// Also, it automatically sets various properties of the button like
// the name, text, tooltip, and icon based on the properties of the
// function, using reflect and gti. The function must be registered
// with gti to get documentation information, but that is not required;
// add a `//gti:add` comment directive and run `goki generate`
// if you want tooltips. If the function is a method, both the method and
// its receiver type must be added to gti to get documentation.
type FuncButton struct { //goki:no-new
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

	// Args are the [Value] objects associated with the
	// arguments of the function. They are automatically set in
	// [SetFunc], but they can be customized to configure
	// default values and other options.
	Args []Value `set:"-"`

	// Returns are the [Value] objects associated with the
	// return values of the function. They are automatically
	// set in [SetFunc], but they can be customized to configure
	// default values and other options. The [reflect.Value]s of
	// the [Value] objects are not set until the function is
	// called, and are thus not typically applicable to access.
	Returns []Value `set:"-"`

	// Confirm is whether to prompt the user for confirmation
	// before calling the function.
	Confirm bool

	// ShowReturn is whether to display the return values of
	// the function (and a success message if there are none).
	// The way that the return values are shown is determined
	// by ShowReturnAsDialog. ShowReturn is on by default, unless
	// the function has no return values.
	ShowReturn bool `def:"true"`

	// ShowReturnAsDialog, if and only if ShowReturn is true,
	// indicates to show the return values of the function in
	// a dialog, instead of in a snackbar, as they are by default.
	// If there are multiple return values from the function, or if
	// one of them is a complex type (pointer, struct, slice,
	// array, map), then ShowReturnAsDialog will
	// automatically be set to true.
	ShowReturnAsDialog bool
}

// NewFuncButton adds a new [FuncButton] with the given function
// to the given parent.
func NewFuncButton(par ki.Ki, fun any) *FuncButton {
	return par.NewChild(FuncButtonType).(*FuncButton).SetFunc(fun)
}

func (fb *FuncButton) OnInit() {
	fb.Button.OnInit()
}

// SetFunc sets the function associated with the FuncButton to the
// given function or method value. For documentation information for
// the function to be obtained, it must be added to gti. SetFunc is
// automatically called by [NewFuncButton].
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
			slog.Info("warning: potential programmer error: giv.FuncButton.SetFunc called with a method whose receiver type has not been added to gti, meaning documentation information can not be obtained; see the documentation for giv.FuncButton for more information", "function", fnm)
			met = &gti.Method{Name: metnm}
		} else {
			met = gtyp.Methods.ValByKey(metnm)
			if met == nil {
				slog.Info("warning: potential programmer error: giv.FuncButton.SetFunc called with a method that has not been added to gti (even though the receiver type was, you still need to add the method itself), meaning documentation information can not be obtained; see the documentation for giv.FuncButton for more information", "function", fnm)
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
		slog.Info("warning: potential programmer error: giv.FuncButton.SetFunc called with a function that has not been added to gti, meaning documentation information can not be obtained; see the documentation for giv.FuncButton for more information", "function", fnm)
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
	fb.SetReturns()
	// get name without package
	snm := gfun.Name
	li := strings.LastIndex(snm, ".")
	if li >= 0 {
		snm = snm[li+1:] // must also get rid of "."
	}
	// func name is not guaranteed to make it unique so we ensure it is (-1 because [ki.New] adds 1 first)
	fb.SetName(snm + "-" + strconv.FormatUint(fb.Parent().NumLifetimeChildren()-1, 10))
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
		gi.NewDialog(fb).Title(fb.Text + "?").Prompt("Are you sure you want to run " + fb.Text + "? " + fb.Tooltip).Cancel().Ok().
			OnAccept(func(e events.Event) {
				rets := fb.ReflectFunc.Call(nil)
				fb.ShowReturnsDialog(rets)
			}).Run()
		return
	}
	ArgViewDialog(gi.NewDialog(fb).Title(fb.Text).Prompt(fb.Tooltip), fb.Args).Cancel().Ok().
		OnAccept(func(e events.Event) {
			rargs := make([]reflect.Value, len(fb.Args))
			for i, arg := range fb.Args {
				rargs[i] = laser.NonPtrValue(arg.Val())
			}

			if !fb.Confirm {
				rets := fb.ReflectFunc.Call(rargs)
				fb.ShowReturnsDialog(rets)
			}
			gi.NewDialog(fb).Title(fb.Text + "?").Prompt("Are you sure you want to run " + fb.Text + "? " + fb.Tooltip).Cancel().Ok().
				OnAccept(func(e events.Event) {
					rets := fb.ReflectFunc.Call(rargs)
					fb.ShowReturnsDialog(rets)
				}).Run()
		}).Run()
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
	fb.SetReturnValues(rets)
	// TODO: handle error return values
	main := "Result of " + fb.Text
	if len(rets) == 0 {
		main = fb.Text + " succeeded"
	}
	if !fb.ShowReturnAsDialog {
		txt := main
		if len(fb.Returns) > 0 {
			txt += ": "
			for i, ret := range fb.Returns {
				txt += laser.NonPtrValue(ret.Val()).String()
				if i < len(fb.Returns)-1 {
					txt += ", "
				}
			}
		}
		gi.NewSnackbar(fb, gi.SnackbarOpts{Text: txt}).Run()
		return
	}
	ArgViewDialog(gi.NewDialog(fb).Title(main).Prompt(fb.Tooltip).ReadOnly(true), fb.Returns).Ok().Run()
}

// SetArgs sets the appropriate [Value] objects for the
// arguments of the function associated with the function button.
// It is called in [FuncButton.SetFunc] and should typically not
// be called by end-user code.
func (fb *FuncButton) SetArgs() {
	narg := fb.ReflectFunc.Type().NumIn()
	fb.Args = make([]Value, narg)
	for i := range fb.Args {
		atyp := fb.ReflectFunc.Type().In(i)

		name := ""
		doc := ""
		if fb.Func.Args != nil {
			ga := fb.Func.Args.ValByIdx(i)
			if ga != nil {
				name = ga.Name
				doc = ga.Doc
			} else {
				name = laser.NonPtrType(atyp).Name()
				doc = "Unnamed argument of type " + laser.LongTypeName(atyp)
			}
		} else {
			name = laser.NonPtrType(atyp).Name()
			doc = "Unnamed argument of type " + laser.LongTypeName(atyp)
		}

		label := sentencecase.Of(name)
		val := reflect.New(atyp)

		view := ToValue(val.Interface(), "")
		view.SetSoloValue(val)
		view.SetName(name)
		view.SetLabel(label)
		view.SetDoc(doc)
		fb.Args[i] = view
	}
}

// SetReturns sets the appropriate [Value] objects for the
// return values of the function associated with the function
// button. It is called in [FuncButton.SetFunc] and should
// typically not be called by end-user code.
func (fb *FuncButton) SetReturns() {
	nret := fb.ReflectFunc.Type().NumOut()
	fb.Returns = make([]Value, nret)
	hasComplex := false
	for i := range fb.Returns {
		rtyp := fb.ReflectFunc.Type().Out(i)
		if !hasComplex {
			k := rtyp.Kind()
			if k == reflect.Pointer || k == reflect.Struct || k == reflect.Slice || k == reflect.Array || k == reflect.Map {
				hasComplex = true
			}
		}

		name := ""
		doc := ""
		if fb.Func.Returns != nil {
			ga := fb.Func.Returns.ValByIdx(i)
			if ga != nil {
				name = ga.Name
				doc = ga.Doc
			} else {
				name = laser.NonPtrType(rtyp).Name()
				doc = "Unnamed return value of type " + laser.LongTypeName(rtyp)
			}
		} else {
			name = laser.NonPtrType(rtyp).Name()
			doc = "Unnamed return value of type " + laser.LongTypeName(rtyp)
		}

		label := sentencecase.Of(name)
		val := reflect.New(rtyp)

		view := ToValue(val.Interface(), "")
		view.SetSoloValue(val)
		view.SetName(name)
		view.SetLabel(label)
		view.SetDoc(doc)
		fb.Returns[i] = view
	}
	fb.ShowReturn = nret > 0
	if nret > 1 || hasComplex {
		fb.ShowReturnAsDialog = true
	}
}

// SetReturnValues sets the [reflect.Value]s of the return
// value [Value] objects. It assumes that [FuncButton.SetReturns]
// has already been called. It is called in [FuncButton.CallFunc]
// and should typically not be called by end-user code.
func (fb *FuncButton) SetReturnValues(rets []reflect.Value) {
	for i, ret := range fb.Returns {
		ret.SetSoloValue(rets[i])
	}
}

// SetKey sets the shortcut of the function button from the given [keyfun.Funs]
func (fb *FuncButton) SetKey(kf keyfun.Funs) *FuncButton {
	fb.SetShortcut(keyfun.ShortcutFor(kf))
	return fb
}

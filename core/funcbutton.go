// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"log/slog"
	"reflect"
	"strings"
	"unicode"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// CallFunc calls the given function in the context of the given widget,
// popping up a dialog to prompt for any arguments and show the return
// values of the function. It is a helper function that uses [NewSoloFuncButton]
// under the hood.
func CallFunc(ctx Widget, fun any) {
	NewSoloFuncButton(ctx).SetFunc(fun).CallFunc()
}

// NewSoloFuncButton returns a standalone [FuncButton] with a fake parent
// with the given context for popping up any dialogs.
func NewSoloFuncButton(ctx Widget) *FuncButton {
	return NewFuncButton(NewWidgetBase()).SetContext(ctx)
}

// FuncButton is a button that is set up to call a function when it
// is pressed, using a dialog to prompt the user for any arguments.
// Also, it automatically sets various properties of the button like
// the text and tooltip based on the properties of the function,
// using [reflect] and [types]. The function must be registered
// with [types] to get documentation information, but that is not
// required; add a `//types:add` comment directive and run `core generate`
// if you want tooltips. If the function is a method, both the method and
// its receiver type must be added to [types] to get documentation.
// The main function to call first is [FuncButton.SetFunc].
type FuncButton struct {
	Button

	// typesFunc is the [types.Func] associated with this button.
	// This function can also be a method, but it must be
	// converted to a [types.Func] first. It should typically
	// be set using [FuncButton.SetFunc].
	typesFunc *types.Func

	// reflectFunc is the [reflect.Value] of the function or
	// method associated with this button. It should typically
	// bet set using [FuncButton.SetFunc].
	reflectFunc reflect.Value

	// Args are the [FuncArg] objects associated with the
	// arguments of the function. They are automatically set in
	// [FuncButton.SetFunc], but they can be customized to configure
	// default values and other options.
	Args []FuncArg `set:"-"`

	// Returns are the [FuncArg] objects associated with the
	// return values of the function. They are automatically
	// set in [FuncButton.SetFunc], but they can be customized
	// to configure options. The [FuncArg.Value]s are not set until
	// the function is called, and are thus not typically applicable
	// to access.
	Returns []FuncArg `set:"-"`

	// Confirm is whether to prompt the user for confirmation
	// before calling the function.
	Confirm bool

	// ShowReturn is whether to display the return values of
	// the function (and a success message if there are none).
	// The way that the return values are shown is determined
	// by ShowReturnAsDialog. Non-nil error return values will
	// always be shown, even if ShowReturn is set to false.
	ShowReturn bool

	// ShowReturnAsDialog, if and only if ShowReturn is true,
	// indicates to show the return values of the function in
	// a dialog, instead of in a snackbar, as they are by default.
	// If there are multiple return values from the function, or if
	// one of them is a complex type (pointer, struct, slice,
	// array, map), then ShowReturnAsDialog will
	// automatically be set to true.
	ShowReturnAsDialog bool

	// NewWindow makes the return value dialog a NewWindow dialog.
	NewWindow bool

	// WarnUnadded is whether to log warnings when a function that
	// has not been added to [types] is used. It is on by default and
	// must be set before [FuncButton.SetFunc] is called for it to
	// have any effect. Warnings are never logged for anonymous functions.
	WarnUnadded bool `default:"true"`

	// Context is used for opening dialogs if non-nil.
	Context Widget

	// AfterFunc is an optional function called after the func button
	// function is executed.
	AfterFunc func()
}

// FuncArg represents one argument or return value of a function
// in the context of a [FuncButton].
type FuncArg struct { //types:add -setters

	// Name is the name of the argument or return value.
	Name string

	// Tag contains any tags associated with the argument or return value,
	// which can be added programmatically to customize [Value] behavior.
	Tag reflect.StructTag

	// Value is the actual value of the function argument or return value.
	// It can be modified when creating a [FuncButton] to set a default value.
	Value any
}

func (fb *FuncButton) WidgetValue() any {
	if !fb.reflectFunc.IsValid() {
		return nil
	}
	return fb.reflectFunc.Interface()
}

func (fb *FuncButton) SetWidgetValue(value any) error {
	fb.SetFunc(reflectx.Underlying(reflect.ValueOf(value)).Interface())
	return nil
}

func (fb *FuncButton) OnBind(value any, tags reflect.StructTag) {
	// If someone is viewing a function value, there is a good chance
	// that it is not added to types (and that is out of their control)
	// (eg: in the inspector), so we do not warn on unadded functions.
	fb.SetWarnUnadded(false).SetType(ButtonTonal)
}

func (fb *FuncButton) Init() {
	fb.Button.Init()
	fb.WarnUnadded = true
	fb.Styler(func(s *styles.Style) {
		// If Disabled, these steps are unnecessary and we want the default NotAllowed cursor, so only check for ReadOnly.
		if s.Is(states.ReadOnly) {
			s.SetAbilities(false, abilities.Hoverable, abilities.Clickable, abilities.Activatable)
			s.Cursor = cursors.None
		}
	})
	fb.OnClick(func(e events.Event) {
		if !fb.IsReadOnly() {
			fb.CallFunc()
		}
	})
}

// SetText sets the [FuncButton.Text] and updates the tooltip to
// correspond to the new name.
func (fb *FuncButton) SetText(v string) *FuncButton {
	ptext := fb.Text
	fb.Text = v
	if fb.typesFunc != nil && fb.Text != ptext && ptext != "" {
		fb.typesFunc.Doc = types.FormatDoc(fb.typesFunc.Doc, ptext, fb.Text)
		fb.SetTooltip(fb.typesFunc.Doc)
	}
	return fb
}

// SetFunc sets the function associated with the FuncButton to the
// given function or method value. For documentation information for
// the function to be obtained, it must be added to [types].
func (fb *FuncButton) SetFunc(fun any) *FuncButton {
	fnm := types.FuncName(fun)
	if fnm == "" {
		return fb.SetText("None")
	}
	fnm = strings.ReplaceAll(fnm, "[...]", "") // remove any labeling for generics
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
		gtyp := types.TypeByName(typnm)
		var met *types.Method
		if gtyp == nil {
			if fb.WarnUnadded {
				slog.Warn("core.FuncButton.SetFunc called with a method whose receiver type has not been added to types", "function", fnm)
			}
			met = &types.Method{Name: metnm}
		} else {
			for _, m := range gtyp.Methods {
				if m.Name == metnm {
					met = &m
					break
				}
			}
			if met == nil {
				if fb.WarnUnadded {
					slog.Warn("core.FuncButton.SetFunc called with a method that has not been added to types (even though the receiver type was, you still need to add the method itself)", "function", fnm)
				}
				met = &types.Method{Name: metnm}
			}
		}
		return fb.setMethodImpl(met, reflect.ValueOf(fun))
	}

	if isAnonymousFunction(fnm) {
		f := &types.Func{Name: fnm, Doc: "Anonymous function " + fnm}
		return fb.setFuncImpl(f, reflect.ValueOf(fun))
	}

	f := types.FuncByName(fnm)
	if f == nil {
		if fb.WarnUnadded {
			slog.Warn("core.FuncButton.SetFunc called with a function that has not been added to types", "function", fnm)
		}
		f = &types.Func{Name: fnm}
	}
	return fb.setFuncImpl(f, reflect.ValueOf(fun))
}

func isAnonymousFunction(fnm string) bool {
	// FuncName.funcN indicates that a function was defined anonymously
	funcN := len(fnm) > 0 && unicode.IsDigit(rune(fnm[len(fnm)-1])) && strings.Contains(fnm, ".func")
	return funcN || fnm == "reflect.makeFuncStub" // used for anonymous functions in yaegi
}

// setFuncImpl is the underlying implementation of [FuncButton.SetFunc].
// It should typically not be used by end-user code.
func (fb *FuncButton) setFuncImpl(gfun *types.Func, rfun reflect.Value) *FuncButton {
	fb.typesFunc = gfun
	fb.reflectFunc = rfun
	fb.setArgs()
	fb.setReturns()
	snm := fb.typesFunc.Name
	// get name without package
	li := strings.LastIndex(snm, ".")
	isAnonymous := isAnonymousFunction(snm)
	if snm == "reflect.makeFuncStub" { // used for anonymous functions in yaegi
		snm = "Anonymous function"
		li = -1
	} else if isAnonymous {
		snm = strings.TrimRightFunc(snm, func(r rune) bool {
			return unicode.IsDigit(r) || r == '.'
		})
		snm = strings.TrimSuffix(snm, ".func")
		// we cut at the second to last period (we want to keep the
		// receiver / package for anonymous functions)
		li = strings.LastIndex(snm[:strings.LastIndex(snm, ".")], ".")
	}
	if li >= 0 {
		snm = snm[li+1:] // must also get rid of "."
		// if we are a global function, we may have gone too far with the second to last period,
		// so we go after the last slash if there still is one
		if strings.Contains(snm, "/") {
			snm = snm[strings.LastIndex(snm, "/")+1:]
		}
	}
	snm = strings.Map(func(r rune) rune {
		if r == '(' || r == ')' || r == '*' {
			return -1
		}
		return r
	}, snm)
	txt := strcase.ToSentence(snm)
	fb.SetText(txt)
	// doc formatting interferes with anonymous functions
	if !isAnonymous {
		fb.typesFunc.Doc = types.FormatDoc(fb.typesFunc.Doc, snm, txt)
	}
	fb.SetTooltip(fb.typesFunc.Doc)
	return fb
}

func (fb *FuncButton) goodContext() Widget {
	ctx := fb.Context
	if fb.Context == nil {
		if fb.This == nil {
			return nil
		}
		ctx = fb.This.(Widget)
	}
	return ctx
}

func (fb *FuncButton) callFuncShowReturns() {
	if fb.AfterFunc != nil {
		defer fb.AfterFunc()
	}
	if len(fb.Args) == 0 {
		rets := fb.reflectFunc.Call(nil)
		fb.showReturnsDialog(rets)
		return
	}
	rargs := make([]reflect.Value, len(fb.Args))
	for i, arg := range fb.Args {
		rargs[i] = reflect.ValueOf(arg.Value)
	}
	rets := fb.reflectFunc.Call(rargs)
	fb.showReturnsDialog(rets)
}

// confirmDialog runs the confirm dialog.
func (fb *FuncButton) confirmDialog() {
	ctx := fb.goodContext()
	d := NewBody(fb.Text + "?")
	NewText(d).SetType(TextSupporting).SetText("Are you sure you want to " + strings.ToLower(fb.Text) + "? " + fb.Tooltip)
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText(fb.Text).OnClick(func(e events.Event) {
			fb.callFuncShowReturns()
		})
	})
	d.RunDialog(ctx)
}

// CallFunc calls the function associated with this button,
// prompting the user for any arguments.
func (fb *FuncButton) CallFunc() {
	if !fb.reflectFunc.IsValid() {
		return
	}
	ctx := fb.goodContext()
	if len(fb.Args) == 0 {
		if !fb.Confirm {
			fb.callFuncShowReturns()
			return
		}
		fb.confirmDialog()
		return
	}
	d := NewBody(fb.Text)
	NewText(d).SetType(TextSupporting).SetText(fb.Tooltip)
	str := funcArgsToStruct(fb.Args)
	sv := NewForm(d).SetStruct(str.Addr().Interface())

	accept := func() {
		for i := range fb.Args {
			fb.Args[i].Value = str.Field(i).Interface()
		}
		fb.callFuncShowReturns()
	}

	// If there is a single value button, automatically
	// open its dialog instead of this one
	if len(fb.Args) == 1 {
		curWin := currentRenderWindow
		sv.UpdateWidget() // need to update first
		bt := AsButton(sv.Child(1))
		if bt != nil {
			bt.OnFinal(events.Change, func(e events.Event) {
				// the dialog for the argument has been accepted, so we call the function
				async := false
				if !tree.IsNil(ctx) && currentRenderWindow != curWin { // calling from another window, must lock
					async = true
					ctx.AsWidget().AsyncLock()
				}
				accept()
				if async {
					ctx.AsWidget().AsyncUnlock()
				}
			})
			bt.Scene = fb.Scene // we must use this scene for context
			bt.Send(events.Click)
			return
		}
	}

	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText(fb.Text).OnClick(func(e events.Event) {
			d.Close() // note: the other Close event happens too late!
			accept()
		})
	})
	d.RunDialog(ctx)
}

// funcArgsToStruct converts a slice of [FuncArg] objects
// to a new non-pointer struct [reflect.Value].
func funcArgsToStruct(args []FuncArg) reflect.Value {
	fields := make([]reflect.StructField, len(args))
	for i, arg := range args {
		fields[i] = reflect.StructField{
			Name: strcase.ToCamel(arg.Name),
			Type: reflect.TypeOf(arg.Value),
			Tag:  arg.Tag,
		}
	}
	typ := reflect.StructOf(fields)
	value := reflect.New(typ).Elem()
	for i, arg := range args {
		value.Field(i).Set(reflect.ValueOf(arg.Value))
	}
	return value
}

// setMethodImpl is the underlying implementation of [FuncButton.SetFunc] for methods.
// It should typically not be used by end-user code.
func (fb *FuncButton) setMethodImpl(gmet *types.Method, rmet reflect.Value) *FuncButton {
	return fb.setFuncImpl(&types.Func{
		Name:       gmet.Name,
		Doc:        gmet.Doc,
		Directives: gmet.Directives,
		Args:       gmet.Args,
		Returns:    gmet.Returns,
	}, rmet)
}

// showReturnsDialog runs a dialog displaying the given function return
// values for the function associated with the function button. It does
// nothing if [FuncButton.ShowReturn] is false.
func (fb *FuncButton) showReturnsDialog(rets []reflect.Value) {
	if !fb.ShowReturn {
		for _, ret := range rets {
			if err, ok := ret.Interface().(error); ok && err != nil {
				ErrorSnackbar(fb, err, fb.Text+" failed")
				return
			}
		}
		return
	}
	ctx := fb.goodContext()
	if ctx == nil {
		return
	}
	for i, ret := range rets {
		fb.Returns[i].Value = ret.Interface()
	}
	main := "Result of " + fb.Text
	if len(rets) == 0 {
		main = fb.Text + " succeeded"
	}
	if !fb.ShowReturnAsDialog {
		txt := main
		if len(fb.Returns) > 0 {
			txt += ": "
			for i, ret := range fb.Returns {
				txt += reflectx.ToString(ret.Value)
				if i < len(fb.Returns)-1 {
					txt += ", "
				}
			}
		}
		MessageSnackbar(ctx, txt)
		return
	}

	d := NewBody(main)
	NewText(d).SetType(TextSupporting).SetText(fb.Tooltip)
	d.AddOKOnly()
	str := funcArgsToStruct(fb.Returns)
	sv := NewForm(d).SetStruct(str.Addr().Interface()).SetReadOnly(true)

	// If there is a single value button, automatically
	// open its dialog instead of this one
	if len(fb.Returns) == 1 {
		sv.UpdateWidget() // need to update first
		bt := AsButton(sv.Child(1))
		if bt != nil {
			bt.Scene = fb.Scene // we must use this scene for context
			bt.Send(events.Click)
			return
		}
	}

	if fb.NewWindow {
		d.RunWindowDialog(ctx)
	} else {
		d.RunDialog(ctx)
	}
}

// setArgs sets the appropriate [Value] objects for the
// arguments of the function associated with the function button.
func (fb *FuncButton) setArgs() {
	narg := fb.reflectFunc.Type().NumIn()
	fb.Args = make([]FuncArg, narg)
	for i := range fb.Args {
		typ := fb.reflectFunc.Type().In(i)

		name := ""
		if fb.typesFunc.Args != nil && len(fb.typesFunc.Args) > i {
			name = fb.typesFunc.Args[i]
		} else {
			name = reflectx.NonPointerType(typ).Name()
		}

		fb.Args[i] = FuncArg{
			Name:  name,
			Value: reflect.New(typ).Elem().Interface(),
		}
	}
}

// setReturns sets the appropriate [Value] objects for the
// return values of the function associated with the function
// button.
func (fb *FuncButton) setReturns() {
	nret := fb.reflectFunc.Type().NumOut()
	fb.Returns = make([]FuncArg, nret)
	hasComplex := false
	for i := range fb.Returns {
		typ := fb.reflectFunc.Type().Out(i)
		if !hasComplex {
			k := typ.Kind()
			if k == reflect.Pointer || k == reflect.Struct || k == reflect.Slice || k == reflect.Array || k == reflect.Map {
				hasComplex = true
			}
		}

		name := ""
		if fb.typesFunc.Returns != nil && len(fb.typesFunc.Returns) > i {
			name = fb.typesFunc.Returns[i]
		} else {
			name = reflectx.NonPointerType(typ).Name()
		}

		fb.Returns[i] = FuncArg{
			Name:  name,
			Value: reflect.New(typ).Elem().Interface(),
		}
	}
	if nret > 1 || hasComplex {
		fb.ShowReturnAsDialog = true
	}
}

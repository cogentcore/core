// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/glop/sentencecase"
	"goki.dev/grease"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
)

// FuncConfig contains the configuration options for a function, to be passed
// to [CallFunc] or to the `gi:toolbar` and `gi:menubar` comment directives.
// These options control both the appearance and behavior of both the function
// button in a toolbar and/or menubar button and the dialog created by [CallFunc].
//
//gti:add
type FuncConfig struct {
	// Name is the actual name in code of the function.
	Name string
	// Label is the user-friendly label for the function button.
	// It defaults to the sentence case version of the
	// name of the function.
	Label string
	// Icon is the icon for the function button. If there
	// is an icon with the same name as the function, it
	// defaults to that icon.
	Icon icons.Icon
	// Doc is the documentation for the function, used as
	// a tooltip on the function button and a label in the
	// [CallFunc] dialog. It defaults to the documentation
	// for the function found in gti.
	Doc string
	// SepBefore is whether to insert a separator before the
	// function button in a toolbar/menubar.
	SepBefore bool
	// SepAfter is whether to insert a separator after the
	// function button in a toolbar/menubar.
	SepAfter bool
	// ShowResult is whether to display the result (return values) of the function
	// after it is called. If this is set to true and there are no return values,
	// it displays a message that the method was successful.
	ShowResult bool

	// Args are the arguments to the function. They are set automatically.
	Args *gti.Fields
	// Returns are the return values of the function. They are set automatically.
	Returns *gti.Fields
}

// ToolbarView adds the method buttons for the given value to the given toolbar.
// It returns whether any method buttons were added.
func ToolbarView(val any, tb *gi.Toolbar) bool {
	typ := gti.TypeByValue(val)
	if typ == nil {
		return false
	}
	gotAny := false
	for _, kv := range typ.Methods.Order {
		met := kv.Val
		var tbDir *gti.Directive
		for _, dir := range met.Directives {
			if dir.Tool == "gi" && dir.Directive == "toolbar" {
				tbDir = dir
				break
			}
		}
		if tbDir == nil {
			continue
		}
		cfg := &FuncConfig{
			Name:    met.Name,
			Label:   sentencecase.Of(met.Name),
			Doc:     met.Doc,
			Args:    met.Args,
			Returns: met.Returns,
		}
		// we default to the icon with the same name as
		// the method, if it exists
		ic := icons.Icon(strcase.ToSnake(met.Name))
		if ic.IsValid() {
			cfg.Icon = ic
		}
		_, err := grease.SetFromArgs(cfg, tbDir.Args, grease.ErrNotFound)
		if err != nil {
			slog.Error("programmer error: error while parsing args to `gi:toolbar` comment directive", "err", err.Error())
			continue
		}
		gotAny = true
		if cfg.SepBefore {
			tb.AddSeparator()
		}
		tb.AddButton(gi.ActOpts{Label: cfg.Label, Icon: cfg.Icon, Tooltip: cfg.Doc}, func(bt *gi.Button) {
			fmt.Println("calling method", met.Name)
			rfun := reflect.ValueOf(val).MethodByName(met.Name)
			CallReflectFunc(tb, rfun, cfg)
		})
		if cfg.SepAfter {
			tb.AddSeparator()
		}
	}
	return gotAny
}

// ArgConfig contains the relevant configuration information for each arg,
// including the reflect.Value, name, optional description, and default value
type ArgConfig struct {
	// Name is the actual name of the arg in code
	Name string
	// Label is the user-friendly label name for the arg.
	// It defaults to the sentence case version of Name.
	Label string
	// Doc is the documentation for the argument
	Doc string
	// Val is the reflect.Value of the argument
	Val reflect.Value
	// View is the [Value] view associated with the argument
	View Value
	// Default, if non-nil, is the default value for the argument
	Default any
}

// CallFunc calls the given function with the given configuration information
// in the context of the given widget. It displays a GUI view for selecting any
// unspecified arguments to the function, and optionally a GUI view for the results
// of the function, if [MethodConfig.ShowResult] is on.
//
//gopy:interface=handle
func CallFunc(ctx gi.Widget, fun any, cfg *FuncConfig) {
	rfun := reflect.ValueOf(fun)
	CallReflectFunc(ctx, rfun, cfg)
}

// CallReflectFunc is the same as [CallFunc], but it takes a [reflect.Value] for
// the function instead of an `any`
func CallReflectFunc(ctx gi.Widget, rfun reflect.Value, cfg *FuncConfig) {
	if cfg.Args.Len() == 0 {
		rets := rfun.Call(nil)
		if !cfg.ShowResult {
			return
		}
		ReturnsDialog(ctx, rets, cfg).Run()
		return
	}
	args := ArgsForFunc(rfun, cfg)
	ArgViewDialog(
		ctx,
		DlgOpts{Title: cfg.Label, Prompt: cfg.Doc, Ok: true, Cancel: true},
		args,
		func(dlg *gi.Dialog) {
			rargs := make([]reflect.Value, len(args))
			for i, arg := range args {
				rargs[i] = laser.NonPtrValue(arg.Val)
			}
			rets := rfun.Call(rargs)
			if !cfg.ShowResult {
				return
			}
			ReturnsDialog(ctx, rets, cfg).Run()
		},
	).Run()
}

// ArgsForFunc returns the appropriate [ArgConfig] objects for the arguments
// of the given function with the given configuration information.
func ArgsForFunc(fun reflect.Value, cfg *FuncConfig) []ArgConfig {
	res := make([]ArgConfig, cfg.Args.Len())
	for i, kv := range cfg.Args.Order {
		arg := kv.Val
		ra := ArgConfig{
			Name:  arg.Name,
			Label: sentencecase.Of(arg.Name),
			Doc:   arg.Doc,
		}

		atyp := fun.Type().In(i)
		ra.Val = reflect.New(atyp)

		ra.View = ToValue(ra.Val.Interface(), "")
		ra.View.SetSoloValue(ra.Val)
		ra.View.SetName(ra.Name)
		res[i] = ra
	}
	return res
}

// ReturnsForFunc returns the appropriate [ArgConfig] objects for the given
// return values from the function with the given configuration information.
func ReturnsForFunc(rets []reflect.Value, cfg *FuncConfig) []ArgConfig {
	res := make([]ArgConfig, cfg.Returns.Len())
	for i, kv := range cfg.Returns.Order {
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

// ReturnsDialog returns a dialog displaying the given function return
// values based on the given configuration information and context widget.
func ReturnsDialog(ctx gi.Widget, rets []reflect.Value, cfg *FuncConfig) *gi.Dialog {
	if len(rets) == 0 {
		return gi.NewStdDialog(ctx, gi.DlgOpts{Title: cfg.Label + " succeeded", Prompt: cfg.Doc, Ok: true}, nil)
	}
	ac := ReturnsForFunc(rets, cfg)
	return ArgViewDialog(ctx, DlgOpts{Title: "Result of " + cfg.Label, Prompt: cfg.Doc, Ok: true}, ac, nil)
}

// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
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
	// Confirm is whether to show a confirmation dialog asking
	// the user whether they are sure they want to call the function
	// before calling it.
	Confirm bool
	// ShowResult is whether to display the result (return values) of the function
	// after it is called. If this is set to true and there are no return values,
	// it displays a message that the method was successful.
	ShowResult bool
	// UpdateMethod, when specified on a method, is the name of a method on the same
	// type this method is on to call with the function button whenever it is updated.
	// See [FuncConfig.UpdateFunc] for more information.
	UpdateMethod string

	// UpdateFunc is a function to call with the function button whenever it
	// is updated. For example, this can be used to change whether a button is
	// disabled based on some other value. When using comment directives, this
	// should be set through [FuncConfig.UpdateMethod].
	UpdateFunc func(bt *gi.Button)

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
		cfg := ConfigForMethod(met, "toolbar")
		if cfg == nil { // not in toolbar
			continue
		}
		gotAny = true
		if cfg.SepBefore {
			tb.AddSeparator()
		}
		tb.AddButton(gi.ActOpts{Label: cfg.Label, Icon: cfg.Icon, Tooltip: cfg.Doc}, func(bt *gi.Button) {
			rfun := reflect.ValueOf(val).MethodByName(met.Name)
			CallReflectFunc(bt, rfun, cfg)
		})
		if cfg.SepAfter {
			tb.AddSeparator()
		}
	}
	return gotAny
}

// ConfigForFunc returns the default [FuncConfig] for the given [gti.Func].
// It is a wrapper on [ConfigForMethod]; see it for more information.
func ConfigForFunc(fun *gti.Func) *FuncConfig {
	return ConfigForMethod(&gti.Method{
		Name:       fun.Name,
		Doc:        fun.Doc,
		Directives: fun.Directives,
		Args:       fun.Args,
		Returns:    fun.Returns,
	})
}

// ConfigForMethod returns the default [FuncConfig] for the given [gti.Method].
// If a directive is passed, it indicates what comment directive is allowed to
// specify the configuration information and indicate that the function should
// be included. For example, if "toolbar" is passed, then a function not decorated
// with the directive "gi:toolbar" will result in a nil return value, and otherwise,
// the configuration information will be read from that directive. If no directive
// is passed, it defaults to "func", and is not required (so something can be undecorated
// and it will still return the config object, just without reading it from any directive).
// This means that passing an explicit "func" is different because it makes it required.
func ConfigForMethod(met *gti.Method, directive ...string) *FuncConfig {
	var dir *gti.Directive
	want := "func"
	if len(directive) > 0 {
		want = directive[0]
	}
	for _, d := range met.Directives {
		if d.Tool == "gi" && d.Directive == want {
			dir = d
			break
		}
	}
	// mandatory if specified
	if dir == nil && len(directive) > 0 {
		return nil
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
	if dir != nil {
		_, err := grease.SetFromArgs(cfg, dir.Args, grease.ErrNotFound)
		if err != nil {
			slog.Error(`programmer error: error while parsing args to gi function comment directive`, "directive", dir, "err", err)
			return nil
		}
	}
	return cfg
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
		if !cfg.Confirm {
			rets := rfun.Call(nil)
			if !cfg.ShowResult {
				return
			}
			ReturnsDialog(ctx, rets, cfg).Run()
			return
		}
		gi.NewStdDialog(ctx, gi.DlgOpts{Title: cfg.Label + "?", Prompt: "Are you sure you want to run " + cfg.Label + "? " + cfg.Doc, Ok: true, Cancel: true},
			func(dlg *gi.Dialog) {
				if !dlg.Accepted {
					return
				}
				rets := rfun.Call(nil)
				if !cfg.ShowResult {
					return
				}
				ReturnsDialog(ctx, rets, cfg).Run()
			}).Run()
		return
	}
	args := ArgsForFunc(rfun, cfg)
	ArgViewDialog(
		ctx,
		DlgOpts{Title: cfg.Label, Prompt: cfg.Doc, Ok: true, Cancel: true},
		args,
		func(dlg *gi.Dialog) {
			if !dlg.Accepted {
				return
			}
			rargs := make([]reflect.Value, len(args))
			for i, arg := range args {
				rargs[i] = laser.NonPtrValue(arg.Val)
			}

			if !cfg.Confirm {
				rets := rfun.Call(rargs)
				if !cfg.ShowResult {
					return
				}
				ReturnsDialog(ctx, rets, cfg).Run()
			}
			gi.NewStdDialog(ctx, gi.DlgOpts{Title: cfg.Label + "?", Prompt: "Are you sure you want to run " + cfg.Label + "? " + cfg.Doc, Ok: true, Cancel: true},
				func(dlg *gi.Dialog) {
					if !dlg.Accepted {
						return
					}
					rets := rfun.Call(rargs)
					if !cfg.ShowResult {
						return
					}
					ReturnsDialog(ctx, rets, cfg).Run()
				}).Run()
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

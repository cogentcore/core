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
	"goki.dev/glop/sentencecase"
	"goki.dev/grease"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
)

// MethodConfig contains the configuration options for a method button in a toolbar or menubar.
// These are the configuration options passed to the `gi:toolbar` and `gi:menubar` comment directives.
//
//gti:add
type MethodConfig struct {
	// Name is the actual name in code of the function to call.
	Name string
	// Label is the label for the method button.
	// It defaults to the sentence case version of the
	// name of the function.
	Label string
	// Icon is the icon for the method button. If there
	// is an icon with the same name as the function, it
	// defaults to that icon.
	Icon icons.Icon
	// Tooltip is the tooltip for the method button.
	// It defaults to the documentation for the function.
	Tooltip string
	// SepBefore is whether to insert a separator before the method button.
	SepBefore bool
	// SepAfter is whether to insert a separator after the method button.
	SepAfter bool
	// Args are the arguments to the method
	Args *gti.Fields
	// Returns are the return values of the method
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
		cfg := &MethodConfig{
			Name:    met.Name,
			Label:   sentencecase.Of(met.Name),
			Tooltip: met.Doc,
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
		tb.AddButton(gi.ActOpts{Label: cfg.Label, Icon: cfg.Icon, Tooltip: cfg.Tooltip}, func(bt *gi.Button) {
			fmt.Println("calling method", met.Name)
			CallMethod(tb, val, cfg)
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

// CallMethod calls the method with the given configuration information on the
// given object value, using a GUI interface to prompt for any args. It uses the
// given context widget for context information for the GUI interface.
// gopy:interface=handle
func CallMethod(ctx gi.Widget, val any, met *MethodConfig) {
	rval := reflect.ValueOf(val)
	rmet := rval.MethodByName(met.Name)

	if met.Args.Len() == 0 {
		rmet.Call(nil)
		return
	}
	args := ArgConfigsFromMethod(val, met, rmet)
	ArgViewDialog(
		ctx,
		DlgOpts{Title: met.Label, Prompt: met.Tooltip, Ok: true, Cancel: true},
		args, func(dlg *gi.Dialog) {
			rargs := make([]reflect.Value, len(args))
			for i, arg := range args {
				rargs[i] = laser.NonPtrValue(arg.Val)
			}
			rmet.Call(rargs)
		}).Run()
}

// ArgConfigsFromMethod returns the appropriate [ArgConfig] objects for the given
// method on the given value. It also takes the method as a [reflect.Value].
func ArgConfigsFromMethod(val any, met *MethodConfig, rmet reflect.Value) []ArgConfig {
	res := make([]ArgConfig, met.Args.Len())
	for i, kv := range met.Args.Order {
		arg := kv.Val
		ra := ArgConfig{
			Name:  arg.Name,
			Label: sentencecase.Of(arg.Name),
			Doc:   arg.Doc,
		}

		atyp := rmet.Type().In(i)
		ra.Val = reflect.New(atyp)

		ra.View = ToValue(ra.Val.Interface(), "")
		ra.View.SetSoloValue(ra.Val)
		ra.View.SetName(ra.Name)
		res[i] = ra
	}
	return res
}

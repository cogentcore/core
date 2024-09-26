// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
)

// Func represents a registered tensor function, which has
// In number of input Tensor arguments, and Out number of output
// arguments (typically 1). There can also be an 'any' first
// argument to support other kinds of parameters.
// This is used to make tensor functions available to the Goal language.
type Func struct {
	// Name is the original CamelCase Go name for function
	Name string

	// Fun is the function, which must _only_ take some number of Tensor
	// args, with an optional any first arg.
	Fun any

	// In is number of input tensor args
	In int

	// Out is number of output tensor args
	Out int
}

// NewFunc creates a new Func desciption of the given
// function, which must have a signature like this:
// func([opt any,] a, b, out tensor.Tensor) error
// i.e., taking some specific number of Tensor arguments (up to 5).
// Functions can also take an 'any' first argument to handle other
// non-tensor inputs (e.g., function pointer, dirfs directory, etc).
// The name should be a standard 'package.FuncName' qualified, exported
// CamelCase name, with 'out' indicating the number of output arguments,
// and an optional arg indicating an 'any' first argument.
// The remaining arguments in the function (automatically
// determined) are classified as input arguments.
func NewFunc(name string, fun any) (*Func, error) {
	fn := &Func{Name: name, Fun: fun}
	// fn.In = 1 - out
	// todo: get signature
	return fn, nil
}

// Funcs is the global tensor named function registry.
// All functions must have a signature like this:
// func([opt any,] a, b, out tensor.Tensor) error
// i.e., taking some specific number of Tensor arguments (up to 5),
// with the number of output vs. input arguments registered.
// Functions can also take an 'any' first argument to handle other
// non-tensor inputs (e.g., function pointer, dirfs directory, etc).
// This is used to make tensor functions available to the Goal
// language.
var Funcs map[string]*Func

// AddFunc adds given named function to the global tensor named function
// registry, which is used by Goal to call functions by name.
// See [NewFunc] for more information.
func AddFunc(name string, fun any) error {
	if Funcs == nil {
		Funcs = make(map[string]*Func)
	}
	_, ok := Funcs[name]
	if ok {
		return errors.Log(fmt.Errorf("tensor.AddFunc: function of name %q already exists, not added", name))
	}
	fn, err := NewFunc(name, fun)
	if errors.Log(err) != nil {
		return err
	}
	Funcs[name] = fn
	// note: can record orig camel name if needed for docs etc later.
	return nil
}

// FuncByName finds function of given name in the registry,
// returning an error if the function name has not been registered.
func FuncByName(name string) (*Func, error) {
	fn, ok := Funcs[name]
	if !ok {
		return nil, fmt.Errorf("tensor.FuncByName: function of name %q not registered", name)
	}
	return fn, nil
}

// todo: definitely switch over to reflection here:

// ArgCount returns the number of tensor arguments the function takes,
// using a type switch.
func (fn *Func) ArgCount() int {
	nargs := -1
	switch fn.Fun.(type) {
	case func(a Tensor) error:
		nargs = 1
	case func(a, b Tensor) error:
		nargs = 2
	case func(a, b, c Tensor) error:
		nargs = 3
	case func(a, b, c, d Tensor) error:
		nargs = 4
	case func(a, b, c, d, e Tensor) error:
		nargs = 5
	// any cases:
	case func(first any, a Tensor) error:
		nargs = 1
	case func(first any, a, b Tensor) error:
		nargs = 2
	case func(first any, a, b, c Tensor) error:
		nargs = 3
	case func(first any, a, b, c, d Tensor) error:
		nargs = 4
	case func(first any, a, b, c, d, e Tensor) error:
		nargs = 5
	}
	return nargs
}

// ArgCheck returns an error if the number of args in list does not
// match the number required as specified.
func (fn *Func) ArgCheck(n int, tsr ...Tensor) error {
	if len(tsr) != n {
		return fmt.Errorf("tensor.Call: args passed to %q: %d does not match required: %d", fn.Name, len(tsr), n)
	}
	return nil
}

// These generic functions provide a one liner for wrapping functions
// that take an output Tensor as the last argument, which is important
// for memory re-use of the output in performance-critical cases.
// The names indicate the number of input tensor arguments.
// Additional generic non-Tensor inputs are supported up to 2,
// with Gen1 and Gen2 versions.

// CallOut1 adds output [Values] tensor for function.
func CallOut1(fun func(a Tensor, out Values) error, a Tensor) Values {
	out := NewOfType(a.DataType())
	errors.Log(fun(a, out))
	return out
}

func CallOut2(fun func(a, b Tensor, out Values) error, a, b Tensor) Values {
	out := NewOfType(a.DataType())
	errors.Log(fun(a, b, out))
	return out
}

func CallOut3(fun func(a, b, c Tensor, out Values) error, a, b, c Tensor) Values {
	out := NewOfType(a.DataType())
	errors.Log(fun(a, b, c, out))
	return out
}

func CallOut2Bool(fun func(a, b Tensor, out *Bool) error, a, b Tensor) *Bool {
	out := NewBool()
	errors.Log(fun(a, b, out))
	return out
}

func CallOut1Gen1[T any](fun func(g T, a Tensor, out Values) error, g T, a Tensor) Values {
	out := NewOfType(a.DataType())
	errors.Log(fun(g, a, out))
	return out
}

func CallOut1Gen2[T any, S any](fun func(g T, h S, a Tensor, out Values) error, g T, h S, a Tensor) Values {
	out := NewOfType(a.DataType())
	errors.Log(fun(g, h, a, out))
	return out
}

func CallOut2Gen1[T any](fun func(g T, a, b Tensor, out Values) error, g T, a, b Tensor) Values {
	out := NewOfType(a.DataType())
	errors.Log(fun(g, a, b, out))
	return out
}

func CallOut2Gen2[T any, S any](fun func(g T, h S, a, b Tensor, out Values) error, g T, h S, a, b Tensor) Values {
	out := NewOfType(a.DataType())
	errors.Log(fun(g, h, a, b, out))
	return out
}

// SetCalcFunc sets a function to calculate updated value for given tensor,
// storing the function pointer in the Metadata "CalcFunc" key for the tensor.
// Can be called by [Calc] function.
func SetCalcFunc(tsr Tensor, fun func() error) {
	tsr.Metadata().Set("CalcFunc", fun)
}

// Calc calls function set by [SetCalcFunc] to compute an updated value for
// given tensor. Returns an error if func not set, or any error from func itself.
// Function is stored as CalcFunc in Metadata.
func Calc(tsr Tensor) error {
	fun, err := metadata.Get[func() error](*tsr.Metadata(), "CalcFunc")
	if err != nil {
		return err
	}
	return fun()
}

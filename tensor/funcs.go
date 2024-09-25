// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"reflect"
	"slices"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
)

// AnyFirstArg should be used to set AnyFirst functions
const AnyFirstArg = true

// Func represents a registered tensor function, which has
// In number of input Tensor arguments, and Out number of output
// arguments (typically 1). There can also be an 'any' first
// argument to support other kinds of parameters.
// This is used to make tensor functions available to the Goal
// language.
type Func struct {
	// Name is the original CamelCase Go name for function
	Name string

	// Fun is the function, which must _only_ take some number of Tensor
	// args, with an optional any first arg.
	Fun any

	// In is number of input args
	In int

	// Out is number of output args
	Out int

	// AnyFirst indicates if there is an 'any' first argument.
	AnyFirst bool
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
func NewFunc(name string, fun any, out int, anyFirst ...bool) (*Func, error) {
	fn := &Func{Name: name, Fun: fun, Out: out}
	if len(anyFirst) == 1 && anyFirst[0] {
		fn.AnyFirst = true
	}
	nargs := fn.ArgCount()
	if out > nargs {
		return nil, fmt.Errorf("tensor.NewFunc: too many output args for function %q, which takes %d (-1 means function signature is not recognized)", name, nargs)
	}
	fn.In = 1 - out
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
func AddFunc(name string, fun any, out int, anyFirst ...bool) error {
	if Funcs == nil {
		Funcs = make(map[string]*Func)
	}
	_, ok := Funcs[name]
	if ok {
		return errors.Log(fmt.Errorf("tensor.AddFunc: function of name %q already exists, not added", name))
	}
	fn, err := NewFunc(name, fun, out, anyFirst...)
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

// Call calls function with given set of input & output arguments
// appropriate for the given function (error if not).
func (fn *Func) Call(tsr ...Tensor) error {
	if fn.AnyFirst {
		return fmt.Errorf("tensor.Call: function %q: requires a first string argument", fn.Name)
	}
	switch f := fn.Fun.(type) {
	case func(a Tensor) error:
		if err := fn.ArgCheck(1, tsr...); err != nil {
			return err
		}
		return f(tsr[0])
	case func(a, b Tensor) error:
		if err := fn.ArgCheck(2, tsr...); err != nil {
			return err
		}
		return f(tsr[0], tsr[1])
	case func(a, b, c Tensor) error:
		if err := fn.ArgCheck(3, tsr...); err != nil {
			return err
		}
		return f(tsr[0], tsr[1], tsr[2])
	case func(a, b, c, d Tensor) error:
		if err := fn.ArgCheck(4, tsr...); err != nil {
			return err
		}
		return f(tsr[0], tsr[1], tsr[2], tsr[3])
	case func(a, b, c, d, e Tensor) error:
		if err := fn.ArgCheck(5, tsr...); err != nil {
			return err
		}
		return f(tsr[0], tsr[1], tsr[2], tsr[3], tsr[4])
	}
	return nil
}

// CallAny calls function with given set of input & output arguments
// appropriate for the given function (error if not),
// with a first 'any' argument.
func (fn *Func) CallAny(first any, tsr ...Tensor) error {
	if !fn.AnyFirst {
		return fmt.Errorf("tensor.CallAny: function %q: does not take a first 'any' argument", fn.Name)
	}
	switch f := fn.Fun.(type) {
	case func(first any, a Tensor) error:
		if err := fn.ArgCheck(1, tsr...); err != nil {
			return err
		}
		return f(first, tsr[0])
	case func(first any, a, b Tensor) error:
		if err := fn.ArgCheck(2, tsr...); err != nil {
			return err
		}
		return f(first, tsr[0], tsr[1])
	case func(first any, a, b, c Tensor) error:
		if err := fn.ArgCheck(3, tsr...); err != nil {
			return err
		}
		return f(first, tsr[0], tsr[1], tsr[2])
	case func(first any, a, b, c, d Tensor) error:
		if err := fn.ArgCheck(4, tsr...); err != nil {
			return err
		}
		return f(first, tsr[0], tsr[1], tsr[2], tsr[3])
	case func(first any, a, b, c, d, e Tensor) error:
		if err := fn.ArgCheck(5, tsr...); err != nil {
			return err
		}
		return f(first, tsr[0], tsr[1], tsr[2], tsr[3], tsr[4])
	}
	return nil
}

// CallOut is like [Call] but it automatically creates an output
// tensor of the same type as the first input tensor passed,
// and returns the output as return values, along with any error.
func (fn *Func) CallOut(tsr ...Tensor) (Tensor, error) {
	if fn.Out == 0 {
		err := fn.Call(tsr...)
		return nil, err
	}
	typ := reflect.Float64
	if fn.In > 0 {
		typ = tsr[0].DataType()
	}
	out := NewOfType(typ)
	tlist := slices.Clone(tsr)
	tlist = append(tlist, out)
	err := fn.Call(tlist...)
	return out, err
}

// CallOutMulti is like [CallOut] but deals with multiple output tensors.
func (fn *Func) CallOutMulti(tsr ...Tensor) ([]Tensor, error) {
	if fn.Out == 0 {
		err := fn.Call(tsr...)
		return nil, err
	}
	typ := reflect.Float64
	if fn.In > 0 {
		typ = tsr[0].DataType()
	}
	outs := make([]Tensor, fn.Out)
	for i := range outs {
		outs[i] = NewOfType(typ)
	}
	tsr = append(tsr, outs...)
	err := fn.Call(tsr...)
	return outs, err
}

// Call calls function of given name, with given set of _input_
// and output arguments appropriate for the given function.
// An error is logged if the function name has not been registered
// in the Funcs global function registry, or the argument count
// does not match, or an error is returned by the function.
func Call(name string, tsr ...Tensor) error {
	fn, err := FuncByName(name)
	if err != nil {
		return err
	}
	return fn.Call(tsr...)
}

// CallOut calls function of given name, with given set of _input_
// arguments appropriate for the given function, returning a created
// output tensor, for the common case with just one return value.
// An error is logged if the function name has not been registered
// in the Funcs global function registry, or the argument count
// does not match, or an error is returned by the function.
func CallOut(name string, tsr ...Tensor) Tensor {
	fn, err := FuncByName(name)
	if errors.Log(err) != nil {
		return nil
	}
	return errors.Log1(fn.CallOut(tsr...))
}

// CallAny calls function of given name, with given set of arguments
// (any, input and output) appropriate for the given function.
// An error is returned if the function name has not been registered
// in the Funcs global function registry, or the argument count
// does not match.  This version of [Call] is for functions that
// have an initial string argument
func CallAny(name string, first any, tsr ...Tensor) error {
	fn, err := FuncByName(name)
	if err != nil {
		return err
	}
	return fn.CallAny(first, tsr...)
}

// CallOutMulti calls function of given name, with given set of _input_
// arguments appropriate for the given function, returning newly created
// output tensors, for the rare case of multiple return values.
// An error is logged if the function name has not been registered
// in the Funcs global function registry, or the argument count
// does not match, or an error is returned by the function.
func CallOutMulti(name string, tsr ...Tensor) []Tensor {
	fn, err := FuncByName(name)
	if errors.Log(err) != nil {
		return nil
	}
	return errors.Log1(fn.CallOutMulti(tsr...))
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

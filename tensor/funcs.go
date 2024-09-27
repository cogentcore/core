// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"reflect"

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

	// Args has parsed information about the function args, for Goal.
	Args []*Arg
}

// Arg has key information that Goal needs about each arg, for converting
// expressions into the appropriate type.
type Arg struct {
	// Type has full reflection type info.
	Type reflect.Type

	// IsTensor is true if it satisfies the Tensor interface.
	IsTensor bool

	// IsInt is true if Kind = Int, for shape, slice etc params.
	IsInt bool

	// IsVariadic is true if this is the last arg and has ...; type will be an array.
	IsVariadic bool
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
	fn.GetArgs()
	return fn, nil
}

// GetArgs gets key info about each arg, for use by Goal transpiler.
func (fn *Func) GetArgs() {
	ft := reflect.TypeOf(fn.Fun)
	n := ft.NumIn()
	if n == 0 {
		return
	}
	fn.Args = make([]*Arg, n)
	tsrt := reflect.TypeFor[Tensor]()
	for i := range n {
		at := ft.In(i)
		ag := &Arg{Type: at}
		if ft.IsVariadic() && i == n-1 {
			ag.IsVariadic = true
		}
		if at.Kind() == reflect.Int || (at.Kind() == reflect.Slice && at.Elem().Kind() == reflect.Int) {
			ag.IsInt = true
		} else if at.Implements(tsrt) {
			ag.IsTensor = true
		}
		fn.Args[i] = ag
	}
}

func (fn *Func) String() string {
	s := fn.Name + "("
	na := len(fn.Args)
	for i, a := range fn.Args {
		if a.IsVariadic {
			s += "..."
		}
		ts := a.Type.String()
		if ts == "interface {}" {
			ts = "any"
		}
		s += ts
		if i < na-1 {
			s += ", "
		}
	}
	s += ")"
	return s
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
// See [NewFunc] for more informa.tion.
func AddFunc(name string, fun any) error {
	if Funcs == nil {
		Funcs = make(map[string]*Func)
	}
	_, ok := Funcs[name]
	if ok {
		return fmt.Errorf("tensor.AddFunc: function of name %q already exists, not added", name)
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

//////////////////// Metadata

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

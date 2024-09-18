// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"reflect"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
)

// StringFirstArg should be used to set StringFirst functions
const StringFirstArg = true

// Func represents a registered tensor function, which has
// In number of input *tensor.Indexed arguments, and Out
// number of output arguments.  There can also be an optional
// string first argument, which is used to specify the name of
// another function in some cases (e.g., a stat or metric function).
type Func struct {
	// Name is the original CamelCase Go name for function
	Name string

	// Fun is the function, which must _only_ take some number of *tensor.Indexed
	// args, with an optional first string arg per [StringFirst].
	Fun any

	// In is number of input args
	In int

	// Out is number of output args
	Out int

	// StringFirst indicates if there is a string first argument, which can be
	// used for many purposes including passing the name of another function to use
	// in computation.
	StringFirst bool
}

// NewFunc creates a new Func desciption of the given
// function, with specified number of output arguments,
// and an optional first string argument.
// The remaining arguments in the function (automatically
// determined) are classified as input arguments.
func NewFunc(name string, fun any, out int, stringFirst ...bool) (*Func, error) {
	fn := &Func{Name: name, Fun: fun, Out: out}
	if len(stringFirst) == 1 && stringFirst[0] {
		fn.StringFirst = true
	}
	nargs := fn.ArgCount()
	if out > nargs {
		return nil, fmt.Errorf("tensor.NewFunc: too many output args for function %q, which takes %d (-1 means function signature is not recognized)", name, nargs)
	}
	fn.In = 1 - out
	return fn, nil
}

// Funcs is the global tensor named function registry.
// All functions must be of the form: func(a, b *tensor.Indexed)
// i.e., taking some specific number of Indexed arguments,
// with the number of output vs. input arguments registered.
var Funcs map[string]*Func

// AddFunc adds given named function to the global tensor named function
// registry, which is used in goal to call functions by name, and
// in specific packages to call functions by enum String() names.
// Use the standard Go CamelCase name -- will be auto-lowercased.
// The number of output arguments must be provided here,
// along with an optional first string argument if present;
// the number of input arguments is automatically set from that.
func AddFunc(name string, fun any, out int, stringFirst ...bool) error {
	if Funcs == nil {
		Funcs = make(map[string]*Func)
	}
	nm := strings.ToLower(name)
	_, ok := Funcs[nm]
	if ok {
		return errors.Log(fmt.Errorf("tensor.AddFunc: function of name %q already exists, not added", name))
	}
	fn, err := NewFunc(name, fun, out, stringFirst...)
	if errors.Log(err) != nil {
		return err
	}
	Funcs[nm] = fn
	// note: can record orig camel name if needed for docs etc later.
	return nil
}

// Call calls function of given name, with given set of arguments
// (input and output) appropriate for the given function.
// An error is returned if the function name has not been registered
// in the Funcs global function registry, or the argument count
// does not match.
func Call(name string, tsr ...*Indexed) error {
	nm := strings.ToLower(name)
	fn, ok := Funcs[nm]
	if !ok {
		return errors.Log(fmt.Errorf("tensor.Call: function of name %q not registered", name))
	}
	return fn.Call(tsr...)
}

// CallString calls function of given name, with given set of arguments
// (string, input and output) appropriate for the given function.
// An error is returned if the function name has not been registered
// in the Funcs global function registry, or the argument count
// does not match.  This version of [Call] is for functions that
// have an initial string argument
func CallString(name, first string, tsr ...*Indexed) error {
	nm := strings.ToLower(name)
	fn, ok := Funcs[nm]
	if !ok {
		return errors.Log(fmt.Errorf("tensor.Call: function of name %q not registered", name))
	}
	return fn.CallString(first, tsr...)
}

// CallOut calls function of given name, with given set of _input_
// arguments appropriate for the given function, returning newly created
// output tensors.
// An error is logged if the function name has not been registered
// in the Funcs global function registry, or the argument count
// does not match.
func CallOut(name string, tsr ...*Indexed) []*Indexed {
	nm := strings.ToLower(name)
	fn, ok := Funcs[nm]
	if !ok {
		errors.Log(fmt.Errorf("tensor.CallOut: function of name %q not registered", name))
		return nil
	}
	return errors.Log1(fn.CallOut(tsr...))
}

// ArgCount returns the number of tensor arguments the function takes,
// using a type switch.
func (fn *Func) ArgCount() int {
	nargs := -1
	switch fn.Fun.(type) {
	case func(a *Indexed):
		nargs = 1
	case func(a, b *Indexed):
		nargs = 2
	case func(a, b, c *Indexed):
		nargs = 3
	case func(a, b, c, d *Indexed):
		nargs = 4
	case func(a, b, c, d, e *Indexed):
		nargs = 5
	// string cases:
	case func(s string, a *Indexed):
		nargs = 1
	case func(s string, a, b *Indexed):
		nargs = 2
	case func(s string, a, b, c *Indexed):
		nargs = 3
	case func(s string, a, b, c, d *Indexed):
		nargs = 4
	case func(s string, a, b, c, d, e *Indexed):
		nargs = 5
	}
	return nargs
}

// ArgCheck returns an error if the number of args in list does not
// match the number required as specified.
func (fn *Func) ArgCheck(n int, tsr ...*Indexed) error {
	if len(tsr) != n {
		return fmt.Errorf("tensor.Call: args passed to %q: %d does not match required: %d", fn.Name, len(tsr), n)
	}
	return nil
}

// Call calls function with given set of input & output arguments
// appropriate for the given function (error if not).
func (fn *Func) Call(tsr ...*Indexed) error {
	if fn.StringFirst {
		return fmt.Errorf("tensor.Call: function %q: requires a first string argument", fn.Name)
	}
	switch f := fn.Fun.(type) {
	case func(a *Indexed):
		if err := fn.ArgCheck(1, tsr...); err != nil {
			return err
		}
		f(tsr[0])
	case func(a, b *Indexed):
		if err := fn.ArgCheck(2, tsr...); err != nil {
			return err
		}
		f(tsr[0], tsr[1])
	case func(a, b, c *Indexed):
		if err := fn.ArgCheck(3, tsr...); err != nil {
			return err
		}
		f(tsr[0], tsr[1], tsr[2])
	case func(a, b, c, d *Indexed):
		if err := fn.ArgCheck(4, tsr...); err != nil {
			return err
		}
		f(tsr[0], tsr[1], tsr[2], tsr[3])
	case func(a, b, c, d, e *Indexed):
		if err := fn.ArgCheck(5, tsr...); err != nil {
			return err
		}
		f(tsr[0], tsr[1], tsr[2], tsr[3], tsr[4])
	}
	return nil
}

// CallString calls function with given set of input & output arguments
// appropriate for the given function (error if not),
// with an initial string argument.
func (fn *Func) CallString(s string, tsr ...*Indexed) error {
	if !fn.StringFirst {
		return fmt.Errorf("tensor.CallString: function %q: does not take a first string argument", fn.Name)
	}
	switch f := fn.Fun.(type) {
	case func(s string, a *Indexed):
		if err := fn.ArgCheck(1, tsr...); err != nil {
			return err
		}
		f(s, tsr[0])
	case func(s string, a, b *Indexed):
		if err := fn.ArgCheck(2, tsr...); err != nil {
			return err
		}
		f(s, tsr[0], tsr[1])
	case func(s string, a, b, c *Indexed):
		if err := fn.ArgCheck(3, tsr...); err != nil {
			return err
		}
		f(s, tsr[0], tsr[1], tsr[2])
	case func(s string, a, b, c, d *Indexed):
		if err := fn.ArgCheck(4, tsr...); err != nil {
			return err
		}
		f(s, tsr[0], tsr[1], tsr[2], tsr[3])
	case func(s string, a, b, c, d, e *Indexed):
		if err := fn.ArgCheck(5, tsr...); err != nil {
			return err
		}
		f(s, tsr[0], tsr[1], tsr[2], tsr[3], tsr[4])
	}
	return nil
}

// CallOut calls function with given set of _input_ arguments
// appropriate for the given function (error if not).
// Newly-created output values are returned.
func (fn *Func) CallOut(tsr ...*Indexed) ([]*Indexed, error) {
	if fn.Out == 0 {
		err := fn.Call(tsr...)
		return nil, err
	}
	typ := reflect.Float64
	if fn.In > 0 {
		typ = tsr[0].Tensor.DataType()
	}
	outs := make([]*Indexed, fn.Out)
	for i := range outs {
		outs[i] = NewIndexed(NewOfType(typ))
	}
	tsr = append(tsr, outs...)
	err := fn.Call(tsr...)
	return outs, err
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

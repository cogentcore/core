// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

import (
	"fmt"
	"sort"

	"golang.org/x/exp/maps"
)

// Function represents the call graph of functions
type Function struct {
	Name    string
	Funcs   map[string]*Function
	Atomics map[string]*Var // variables that have atomic operations in this function
}

func NewFunction(name string) *Function {
	return &Function{Name: name, Funcs: make(map[string]*Function)}
}

// get or add a function of given name
func (st *State) RecycleFunc(name string) *Function {
	fn, ok := st.FuncGraph[name]
	if !ok {
		fn = NewFunction(name)
		st.FuncGraph[name] = fn
	}
	return fn
}

func getAllFuncs(f *Function, all map[string]*Function) {
	for fnm, fn := range f.Funcs {
		_, ok := all[fnm]
		if ok {
			continue
		}
		all[fnm] = fn
		getAllFuncs(fn, all)
	}
}

// AllFuncs returns aggregated list of all functions called be given function
func (st *State) AllFuncs(name string) map[string]*Function {
	fn, ok := st.FuncGraph[name]
	if !ok {
		fmt.Printf("gosl: ERROR kernel function named: %q not found\n", name)
		return nil
	}
	all := make(map[string]*Function)
	all[name] = fn
	getAllFuncs(fn, all)
	// cfs := maps.Keys(all)
	// sort.Strings(cfs)
	// for _, cfnm := range cfs {
	// 	fmt.Println("\t" + cfnm)
	// }
	return all
}

// AtomicVars returns all the variables marked as atomic
// within the list of functions.
func (st *State) AtomicVars(funcs map[string]*Function) map[string]*Var {
	avars := make(map[string]*Var)
	for _, fn := range funcs {
		if fn.Atomics == nil {
			continue
		}
		for vn, v := range fn.Atomics {
			avars[vn] = v
		}
	}
	return avars
}

func (st *State) PrintFuncGraph() {
	funs := maps.Keys(st.FuncGraph)
	sort.Strings(funs)
	for _, fname := range funs {
		fmt.Println(fname)
		fn := st.FuncGraph[fname]
		cfs := maps.Keys(fn.Funcs)
		sort.Strings(cfs)
		for _, cfnm := range cfs {
			fmt.Println("\t" + cfnm)
		}
	}
}

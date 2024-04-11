// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"sync/atomic"
)

var (
	// Funcs records all types (i.e., a type registry)
	// key is long type name: package_url.Func, e.g., cogentcore.org/core/core.Button
	Funcs = map[string]*Func{}

	// FuncIDCounter is an atomically incremented uint64 used
	// for assigning new [Func.ID] numbers
	FuncIDCounter uint64
)

// FuncByName returns a Func by name (package_url.Type, e.g., cogentcore.org/core/core.Button),
func FuncByName(nm string) *Func {
	fi, ok := Funcs[nm]
	if !ok {
		return nil
	}
	return fi
}

// FuncByNameTry returns a Func by name (package_url.Type, e.g., cogentcore.org/core/core.Button),
// or error if not found
func FuncByNameTry(nm string) (*Func, error) {
	fi, ok := Funcs[nm]
	if !ok {
		return nil, fmt.Errorf("func %q not found", nm)
	}
	return fi, nil
}

// FuncInfo returns function info for given function.
func FuncInfo(f any) *Func {
	return FuncByName(FuncName(f))
}

// FuncInfoTry returns function info for given function.
func FuncInfoTry(f any) (*Func, error) {
	return FuncByNameTry(FuncName(f))
}

// AddFunc adds a constructed [Func] to the registry
// and returns it. This sets the ID.
func AddFunc(fun *Func) *Func {
	if _, has := Funcs[fun.Name]; has {
		slog.Debug("types.AddFunc: Func already exists", "Func.Name", fun.Name)
		return fun
	}
	fun.ID = atomic.AddUint64(&FuncIDCounter, 1)
	Funcs[fun.Name] = fun
	return fun
}

// FuncName returns the fully package-qualified name of given function
// This is guaranteed to be unique and used for the Funcs registry.
func FuncName(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

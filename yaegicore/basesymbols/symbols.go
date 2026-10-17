// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package basesymbols contains yaegi symbols for base (non-GUI) packages.
package basesymbols

//go:generate ./make

import (
	"reflect"
	"sync"
)

// Symbols are the yaegi symbols of the packages, by import path and name.
// They are added by [Load], which is called at init time except on the web
// (see [Load]).
var Symbols = map[string]map[string]reflect.Value{}

var (
	// loaders are the functions that add the symbols of each package
	// to [Symbols], which are registered at init time.
	loaders []func()

	// loadOnce makes [Load] add the symbols once.
	loadOnce sync.Once
)

// register adds the given function that adds symbols to [Symbols]
// to the functions that [Load] calls.
func register(loader func()) bool {
	loaders = append(loaders, loader)
	return true
}

// Load adds the symbols of all packages to [Symbols], once. Natively, it
// is called at init time. On the web, making the symbols at init time is
// a noticeable part of the startup time of an app, so it is instead called
// by yaegicore when it first makes an interpreter; other code using
// [Symbols] on the web must call it first.
func Load() {
	loadOnce.Do(func() {
		for _, loader := range loaders {
			loader()
		}
		loaders = nil
	})
}

func init() {
	if !lazyLoad {
		Load()
	}
}

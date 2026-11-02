// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && !ios

package apps

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework UniformTypeIdentifiers
#include <stdlib.h>
char* appsForFile(const char* path);
char* openWithApp(const char* path, const char* app);
char* setDefaultApp(const char* path, const char* app);
*/
import "C"

import (
	"errors"
	"strings"
	"unsafe"
)

// cString calls the given function with the given strings as C strings,
// returning its C string result as a Go string, freeing them.
func cString(f func(a, b *C.char) *C.char, a, b string) (string, bool) {
	ca, cb := C.CString(a), C.CString(b)
	defer C.free(unsafe.Pointer(ca))
	defer C.free(unsafe.Pointer(cb))
	r := f(ca, cb)
	if r == nil {
		return "", false
	}
	defer C.free(unsafe.Pointer(r))
	return C.GoString(r), true
}

func forFile(file string) ([]App, error) {
	out, _ := cString(func(a, _ *C.char) *C.char { return C.appsForFile(a) }, file, "")
	var as []App
	for ln := range strings.SplitSeq(out, "\n") {
		ln, def := strings.CutPrefix(ln, "*")
		id, name, ok := strings.Cut(ln, "\t")
		if !ok {
			continue
		}
		as = append(as, App{Name: name, ID: id, Default: def})
	}
	return as, nil
}

func openWith(file string, app App) error {
	if msg, ok := cString(func(a, b *C.char) *C.char { return C.openWithApp(a, b) }, file, app.ID); ok {
		return errors.New("apps: could not open " + file + " with " + app.Name + ": " + msg)
	}
	return nil
}

func setDefault(file string, app App) error {
	if msg, ok := cString(func(a, b *C.char) *C.char { return C.setDefaultApp(a, b) }, file, app.ID); ok {
		return errors.New("apps: could not make " + app.Name + " the default app for " + file + ": " + msg)
	}
	return nil
}

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
	"os"
	"path/filepath"
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

// appDirs are the directories with the apps, in which their subdirectories,
// as Utilities, also have apps.
var appDirs = []string{"/Applications", "/System/Applications", "~/Applications"}

func allApps() ([]App, error) {
	var as []App
	seen := map[string]bool{}
	add := func(path string) {
		if seen[path] {
			return
		}
		seen[path] = true
		as = append(as, App{Name: strings.TrimSuffix(filepath.Base(path), ".app"), ID: path})
	}
	for _, dir := range appDirs {
		if rest, ok := strings.CutPrefix(dir, "~/"); ok {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			dir = filepath.Join(home, rest)
		}
		ents, _ := os.ReadDir(dir)
		for _, ent := range ents {
			path := filepath.Join(dir, ent.Name())
			if strings.HasSuffix(ent.Name(), ".app") {
				add(path)
				continue
			}
			if !ent.IsDir() {
				continue
			}
			subs, _ := os.ReadDir(path)
			for _, sub := range subs {
				if strings.HasSuffix(sub.Name(), ".app") {
					add(filepath.Join(path, sub.Name()))
				}
			}
		}
	}
	return as, nil
}

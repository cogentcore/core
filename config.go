// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package fs

import (
	"syscall"
	"syscall/js"
)

// Config configures the given JavaScript object to be a filesystem that implements
// the Node.js fs API. It is the main entry point for code using jsfs. It returns
// the resulting [FS], which should not typically be needed.
func Config(jfs js.Value) (*FS, error) {
	fs, err := NewFS()
	if err != nil {
		return nil, err
	}

	constants := jfs.Get("constants")
	constants.Set("O_RDONLY", syscall.O_RDONLY)
	constants.Set("O_WRONLY", syscall.O_WRONLY)
	constants.Set("O_RDWR", syscall.O_RDWR)
	constants.Set("O_CREAT", syscall.O_CREATE)
	constants.Set("O_TRUNC", syscall.O_TRUNC)
	constants.Set("O_APPEND", syscall.O_APPEND)
	constants.Set("O_EXCL", syscall.O_EXCL)

	SetFunc(jfs, "chmod", fs.Chmod)
	SetFunc(jfs, "chown", fs.Chown)
	SetFunc(jfs, "close", fs.Close)

	return fs, nil
}

// Func is the type of a jsfs function.
type Func func(args []js.Value) (any, error)

// SetFunc sets the function with the given name on the given value to the given function.
func SetFunc(v js.Value, name string, fn Func) {
	f := FuncOf(name, fn)
	v.Set(name, f)
}

// FuncOf turns the given function into a callback [js.Func].
func FuncOf(name string, fn Func) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		callback := args[len(args)-1]
		args = args[:len(args)-1]

		res, err := fn(args)

		errv := JSError(err, name, args...)
		callback.Invoke([]any{errv, res})
		return nil
	})
}

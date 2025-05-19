// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package jsfs

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
		js.Global().Get("console").Call("error", err.Error())
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
	// constants.Set("O_DIRECTORY", syscall.O_DIRECTORY) TODO(go1.24): uncomment

	SetFunc(jfs, "chmod", fs.Chmod)
	SetFunc(jfs, "chown", fs.Chown)
	SetFunc(jfs, "close", fs.Close)
	SetFunc(jfs, "fchmod", fs.Fchmod)
	SetFunc(jfs, "fchown", fs.Fchown)
	SetFunc(jfs, "fstat", fs.Fstat)
	SetFunc(jfs, "fsync", fs.Fsync)
	SetFunc(jfs, "ftruncate", fs.Ftruncate)
	SetFunc(jfs, "lchown", fs.Lchown)
	SetFunc(jfs, "link", fs.Link)
	SetFunc(jfs, "lstat", fs.Lstat)
	SetFunc(jfs, "mkdir", fs.Mkdir)
	SetFunc(jfs, "mkdirall", fs.MkdirAll)
	SetFunc(jfs, "open", fs.Open)
	SetFunc(jfs, "readdir", fs.Readdir)
	SetFunc(jfs, "readlink", fs.Readlink)
	SetFunc(jfs, "rename", fs.Rename)
	SetFunc(jfs, "rmdir", fs.Rmdir)
	SetFunc(jfs, "stat", fs.Stat)
	SetFunc(jfs, "symlink", fs.Symlink)
	SetFunc(jfs, "unlink", fs.Unlink)
	SetFunc(jfs, "utimes", fs.Utimes)
	SetFunc(jfs, "truncate", fs.Truncate)
	SetFunc(jfs, "read", fs.Read)
	SetFunc(jfs, "readfile", fs.ReadFile)
	SetFunc(jfs, "write", fs.Write)

	return fs, err
}

// Func is the type of a jsfs function.
type Func interface {
	func(args []js.Value) (any, error) | func(args []js.Value) (any, any, error)
}

// SetFunc sets the function with the given name on the given value to the given function.
func SetFunc[F Func](v js.Value, name string, fn F) {
	f := FuncOf(name, fn)
	v.Set(name, f)
}

// FuncOf turns the given function into a callback [js.Func].
func FuncOf[F Func](name string, fn F) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		// the callback is always the last argument
		callback := args[len(args)-1]
		args = args[:len(args)-1]

		// we need to wrap the function call in a separate
		// goroutine because these functions are asynchronous
		// and return immediately, calling the callback later
		go func() {
			var res []any
			var err error

			switch fn := any(fn).(type) {
			case func(args []js.Value) (any, error):
				r, e := fn(args)
				res = []any{r}
				err = e
			case func(args []js.Value) (any, any, error):
				r0, r1, e := fn(args)
				res = []any{r0, r1}
				err = e
			}

			errv := JSError(err, name, args...)
			callback.Invoke(append([]any{errv}, res...)...)
		}()
		return nil
	})
}

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

	jfs.Set("chmod", FuncOf(fs.Chmod))

	return fs, nil
}

// FuncOf is a simple wrapper for [js.FuncOf] for functions that do not
// need a this value and return value.
func FuncOf(fn func(args []js.Value)) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		fn(args)
		return nil
	})
}

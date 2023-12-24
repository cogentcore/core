// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package fs

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"syscall"
	"syscall/js"

	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
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

// JsError converts the given error value into a JS error value.
func JSError(err error, message string, args ...js.Value) js.Value {
	if err == nil {
		return js.Null()
	}

	errMessage := errors.Wrap(err, message).Error()
	for _, arg := range args {
		errMessage += fmt.Sprintf("\n%v", arg)
	}

	return js.ValueOf(map[string]interface{}{
		"message": js.ValueOf(errMessage),
		"code":    js.ValueOf(GetErrType(err, errMessage)),
	})
}

// GetErrType returns the JS type of the given error.
func GetErrType(err error, debugMessage string) string {
	if err := errors.Unwrap(err); err != nil {
		return GetErrType(err, debugMessage)
	}
	switch err {
	case io.EOF, exec.ErrNotFound:
		return "ENOENT"
	}
	switch {
	case errors.Is(err, hackpadfs.ErrClosed):
		return "EBADF" // if it was already closed, then the file descriptor was invalid
	case errors.Is(err, hackpadfs.ErrNotExist):
		return "ENOENT"
	case errors.Is(err, hackpadfs.ErrExist):
		return "EEXIST"
	case errors.Is(err, hackpadfs.ErrIsDir):
		return "EISDIR"
	case errors.Is(err, hackpadfs.ErrPermission):
		return "EPERM"
	default:
		log.Printf("error: jsfs.GetErrType: unknown error type: (%T) %+v\n\n%s\n", err, err, debugMessage)
		return "EPERM"
	}
}

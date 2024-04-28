// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package jsfs

import (
	"fmt"
	"io"
	"syscall/js"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

func (f *FS) Write(args []js.Value) (any, any, error) {
	fl, err := f.GetFile(args)
	if err != nil {
		return 0, nil, err
	}
	flw, ok := fl.(io.Writer)
	if !ok {
		return 0, nil, hackpadfs.ErrNotImplemented
	}

	var n int

	buffer := args[1]

	iblob, err := idbblob.New(buffer)
	if err != nil {
		return n, nil, err
	}

	offset := args[2].Int()
	length := args[3].Int()
	position := args[4]

	// 'offset' in Node.js's read is the offset in the buffer to start writing at,
	// and 'position' is where to begin reading from in the file.
	if !position.IsUndefined() && !position.IsNull() {
		_, err := hackpadfs.SeekFile(fl, int64(position.Int()), io.SeekStart)
		if err != nil {
			return n, nil, err
		}
	}
	dataToCopy, err := blob.View(iblob, int64(offset), int64(offset+length))
	if err != nil {
		// TODO: is this the right path?
		return 0, nil, &hackpadfs.PathError{Op: "write", Path: fmt.Sprint(fl), Err: err}
	}
	n, err = blob.Write(flw, dataToCopy)
	if err == io.EOF {
		err = nil
	}
	return n, buffer, err
}

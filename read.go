// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package fs

import (
	"io"
	"syscall/js"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

func (f *FS) Read(args []js.Value) (any, error) { // fd FID, buffer blob.Blob, offset, length int, position *int64) (n int, err error) {
	fl, err := f.GetFile(args)
	if err != nil {
		return nil, err
	}
	// 'offset' in Node.js's read is the offset in the buffer to start writing at,
	// and 'position' is where to begin reading from in the file.
	var readBuf blob.Blob
	var n int

	buffer := args[1]
	offset := args[2].Int()
	length := args[3].Int()
	position := args[4]

	if position.IsUndefined() {
		readBuf, n, err = blob.Read(fl, length)
	} else {
		readerAt, ok := fl.(io.ReaderAt)
		if ok {
			readBuf, n, err = blob.ReadAt(readerAt, length, int64(position.Int()))
		} else {
			err = &hackpadfs.PathError{Op: "read", Path: fl.openedName, Err: hackpadfs.ErrNotImplemented}
		}
	}
	if err == io.EOF {
		err = nil
	}
	if readBuf != nil {
		iblob, ierr := idbblob.New(buffer)
		if ierr != nil {
			return n, ierr
		}
		_, setErr := blob.Set(iblob, readBuf, int64(offset))
		if err == nil && setErr != nil {
			err = &hackpadfs.PathError{Op: "read", Path: fl.openedName, Err: setErr}
		}
	}
	return n, err
}

func (f *FS) ReadFile(args []js.Value) (any, error) {
	fda, err := f.Open([]js.Value{args[0], js.ValueOf(0), js.ValueOf(0)})
	if err != nil {
		return nil, err
	}
	fd := js.ValueOf(fda)
	defer f.Close([]js.Value{fd})

	infoa, err := f.Fstat([]js.Value{fd})
	if err != nil {
		return nil, err
	}
	info := js.ValueOf(infoa)

	buf, err := idbblob.NewLength(info.Get("size").Int())
	if err != nil {
		return nil, err
	}
	_, err = f.Read([]js.Value{fd, buf.JSValue(), js.ValueOf(0), js.ValueOf(buf.Len())})
	return buf, err
}

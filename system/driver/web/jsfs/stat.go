// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package jsfs

import (
	"os"
	"syscall"
	"syscall/js"
)

var (
	FuncTrue = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return true
	})
	FuncFalse = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return false
	})
)

func JSStat(info os.FileInfo) js.Value {
	if info == nil {
		return js.Null()
	}
	const blockSize = 4096 // TODO find useful value for blksize
	modTime := info.ModTime().UnixNano() / 1e6
	return js.ValueOf(map[string]interface{}{
		"dev":     0,
		"ino":     0,
		"mode":    JSMode(info.Mode()),
		"nlink":   1,
		"uid":     0, // TODO use real values for uid and gid
		"gid":     0,
		"rdev":    0,
		"size":    info.Size(),
		"blksize": blockSize,
		"blocks":  BlockCount(info.Size(), blockSize),
		"atimeMs": modTime,
		"mtimeMs": modTime,
		"ctimeMs": modTime,

		"isBlockDevice":     FuncFalse,
		"isCharacterDevice": FuncFalse,
		"isDirectory":       JSBoolFunc(info.IsDir()),
		"isFIFO":            FuncFalse,
		"isFile":            JSBoolFunc(info.Mode().IsRegular()),
		"isSocket":          FuncFalse,
		"isSymbolicLink":    JSBoolFunc(info.Mode()&os.ModeSymlink == os.ModeSymlink),
	})
}

var ModeBitTranslation = map[os.FileMode]uint32{
	os.ModeDir:        syscall.S_IFDIR,
	os.ModeCharDevice: syscall.S_IFCHR,
	os.ModeNamedPipe:  syscall.S_IFIFO,
	os.ModeSymlink:    syscall.S_IFLNK,
	os.ModeSocket:     syscall.S_IFSOCK,
}

func JSMode(mode os.FileMode) uint32 {
	for goBit, jsBit := range ModeBitTranslation {
		if mode&goBit == goBit {
			mode = mode & ^goBit | os.FileMode(jsBit)
		}
	}
	return uint32(mode)
}

func BlockCount(size, blockSize int64) int64 {
	blocks := size / blockSize
	if size%blockSize > 0 {
		return blocks + 1
	}
	return blocks
}

func JSBoolFunc(b bool) js.Func {
	if b {
		return FuncTrue
	}
	return FuncFalse
}

// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transpile

import (
	"errors"
	"fmt"

	"cogentcore.org/core/tensor/datafs"
)

var datafsCommands = map[string]func(mp *mathParse) error{
	"cd":    cd,
	"mkdir": mkdir,
	"ls":    ls,
}

func cd(mp *mathParse) error {
	var dir string
	if len(mp.ewords) > 1 {
		dir = mp.ewords[1]
	}
	return datafs.Chdir(dir)
}

func mkdir(mp *mathParse) error {
	if len(mp.ewords) == 1 {
		return errors.New("datafs mkdir requires a directory name")
	}
	dir := mp.ewords[1]
	_, err := datafs.CurDir.Mkdir(dir)
	return err
}

func ls(mp *mathParse) error {
	var dir string
	if len(mp.ewords) > 1 {
		dir = mp.ewords[1]
	}
	_ = dir
	fmt.Println(datafs.CurDir.ListShort(false, 0))
	return nil
}

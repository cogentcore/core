// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transpile

import (
	"errors"
	"go/token"
)

var tensorfsCommands = map[string]func(mp *mathParse) error{
	"cd":    cd,
	"mkdir": mkdir,
	"ls":    ls,
}

func cd(mp *mathParse) error {
	var dir string
	if len(mp.ewords) > 1 {
		dir = mp.ewords[1]
	}
	mp.out.Add(token.IDENT, "tensorfs.Chdir")
	mp.out.Add(token.LPAREN)
	mp.out.Add(token.STRING, `"`+dir+`"`)
	mp.out.Add(token.RPAREN)
	return nil
}

func mkdir(mp *mathParse) error {
	if len(mp.ewords) == 1 {
		return errors.New("tensorfs mkdir requires a directory name")
	}
	dir := mp.ewords[1]
	mp.out.Add(token.IDENT, "tensorfs.Mkdir")
	mp.out.Add(token.LPAREN)
	mp.out.Add(token.STRING, `"`+dir+`"`)
	mp.out.Add(token.RPAREN)
	return nil
}

func ls(mp *mathParse) error {
	mp.out.Add(token.IDENT, "tensorfs.List")
	mp.out.Add(token.LPAREN)
	for i := 1; i < len(mp.ewords); i++ {
		mp.out.Add(token.STRING, `"`+mp.ewords[i]+`"`)
	}
	mp.out.Add(token.RPAREN)
	return nil
}

// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/reflectx"
)

func (sh *Shell) ExecLine(ln string) error {
	if sh.Debug >= 1 {
		fmt.Println("shell exec:", ln)
	}
	args := strings.Fields(ln)

	if len(args) == 0 {
		return nil
	}

	cmd := args[0]
	if len(args) == 1 {
		_, err := sh.Exec.Exec(cmd)
		return err
	}

	// do variable substitution based on current symbols from interpreter
	args = args[1:]
	sh.GetSymbols()
	for ai, arg := range args {
		has, v := sh.SymbolByName(arg)
		if has {
			str := reflectx.ToString(v.Interface())
			if str != "" {
				args[ai] = str
			}
		}
	}

	_, err := sh.Exec.Exec(cmd, args...)
	return err
}

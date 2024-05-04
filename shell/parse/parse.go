// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"go/scanner"
	"go/token"
	"reflect"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"github.com/traefik/yaegi/interp"
)

func ErrHand(pos token.Position, msg string) {
	fmt.Println("Error:", pos, msg)
}

func ParseLine(ctx *Context, ln string) {
	fs := token.NewFileSet()
	f := fs.AddFile("curline", fs.Base(), len(ln))
	var sc scanner.Scanner
	sc.Init(f, []byte(ln), ErrHand, scanner.ScanComments)

	// todo: crashing here
	syms := ctx.Interp.Symbols("")

	isGo := false
	pos, tok, lit := sc.Scan()
	fmt.Println(pos, tok, lit)
	switch tok {
	case token.IDENT:
		// only tricky case
		has, _ := SymByName(syms, lit)
		if has {
			isGo = true
		}
	default:
		isGo = true
	}

	if isGo {
		val, err := ctx.Interp.Eval(ln)
		fmt.Println(val, err)
	} else {
		ExecShell(ctx, ln)
	}
}

func ExecShell(ctx *Context, ln string) error {
	fmt.Println("in shell:", ln)
	args := strings.Fields(ln)

	if len(args) == 0 {
		return nil
	}

	cmd := args[0]
	if len(args) == 1 {
		_, err := ctx.Config.Exec(cmd)
		if err != nil {
			fmt.Println(err.Error())
		}
		return err
	}

	args = args[1:]

	syms := ctx.Interp.Symbols("")

	for ai, arg := range args {
		has, v := SymByName(syms, arg)
		if has {
			str := reflectx.ToString(v)
			if str != "" {
				args[ai] = str
			}
		}
	}

	_, err := ctx.Config.Exec(args[0], args[1:]...)
	if err != nil {
		fmt.Println(err.Error())
	}
	return err
}

func SymByName(syms interp.Exports, nm string) (bool, reflect.Value) {
	for path, sy := range syms {
		fmt.Println(path)
		for nm, v := range sy {
			fmt.Println(nm)
			if nm == nm {
				return true, v
			}
		}
	}
	return false, reflect.Value{}
}

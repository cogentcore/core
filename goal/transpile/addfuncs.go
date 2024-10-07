// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transpile

import (
	"path"
	"reflect"
	"strings"

	"cogentcore.org/core/tensor"
	"cogentcore.org/core/yaegicore/nogui"
)

func init() {
	AddYaegiTensorFuncs()
}

// AddYaegiTensorFuncs grabs all tensor* package functions registered
// in yaegicore and adds them to the `tensor.Funcs` map so we can
// properly convert symbols to either tensors or basic literals,
// depending on the arg types for the current function.
func AddYaegiTensorFuncs() {
	for pth, symap := range nogui.Symbols {
		if !strings.Contains(pth, "/core/tensor/") {
			continue
		}
		_, pkg := path.Split(pth)
		for name, val := range symap {
			if val.Kind() != reflect.Func {
				continue
			}
			pnm := pkg + "." + name
			tensor.AddFunc(pnm, val.Interface())
		}
	}
}

// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"fmt"
	"syscall/js"
)

func (app *appImpl) addEventListeners() {
	js.Global().Call("addEventListener", "click", js.FuncOf(app.onClick))
}

func (app *appImpl) onClick(this js.Value, args []js.Value) any {
	x, y := args[0].Get("pageX").Int(), args[0].Get("pageY").Int()
	fmt.Printf("got click event at (%d, %d)\n", x, y)
	return nil
}

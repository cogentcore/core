// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"syscall/js"

	"goki.dev/goosi/mimedata"
)

// TODO(kai/web): support copying images and other mime formats, etc

type clipImpl struct {
}

var theClip = clipImpl{}

func (ci *clipImpl) IsEmpty() bool {
	// no-op
	return false
}

func (ci *clipImpl) Read(types []string) mimedata.Mimes {
	str := make(chan string)
	js.Global().Get("navigator").Get("clipboard").Call("readText").
		Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			str <- args[0].String()
			return nil
		}))
	return mimedata.NewText(<-str)
}

func (ci *clipImpl) Write(data mimedata.Mimes) error {
	if len(data) == 0 {
		return nil
	}
	str := ""
	if len(data) > 1 { // multipart
		mpd := data.ToMultipart()
		str = string(mpd)
	} else {
		d := data[0]
		if mimedata.IsText(d.Type) {
			str = string(d.Data)
		}
	}
	js.Global().Get("navigator").Get("clipboard").Call("writeText", str)
	return nil
}

func (ci *clipImpl) Clear() {
	// nop
}

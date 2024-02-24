// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"syscall/js"

	"cogentcore.org/core/mimedata"
)

// TODO(kai/web): support copying images and other mime formats, etc

// TheClipboard is the single [goosi.Clipboard] for the web platform
var TheClipboard = &Clipboard{}

// Clipboard is the [goosi.Clipboard] implementation for the web platform
type Clipboard struct{}

func (cl *Clipboard) IsEmpty() bool {
	return len(cl.Read(nil).Text(mimedata.TextPlain)) == 0
}

func (cl *Clipboard) Read(types []string) mimedata.Mimes {
	str := make(chan string)
	js.Global().Get("navigator").Get("clipboard").Call("readText").
		Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			str <- args[0].String()
			return nil
		}))
	return mimedata.NewText(<-str)
}

func (cl *Clipboard) Write(data mimedata.Mimes) error {
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

func (cl *Clipboard) Clear() {
	// no-op
}

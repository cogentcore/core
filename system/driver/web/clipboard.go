// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"syscall/js"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/system"
)

// TODO(kai/web): support copying images and other mime formats, etc

// TheClipboard is the single [system.Clipboard] for the web platform
var TheClipboard = &Clipboard{}

// Clipboard is the [system.Clipboard] implementation for the web platform
type Clipboard struct {
	system.ClipboardBase
}

func (cl *Clipboard) Read(types []string) mimedata.Mimes {
	str := make(chan string)
	clip := js.Global().Get("navigator").Get("clipboard")
	if clip.IsUndefined() {
		errors.Log(errors.New("web.Clipboard.Read: navigator.clipboard unsupported"))
		return mimedata.NewText("")
	}
	clip.Call("readText").
		Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			str <- args[0].String()
			return nil
		}), js.FuncOf(func(this js.Value, args []js.Value) any {
			errors.Log(errors.New("web.Clipboard.Read: unable to read clipboard text"))
			str <- ""
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
	clip := js.Global().Get("navigator").Get("clipboard")
	if clip.IsUndefined() {
		return errors.Log(errors.New("web.Clipboard.Write: navigator.clipboard unsupported"))
	}
	clip.Call("writeText", str)
	return nil
}

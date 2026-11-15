// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"sync"
	"syscall/js"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/system"
)

// TODO(kai/web): support copying images and other mime formats, etc

// TheClipboard is the single [system.Clipboard] for the web platform
var TheClipboard = &Clipboard{}

// Clipboard is the [system.Clipboard] implementation for the web platform.
// It keeps the text that it last wrote, which it reads when the browser
// refuses to read the clipboard (as without the permission, or the focus
// of the page), so that copying and pasting in the app still works then.
type Clipboard struct {
	system.ClipboardBase

	// mu guards last.
	mu sync.Mutex

	// last is the text that was last written.
	last string
}

// lastText returns the text that was last written.
func (cl *Clipboard) lastText() string {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	return cl.last
}

func (cl *Clipboard) Read(types []string) mimedata.Mimes {
	str := make(chan string)
	clip := js.Global().Get("navigator").Get("clipboard")
	if clip.IsUndefined() {
		return mimedata.NewText(cl.lastText())
	}
	clip.Call("readText").
		Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			str <- args[0].String()
			return nil
		}), js.FuncOf(func(this js.Value, args []js.Value) any {
			// (refused: the text that the app last wrote)
			str <- cl.lastText()
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
	cl.mu.Lock()
	cl.last = str
	cl.mu.Unlock()
	clip := js.Global().Get("navigator").Get("clipboard")
	if clip.IsUndefined() {
		return nil // (only in the app)
	}
	clip.Call("writeText", str).Call("catch", clipboardWriteFailed())
	return nil
}

// clipboardWriteFailed returns the function that logs a write of the
// clipboard that the browser refuses (as without the focus of the page, or
// the permission), rather than leaving it an unhandled rejection. It is
// made once, as it is used for every write.
var clipboardWriteFailed = sync.OnceValue(func() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		msg := "unable to write clipboard text"
		if len(args) > 0 && args[0].Truthy() {
			msg += ": " + args[0].Call("toString").String()
		}
		errors.Log(errors.New("web.Clipboard.Write: " + msg))
		return nil
	})
})

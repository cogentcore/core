// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"slices"
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
// It keeps the data that it last wrote, with their types (the browser only
// has its text), which it reads while the browser has its text, and until
// the browser has written it, and when the browser refuses to write or
// read the clipboard (as without the permission, or the focus of the
// page), so that copying and pasting in the app still works then.
type Clipboard struct {
	system.ClipboardBase

	// mu guards the fields below.
	mu sync.Mutex

	// last is the text that was last written.
	last string

	// lastData are the data that were last written.
	lastData mimedata.Mimes

	// pending is the number of writes that the browser has not done yet
	// (its writes are asynchronous).
	pending int

	// local is whether the clipboard is read from the data of the app,
	// as the browser does not have them: a write of them is pending, or
	// the last one failed, or they are not text. Otherwise the clipboard
	// is read from the browser (which can have the text of another app).
	local bool
}

func (cl *Clipboard) Read(types []string) mimedata.Mimes {
	cl.mu.Lock()
	local, last, lastData := cl.local, cl.last, cl.lastData
	cl.mu.Unlock()
	if lastData == nil {
		lastData = mimedata.NewText("")
	}
	clip := js.Global().Get("navigator").Get("clipboard")
	if local || clip.IsUndefined() {
		return lastData
	}
	str := make(chan string)
	refused := make(chan bool, 1)
	clip.Call("readText").
		Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			str <- args[0].String()
			return nil
		}), js.FuncOf(func(this js.Value, args []js.Value) any {
			refused <- true
			str <- ""
			return nil
		}))
	text := <-str
	if len(refused) > 0 || text == last {
		return lastData // (the data of the app, with their types)
	}
	return clipboardMimes(text)
}

// clipboardMimes returns the data of the given text of the clipboard,
// which is several of them in a multipart (see [mimedata.Mimes.ToMultipart]),
// as for the copies of lists and trees.
func clipboardMimes(str string) mimedata.Mimes {
	isMulti, mediaType, boundary, body := mimedata.IsMultipart([]byte(str))
	if isMulti {
		return mimedata.FromMultipart(body, boundary)
	}
	if mediaType != "" {
		return mimedata.NewMime(mediaType, []byte(str))
	}
	return mimedata.NewText(str)
}

func (cl *Clipboard) Write(data mimedata.Mimes) error {
	if len(data) == 0 {
		return nil
	}
	// The text of several data is a multipart; data that is not text
	// (as an image) is only in the app.
	str := ""
	if len(data) > 1 {
		str = string(data.ToMultipart())
	} else if mimedata.IsText(data[0].Type) {
		str = string(data[0].Data)
	}
	clip := js.Global().Get("navigator").Get("clipboard")
	toBrowser := !clip.IsUndefined() && str != ""
	cl.mu.Lock()
	cl.last = str
	cl.lastData = slices.Clone(data)
	cl.local = true
	if toBrowser {
		cl.pending++
	}
	cl.mu.Unlock()
	if !toBrowser {
		return nil // (only in the app)
	}
	clip.Call("writeText", str).Call("then", clipboardWritten(), clipboardWriteFailed())
	return nil
}

// written records a write of the browser, which succeeded or not.
func (cl *Clipboard) written(ok bool) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.pending--
	if ok && cl.pending == 0 {
		cl.local = false
	}
}

// clipboardWritten returns the function that records a write of the
// clipboard by the browser. It is made once, as it is used for every write.
var clipboardWritten = sync.OnceValue(func() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		TheClipboard.written(true)
		return nil
	})
})

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
		TheClipboard.written(false)
		return nil
	})
})

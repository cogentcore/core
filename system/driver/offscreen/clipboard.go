// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"sync"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/system"
)

// Clipboard is an in-memory [system.Clipboard] shared by all of the
// windows of the offscreen app, so that copy and paste can be tested.
// Like the desktop clipboards, it holds only text: multiple items are
// encoded as a multipart MIME string, and only text types are read back.
type Clipboard struct {
	mu sync.Mutex

	// text is the text on the clipboard, nil if it is empty.
	text []byte
}

var _ system.Clipboard = &Clipboard{}

// TheClipboard is the single [Clipboard] of the offscreen app.
var TheClipboard = &Clipboard{}

func (a *App) Clipboard(win system.Window) system.Clipboard {
	return TheClipboard
}

func (cl *Clipboard) IsEmpty() bool {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	return cl.text == nil
}

func (cl *Clipboard) Read(types []string) mimedata.Mimes {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if len(types) == 0 || cl.text == nil || !mimedata.IsText(types[0]) {
		return nil
	}
	isMulti, mediaType, boundary, body := mimedata.IsMultipart(cl.text)
	if isMulti {
		return mimedata.FromMultipart(body, boundary)
	}
	if mediaType != "" {
		return mimedata.NewMime(mediaType, cl.text)
	}
	return mimedata.NewMime(types[0], cl.text)
}

func (cl *Clipboard) Write(data mimedata.Mimes) error {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.text = nil
	if len(data) > 1 {
		cl.text = data.ToMultipart()
	} else if len(data) == 1 && mimedata.IsText(data[0].Type) {
		cl.text = append([]byte{}, data[0].Data...)
	}
	return nil
}

func (cl *Clipboard) Clear() {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.text = nil
}

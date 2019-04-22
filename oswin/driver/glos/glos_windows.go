// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package glos

import (
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mimedata"
)

/////////////////////////////////////////////////////////////////
// OS-specific methods

// this is the main call to create the main menu if not exist
func (w *windowImpl) MainMenu() oswin.MainMenu {
	return nil
}

func (w *windowImpl) OSHandle() uintptr {
	return uintptr(w.glw.GetWin32Window())
}

/////////////////////////////////////////////////////////////////
//   Clipboard

type clipImpl struct {
	lastWrite mimedata.Mimes
}

var theClip = clipImpl{}

func (ci *clipImpl) IsEmpty() bool {
	w := theApp.ctxtwin
	str, err := w.glw.GetClipboardString()
	if err != nil || len(str) == 0 {
		return true
	}
	return false
}

func (ci *clipImpl) Read(types []string) mimedata.Mimes {
	w := theApp.ctxtwin
	str, err := w.glw.GetClipboardString()
	if err != nil || len(str) == 0 {
		return nil
	}
	wantText := mimedata.IsText(types[0])
	if wantText {
		str, err := w.glw.GetClipboardString()
		if err != nil || len(str) == 0 {
			return nil
		}
		isMulti, mediaType, boundary, body := mimedata.IsMultipart(str)
		if isMulti {
			return mimedata.FromMultipart(body, boundary)
		} else {
			if mediaType != "" { // found a mime type encoding
				return mimedata.NewMime(mediaType, str)
			} else {
				// we can't really figure out type, so just assume..
				return mimedata.NewMime(types[0], str)
			}
		}
	} else {
		// todo: deal with image formats etc
	}
	return nil
}

func (ci *clipImpl) Write(data mimedata.Mimes) error {
	if len(data) == 0 {
		return nil
	}
	w := theApp.ctxtwin
	if len(data) > 1 { // multipart
		mpd := data.ToMultipart()
		w.wgl.SetClipboardString(string(mpd))
	} else {
		d := data[0]
		if mimedata.IsText(d.Type) {
			w.wgl.SetClipboardString(string(d.Data))
		}
	}
	return nil
}

func (ci *clipImpl) Clear() {
	// nop
}

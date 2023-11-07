// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package web

import (
	"syscall/js"

	"goki.dev/goosi/mimedata"
)

type clipImpl struct {
	lastWrite mimedata.Mimes
}

var theClip = clipImpl{}

func (ci *clipImpl) IsEmpty() bool {
	// str := glfw.GetClipboardString()
	// if len(str) == 0 {
	// 	return true
	// }
	return false
}

func (ci *clipImpl) Read(types []string) mimedata.Mimes {
	// str := glfw.GetClipboardString()
	// if len(str) == 0 {
	// 	return nil
	// }
	// wantText := mimedata.IsText(types[0])
	// if wantText {
	// 	bstr := []byte(str)
	// 	isMulti, mediaType, boundary, body := mimedata.IsMultipart(bstr)
	// 	if isMulti {
	// 		return mimedata.FromMultipart(body, boundary)
	// 	} else {
	// 		if mediaType != "" { // found a mime type encoding
	// 			return mimedata.NewMime(mediaType, bstr)
	// 		} else {
	// 			// we can't really figure out type, so just assume..
	// 			return mimedata.NewMime(types[0], bstr)
	// 		}
	// 	}
	// } else {
	// 	// todo: deal with image formats etc
	// }
	return nil
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

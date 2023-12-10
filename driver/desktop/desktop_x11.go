// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (linux && !android) || dragonfly || openbsd

package desktop

import (
	"log"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/goosi"
	"goki.dev/goosi/mimedata"
	"goki.dev/grr"
)

// Notes on intermixing glfw and xgb: bottom line, can't do:
//
// can include Xlib-xcb.h to get XGetXCBConnection method, which can get the
// xcb_connection from the Display: https://xcb.freedesktop.org/MixingCalls/
// which glfw can return, in GetX11Display().
// BUT, BurntSushi/xgb does NOT seem to directly use the xcb_connection
// and instead is a complete ground-up rewrite using net.Conn connection protocol
// not sure if we can have 2 separate connections..
// and really, maybe we don't need it after all!?  just use the
// text-based clipboard mechanisms to write mime-encoded content, and
// cursor impl has full support for creating new cursors, so..

/////////////////////////////////////////////////////////////////
// OS-specific methods

func (a *App) Platform() goosi.Platforms {
	return goosi.LinuxX11
}

func (a *App) OpenURL(url string) {
	cmd := exec.Command("xdg-open", url)
	grr.Log(cmd.Run())
}

func (a *App) DataDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Print(err)
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, ".config")
}

func (w *Window) Handle() any {
	return uintptr(w.Glw.GetX11Window())
}

/////////////////////////////////////////////////////////////////
//   Clipboard

// TheClip is the single [clip.Board] for Linux
var TheClip = &Clip{}

// Clip is the [clip.Board] implementation for Linux
type Clip struct{}

func (cl *Clip) IsEmpty() bool {
	str := glfw.GetClipboardString()
	if len(str) == 0 {
		return true
	}
	return false
}

func (cl *Clip) Read(types []string) mimedata.Mimes {
	str := glfw.GetClipboardString()
	if len(str) == 0 {
		return nil
	}
	wantText := mimedata.IsText(types[0])
	if wantText {
		bstr := []byte(str)
		isMulti, mediaType, boundary, body := mimedata.IsMultipart(bstr)
		if isMulti {
			return mimedata.FromMultipart(body, boundary)
		} else {
			if mediaType != "" { // found a mime type encoding
				return mimedata.NewMime(mediaType, bstr)
			} else {
				// we can't really figure out type, so just assume..
				return mimedata.NewMime(types[0], bstr)
			}
		}
	} else {
		// todo: deal with image formats etc
	}
	return nil
}

func (cl *Clip) Write(data mimedata.Mimes) error {
	if len(data) == 0 {
		return nil
	}
	// w := theApp.ctxtwin
	if len(data) > 1 { // multipart
		mpd := data.ToMultipart()
		glfw.SetClipboardString(string(mpd))
	} else {
		d := data[0]
		if mimedata.IsText(d.Type) {
			glfw.SetClipboardString(string(d.Data))
		}
	}
	return nil
}

func (cl *Clip) Clear() {
	// nop
}

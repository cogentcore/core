// Copyright 2018 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package desktop

import (
	"os/exec"
	"os/user"
	"path/filepath"
	"unsafe"

	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/mimedata"
	"github.com/go-gl/glfw/v3.3/glfw"
)

/////////////////////////////////////////////////////////////////
// OS-specific methods

func (app *App) Platform() goosi.Platforms {
	return goosi.Windows
}

func (app *App) OpenURL(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	grr.Log(cmd.Run())
}

func (app *App) DataDir() string {
	// todo: could use a more official windows protocol to get this stuff..
	// https://msdn.microsoft.com/en-us/library/bb762188%28VS.85%29.aspx
	// with FOLDERID_RoamingAppData
	// https://stackoverflow.com/questions/6883779/what-are-the-best-practices-for-storing-user-preferences-and-settings-in-win32-d
	usr, err := user.Current()
	if grr.Log(err) != nil {
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "AppData", "Roaming")
	// todo: convention is "manufacturer" before app -- not sure what that means in this context -- "Go"?
}

func (w *Window) Handle() any {
	return uintptr(unsafe.Pointer(w.Glw.GetWin32Window()))
}

/////////////////////////////////////////////////////////////////
//   Clipboard

// TheClip is the single [goosi.Clipboard] for Windows
var TheClip = &Clip{}

// Clip is the [goosi.Clipboard] implementation for Windows
type Clip struct{}

func (cl *Clip) IsEmpty() bool {
	str := glfw.GetClipboardString()
	return len(str) == 0
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
	// no-op
}

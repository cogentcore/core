// Copyright 2018 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package desktop

import (
	"os/exec"
	"os/user"
	"path/filepath"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/system"
	"github.com/go-gl/glfw/v3.3/glfw"
)

/////////////////////////////////////////////////////////////////
// OS-specific methods

func (a *App) Platform() system.Platforms {
	return system.Windows
}

func (a *App) OpenURL(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	errors.Log(cmd.Run())
}

func (a *App) DataDir() string {
	// todo: could use a more official windows protocol to get this stuff..
	// https://msdn.microsoft.com/en-us/library/bb762188%28VS.85%29.aspx
	// with FOLDERID_RoamingAppData
	// https://stackoverflow.com/questions/6883779/what-are-the-best-practices-for-storing-user-preferences-and-settings-in-win32-d
	usr, err := user.Current()
	if errors.Log(err) != nil {
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "AppData", "Roaming")
}

/////////////////////////////////////////////////////////////////
//   Clipboard

// TheClipboard is the single [system.Clipboard] for Windows
var TheClipboard = &Clipboard{}

// Clipboard is the [system.Clipboard] implementation for Windows
type Clipboard struct{}

func (cl *Clipboard) IsEmpty() bool {
	str := glfw.GetClipboardString()
	return len(str) == 0
}

func (cl *Clipboard) Read(types []string) mimedata.Mimes {
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

func (cl *Clipboard) Write(data mimedata.Mimes) error {
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

func (cl *Clipboard) Clear() {
	// no-op
}

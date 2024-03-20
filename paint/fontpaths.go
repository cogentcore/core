// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import "runtime"

// FontPaths contains the filepaths in which fonts are stored for the current platform.
var FontPaths []string

func init() {
	switch runtime.GOOS {
	case "android":
		FontPaths = []string{"/system/fonts"}
	case "darwin", "ios":
		FontPaths = []string{"/System/Library/Fonts", "/Library/Fonts"}
	case "js":
		FontPaths = []string{"/fonts"}
	case "linux":
		// different distros have a different path
		FontPaths = []string{"/usr/share/fonts/truetype", "/usr/share/fonts/TTF"}
	case "windows":
		FontPaths = []string{"C:\\Windows\\Fonts"}
	}
}

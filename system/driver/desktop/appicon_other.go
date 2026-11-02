// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !darwin

package desktop

import "image"

// setAppIcon does nothing: the windows have the icons (see [Window.SetIcon]),
// which are only set as the app icon on macOS.
func setAppIcon(images []image.Image) {}

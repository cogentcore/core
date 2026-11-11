// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && novideo

package video

import "errors"

var errUnsupported = errors.New("video: this app is built without video (the novideo build tag)")

// OpenMedia returns an error, as the app is built without video
// (the novideo build tag), which needs the FFmpeg libraries.
func OpenMedia(fpath string, options map[string]string) (*Media, error) {
	return nil, errUnsupported
}

// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package video

import "errors"

var errUnsupported = errors.New("video: decoding video is not supported on the web, where the browser plays it")

// OpenMedia returns the media at the given path or URL, which the browser
// plays; its properties are not known until it has loaded it, and the
// options are not used.
func OpenMedia(fpath string, options map[string]string) (*Media, error) {
	return &Media{Path: fpath, Options: options}, nil
}

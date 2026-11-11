// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js || novideo

package video

import "time"

// The decoder of FFmpeg is not used on the web (where the browser plays
// videos) nor with the novideo build tag (see [Media]).

type decoder struct{ scaleWidth, scaleHeight int }

func openDecoder(fpath string, options map[string]string, audio bool) (*decoder, error) {
	return nil, errUnsupported
}

func (d *decoder) free() {}

func (d *decoder) seek(t time.Duration) error { return errUnsupported }

func (d *decoder) run(sink decodeSink) error { return errUnsupported }

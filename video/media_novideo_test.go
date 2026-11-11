// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && novideo

package video

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNoVideo tests that an app built without video (the novideo build tag)
// reports that it cannot open or play media.
func TestNoVideo(t *testing.T) {
	_, err := OpenMedia("../examples/video/countdown.mp4", nil)
	assert.ErrorIs(t, err, errUnsupported)
	p := &Playback{Media: &Media{Path: "x.mp4"}, OnFrame: func(Frame) {}}
	assert.ErrorIs(t, p.Play(), errUnsupported)
	assert.False(t, p.IsPlaying())
}

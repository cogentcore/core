// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

import (
	"bytes"
	"fmt"
	"image"

	"github.com/disintegration/imaging"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func init() {
	ffmpeg.LogCompiledCommand = false
}

// ReadFrame reads the given frame number from the given video file as a JPEG image.
func ReadFrame(file string, frame int) (image.Image, error) {
	buf := &bytes.Buffer{}
	err := ffmpeg.Input(file).
		Filter("select", ffmpeg.Args{fmt.Sprintf("gte(n,%d)", frame)}).
		Output("pipe:", ffmpeg.KwArgs{"vframes": 1, "format": "image2", "vcodec": "mjpeg"}).
		WithOutput(buf).
		Run()
	if err != nil {
		return nil, fmt.Errorf("error getting frame %d from video %q: %w", frame, file, err)
	}
	img, err := imaging.Decode(buf)
	if err != nil {
		return nil, fmt.Errorf("error decoding frame %d from video %q into image: %w", frame, file, err)
	}
	return img, nil
}

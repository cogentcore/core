// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/disintegration/imaging"
	"goki.dev/video"
)

func main() {
	img, err := video.ReadFrame("./in.mp4", 150)
	if err != nil {
		panic(err)
	}
	err = imaging.Save(img, "./out.jpg")
	if err != nil {
		panic(err)
	}
}

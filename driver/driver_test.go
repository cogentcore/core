// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package driver

import (
	"fmt"
	"image"
	"testing"

	"goki.dev/goosi"
)

func TestMain(t *testing.T) {
	Main(func(a goosi.App) {
		opts := &goosi.NewWindowOptions{
			Size:      image.Pt(1024, 768),
			StdPixels: true,
			Title:     "Goosi Test Window",
		}
		w, err := goosi.TheApp.NewWindow(opts)
		if err != nil {
			t.Error(err)
		}
		fmt.Println(w.Name())
		for {
			evi := w.NextEvent()
			fmt.Println(evi)
		}
	})
}

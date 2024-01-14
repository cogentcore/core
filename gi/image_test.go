// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"testing"

	"goki.dev/grr"
)

func TestImageBasic(t *testing.T) {
	sc := NewScene()
	img := NewImage(sc)
	grr.Test(t, img.OpenImage(FileName(filepath.Join("..", "logo", "goki_logo.png"))))
	sc.AssertPixelsOnShow(t, filepath.Join("image", "basic"))
}

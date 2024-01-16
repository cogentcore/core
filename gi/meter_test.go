// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"path/filepath"
	"strconv"
	"testing"
)

func TestMeter(t *testing.T) {
	for v := 0; v <= 100; v += 10 {
		sc := NewScene()
		NewMeter(sc).SetMax(100).SetValue(float32(v))
		sc.AssertRender(t, filepath.Join("meter", strconv.Itoa(v)))
	}
}

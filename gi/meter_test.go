// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"strconv"
	"testing"

	"cogentcore.org/core/styles"
)

func TestMeter(t *testing.T) {
	for _, d := range styles.DirectionsValues() {
		for v := 0; v <= 100; v += 10 {
			b := NewBody()
			NewMeter(b).SetMax(100).SetValue(float32(v)).Style(func(s *styles.Style) {
				s.Direction = d
			})
			b.AssertRender(t, "meter/"+d.String()+"/"+strconv.Itoa(v))
		}
	}
}

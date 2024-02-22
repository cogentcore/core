// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/styles"
)

func TestMeter(t *testing.T) {
	for v := float32(0); v <= 100; v += 10 {
		for _, typ := range MeterTypesValues() {
			b := NewBody()
			NewMeter(b).SetMax(100).SetType(typ).SetValue(v)
			b.AssertRender(t, testName("meter", typ, v))
		}
		b := NewBody()
		NewMeter(b).SetMax(100).SetValue(v).Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})
		b.AssertRender(t, testName("meter", "column", v))
	}
}

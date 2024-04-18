// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"testing"

	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
)

func TestMeter(t *testing.T) {
	for v := float32(0); v <= 100; v += 10 {
		for _, typ := range MeterTypesValues() {
			b := NewBody()
			NewMeter(b).SetMax(100).SetType(typ).SetValue(v).SetText(fmt.Sprintf("%g%%", v))
			b.AssertRender(t, fmt.Sprintf("meter/%s/%g", strcase.ToKebab(typ.String()), v))
		}
		b := NewBody()
		NewMeter(b).SetMax(100).SetValue(v).Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})
		b.AssertRender(t, fmt.Sprintf("meter/column/%g", v))
	}
}

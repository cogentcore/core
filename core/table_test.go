// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
)

type language struct {
	Name   string
	Rating int
}

func (l *language) FieldWidget(field string) Value {
	if field == "Rating" {
		return NewSlider().SetMin(1).SetMax(10).SetStep(1)
	}
	return nil
}

func TestTable(t *testing.T) {
	b := NewBody()
	NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
	b.AssertRender(t, "table/basic")
}

func TestTableReadOnly(t *testing.T) {
	b := NewBody()
	NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}}).SetReadOnly(true)
	b.AssertRender(t, "table/read-only")
}

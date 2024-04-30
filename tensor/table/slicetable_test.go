// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Data struct {
	City       string
	Population float32
	Area       float32
}

func TestSliceTable(t *testing.T) {
	data := []Data{
		{"Davis", 62000, 500},
		{"Boulder", 85000, 800},
	}

	dt, err := NewSliceTable(data)
	if err != nil {
		t.Error(err.Error())
	}
	assert.Equal(t, 2, dt.Rows)
}

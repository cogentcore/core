// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histogram

import (
	"testing"

	"cogentcore.org/core/tensor/table"
	"github.com/stretchr/testify/assert"
)

func TestHistogram32(t *testing.T) {
	vals := []float32{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}
	ex := []float32{4, 3, 4}
	res := []float32{}

	F32(&res, vals, 3, 0, 1)

	assert.Equal(t, ex, res)

	exvals := []float32{0, 0.3333, 0.6667}
	dt := table.NewTable()
	F32Table(dt, vals, 3, 0, 1)
	for ri, v := range ex {
		vv := float32(dt.Float("Value", ri))
		cv := float32(dt.Float("Count", ri))
		assert.Equal(t, exvals[ri], vv)
		assert.Equal(t, v, cv)
	}
}

func TestHistogram64(t *testing.T) {
	vals := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1}
	ex := []float64{4, 3, 4}
	res := []float64{}

	F64(&res, vals, 3, 0, 1)

	assert.Equal(t, ex, res)

	exvals := []float64{0, 0.3333, 0.6667}
	dt := table.NewTable()
	F64Table(dt, vals, 3, 0, 1)
	for ri, v := range ex {
		vv := dt.Float("Value", ri)
		cv := dt.Float("Count", ri)
		assert.Equal(t, exvals[ri], vv)
		assert.Equal(t, v, cv)
	}
}

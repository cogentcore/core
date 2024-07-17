// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histogram

//go:generate core generate

import (
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// F64 generates a histogram of counts of values within given
// number of bins and min / max range.  hist vals is sized to nBins.
// if value is < min or > max it is ignored.
func F64(hist *[]float64, vals []float64, nBins int, min, max float64) {
	*hist = slicesx.SetLength(*hist, nBins)
	h := *hist
	// 0.1.2.3 = 3-0 = 4 bins
	inc := (max - min) / float64(nBins)
	for i := 0; i < nBins; i++ {
		h[i] = 0
	}
	for _, v := range vals {
		if v < min || v > max {
			continue
		}
		bin := int((v - min) / inc)
		if bin >= nBins {
			bin = nBins - 1
		}
		h[bin] += 1
	}
}

// F64Table generates an table with a histogram of counts of values within given
// number of bins and min / max range. The table has columns: Value, Count
// if value is < min or > max it is ignored.
// The Value column represents the min value for each bin, with the max being
// the value of the next bin, or the max if at the end.
func F64Table(dt *table.Table, vals []float64, nBins int, min, max float64) {
	dt.DeleteAll()
	dt.AddFloat64Column("Value")
	dt.AddFloat64Column("Count")
	dt.SetNumRows(nBins)
	ct := dt.Columns[1].(*tensor.Float64)
	F64(&ct.Values, vals, nBins, min, max)
	inc := (max - min) / float64(nBins)
	vls := dt.Columns[0].(*tensor.Float64).Values
	for i := 0; i < nBins; i++ {
		vls[i] = math32.Truncate64(min+float64(i)*inc, 4)
	}
}

//////////////////////////////////////////////////////
// float32

// F32 generates a histogram of counts of values within given
// number of bins and min / max range.  hist vals is sized to nBins.
// if value is < min or > max it is ignored.
func F32(hist *[]float32, vals []float32, nBins int, min, max float32) {
	*hist = slicesx.SetLength(*hist, nBins)
	h := *hist
	// 0.1.2.3 = 3-0 = 4 bins
	inc := (max - min) / float32(nBins)
	for i := 0; i < nBins; i++ {
		h[i] = 0
	}
	for _, v := range vals {
		if v < min || v > max {
			continue
		}
		bin := int((v - min) / inc)
		if bin >= nBins {
			bin = nBins - 1
		}
		h[bin] += 1
	}
}

// F32Table generates an table with a histogram of counts of values within given
// number of bins and min / max range. The table has columns: Value, Count
// if value is < min or > max it is ignored.
// The Value column represents the min value for each bin, with the max being
// the value of the next bin, or the max if at the end.
func F32Table(dt *table.Table, vals []float32, nBins int, min, max float32) {
	dt.DeleteAll()
	dt.AddFloat32Column("Value")
	dt.AddFloat32Column("Count")
	dt.SetNumRows(nBins)
	ct := dt.Columns[1].(*tensor.Float32)
	F32(&ct.Values, vals, nBins, min, max)
	inc := (max - min) / float32(nBins)
	vls := dt.Columns[0].(*tensor.Float32).Values
	for i := 0; i < nBins; i++ {
		vls[i] = math32.Truncate(min+float32(i)*inc, 4)
	}
}

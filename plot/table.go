// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"fmt"
	"log/slog"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// NewTablePlot returns a new Plot with all configuration based on given
// [table.Table] set of columns and associated metadata.
// Only columns marked as On are plotted.
// Must have a column with a Role = X (first one found is used).
func NewTablePlot(dt *table.Table) (*Plot, error) {
	var xt tensor.Values
	// var xst *Style
	for _, cl := range dt.Columns.Values {
		st := &Style{}
		stl := GetStylersFrom(cl)
		if stl == nil {
			continue
		}
		stl.Run(st)
		if st.Role == X {
			xt = cl
			// xst = st
			break
		}
	}
	if xt == nil {
		return nil, errors.New("plot.NewTablePlot: X axis (Style.Role = X) not found")
	}
	plt := New()
	for _, cl := range dt.Columns.Values {
		st := &Style{}
		stl := GetStylersFrom(cl)
		if stl == nil {
			continue
		}
		stl.Run(st)
		if st.On != On || st.Role == X {
			continue
		}
		ptyp := "XY"
		if st.Plotter != "" {
			ptyp = st.Plotter
		}
		pt, err := PlotterByType(ptyp)
		if errors.Log(err) != nil {
			continue
		}
		// todo: collect all roles from group
		pl := pt.New(Data{X: xt, st.Role: cl})
		if pl != nil {
			fmt.Println("adding pl", pl)
			plt.Add(pl)
		} else {
			slog.Error("plot.NewTablePlot: error in creating plotter of type:", ptyp)
		}
	}
	return plt, nil
}

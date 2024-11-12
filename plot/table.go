// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"fmt"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// NewTablePlot returns a new Plot with all configuration based on given
// [table.Table] set of columns and associated metadata, which must have
// [Stylers] functions set (e.g., [SetStylersTo]) that at least set basic
// table parameters, including:
//   - On: Set the main (typically Role = Y) column On to include in plot.
//   - Role: Set the appropriate [Roles] role for this column (Y, X, etc).
//   - Group: Multiple columns used for a given Plotter type must be grouped
//     together with a common name (typically the name of the main Y axis),
//     e.g., for Low, High error bars, Size, Color, etc. If only one On column,
//     then Group can be empty and all other such columns will be grouped.
//   - Plotter: Determines the type of Plotter element to use, which in turn
//     determines the additional Roles that can be used within a Group.
func NewTablePlot(dt *table.Table) (*Plot, error) {
	nc := len(dt.Columns.Values)
	if nc == 0 {
		return nil, errors.New("plot.NewTablePlot: no columns in data table")
	}
	csty := make(map[tensor.Values]*Style, nc)
	gps := make(map[string][]tensor.Values, nc)
	var xt tensor.Values
	var errs []error
	for _, cl := range dt.Columns.Values {
		st := &Style{}
		st.Defaults()
		stl := GetStylersFrom(cl)
		if stl == nil {
			continue
		}
		csty[cl] = st
		stl.Run(st)
		gps[st.Group] = append(gps[st.Group], cl)
		if xt == nil && st.Role == X {
			xt = cl
		}
	}
	doneGps := map[string]bool{}
	plt := New()
	for ci, cl := range dt.Columns.Values {
		cnm := dt.Columns.Keys[ci]
		st := csty[cl]
		if st == nil || !st.On || st.Role == X {
			continue
		}
		gp := st.Group
		if doneGps[gp] {
			continue
		}
		if gp != "" {
			doneGps[gp] = true
		}
		ptyp := "XY"
		if st.Plotter != "" {
			ptyp = string(st.Plotter)
		}
		pt, err := PlotterByType(ptyp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		data := Data{st.Role: cl}
		gcols := gps[gp]
		gotReq := true
		for _, rl := range pt.Required {
			if rl == st.Role {
				continue
			}
			got := false
			for _, gc := range gcols {
				gst := csty[gc]
				if gst.Role == rl {
					data[rl] = gc
					got = true
					break
				}
			}
			if !got {
				if rl == X && xt != nil {
					data[rl] = xt
				} else {
					err = fmt.Errorf("plot.NewTablePlot: Required Role %q not found in Group %q, Plotter %q not added for Column: %q", rl.String(), gp, ptyp, cnm)
					errs = append(errs, err)
					gotReq = false
					fmt.Println(err)
				}
			}
		}
		if !gotReq {
			continue
		}
		for _, rl := range pt.Optional {
			if rl == st.Role { // should not happen
				continue
			}
			for _, gc := range gcols {
				gst := csty[gc]
				if gst.Role == rl {
					data[rl] = gc
					break
				}
			}
		}
		pl := pt.New(data)
		if pl != nil {
			plt.Add(pl)
		} else {
			err = fmt.Errorf("plot.NewTablePlot: error in creating plotter type: %q", ptyp)
			errs = append(errs, err)
		}
	}
	return plt, errors.Join(errs...)
}

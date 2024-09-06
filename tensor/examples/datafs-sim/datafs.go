// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/datafs"
)

var data *datafs.Data

// ConfigStats adds basic stats that we record for our simulation.
func ConfigStats(dir *datafs.Data) *datafs.Data {
	stats := errors.Log1(dir.Mkdir("Stats"))
	errors.Log1(datafs.New[int](stats, "Run", "Epoch", "Trial")) // counters
	errors.Log1(datafs.New[string](stats, "TrialName"))
	errors.Log1(datafs.New[float32](stats, "SSE", "AvgErr", "TrlErr"))
	z1 := datafs.PlotColumnZeroOne()
	stats.SetPlotColumnOptions(z1, "AvgErr", "TrlErr")
	zmax := datafs.PlotColumnZeroOne()
	zmax.Range.FixMax = false
	stats.SetPlotColumnOptions(z1, "SSE")
	return stats
}

// ConfigTrialLog adds first-level logging of stats into tensors
func ConfigTrialLog(dir *datafs.Data, stats *datafs.Data, nrows int) *datafs.Data {
	trial := errors.Log1(dir.Mkdir("Trial"))
	sitems := stats.ItemsAddedFunc(func(it *datafs.Data) bool {
		return !it.IsDir()
	})
	for _, st := range sitems {
		dt := errors.Log1(datafs.NewData(trial, st.Name()))
		dt.Value = tensor.NewOfType(st.DataType(), []int{nrows}, "row")
		dt.CopyMetadata(st.Meta) // key affordance: we get meta data from source
	}
	return trial
}

// ConfigAggLog adds a higher-level logging of lower-level into higher-level tensors
func ConfigAggLog(dir *datafs.Data, level string, stats *datafs.Data, nrows int, aggs ...string) *datafs.Data {
	aglog := errors.Log1(dir.Mkdir(level))
	sitems := stats.ItemsAddedFunc(func(it *datafs.Data) bool {
		return !it.IsDir()
	})
	for _, st := range sitems {
		dd := errors.Log1(aglog.Mkdir(st.Name()))
		for _, ag := range aggs { // key advantage of dir structure: multiple stats per item
			dt := errors.Log1(datafs.NewData(dd, ag))
			dt.Value = tensor.NewOfType(st.DataType(), []int{nrows}, "row")
		}
	}
	return aglog
}

func main() {
	data = errors.Log1(datafs.NewDir("/"))
	sim := errors.Log1(data.Mkdir("sim"))
	stats := ConfigStats(sim)
	ntrials := 25
	trial := ConfigTrialLog(sim, stats, ntrials)
	epoch := ConfigAggLog(sim, "Epoch", stats, ntrials, "Mean", "SEM", "Min")
	_ = trial
	_ = epoch

	// note: it would be convenient to be able to put the compute closure right here
	// like it is in the logitem

}

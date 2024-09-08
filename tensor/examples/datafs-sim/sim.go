// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math/rand/v2"
	"reflect"
	"strconv"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/databrowser"
	"cogentcore.org/core/tensor/datafs"
	"cogentcore.org/core/tensor/stats/stats"
)

type Sim struct {
	Root   *datafs.Data
	Config *datafs.Data
	Stats  *datafs.Data
	Logs   *datafs.Data
}

// ConfigAll configures the sim
func (ss *Sim) ConfigAll() {
	ss.Root = errors.Log1(datafs.NewDir("Root"))
	ss.Config = errors.Log1(ss.Root.Mkdir("Config"))
	errors.Log1(datafs.New[int](ss.Config, "NRun", "NEpoch", "NTrial"))
	ss.Config.Item("NRun").SetInt(5)
	ss.Config.Item("NEpoch").SetInt(20)
	ss.Config.Item("NTrial").SetInt(25)

	ss.Stats = ss.ConfigStats(ss.Root)
	ss.Logs = ss.ConfigLogs(ss.Root)
}

// ConfigStats adds basic stats that we record for our simulation.
func (ss *Sim) ConfigStats(dir *datafs.Data) *datafs.Data {
	stats := errors.Log1(dir.Mkdir("Stats"))
	errors.Log1(datafs.New[int](stats, "Run", "Epoch", "Trial")) // counters
	errors.Log1(datafs.New[string](stats, "TrialName"))
	errors.Log1(datafs.New[float32](stats, "SSE", "AvgSSE", "TrlErr"))
	z1 := datafs.PlotColumnZeroOne()
	stats.SetPlotColumnOptions(z1, "AvgErr", "TrlErr")
	zmax := datafs.PlotColumnZeroOne()
	zmax.Range.FixMax = false
	stats.SetPlotColumnOptions(z1, "SSE")
	return stats
}

// ConfigLogs adds first-level logging of stats into tensors
func (ss *Sim) ConfigLogs(dir *datafs.Data) *datafs.Data {
	logd := errors.Log1(dir.Mkdir("Log"))
	trial := ss.ConfigTrialLog(logd)
	ss.ConfigAggLog(logd, "Epoch", trial, stats.Mean, stats.Sem, stats.Min)
	return logd
}

// ConfigTrialLog adds first-level logging of stats into tensors
func (ss *Sim) ConfigTrialLog(dir *datafs.Data) *datafs.Data {
	logd := errors.Log1(dir.Mkdir("Trial"))
	ntrial, _ := ss.Config.Item("NTrial").AsInt()
	sitems := ss.Stats.ItemsByTimeFunc(nil)
	for _, st := range sitems {
		dt := errors.Log1(datafs.NewData(logd, st.Name()))
		tsr := tensor.NewOfType(st.DataType(), []int{ntrial}, "row")
		dt.Value = tsr
		dt.Meta.Copy(st.Meta) // key affordance: we get meta data from source
		dt.SetCalcFunc(func() error {
			trl, _ := ss.Stats.Item("Trial").AsInt()
			if st.IsNumeric() {
				v, _ := st.AsFloat64()
				tsr.SetFloat1D(trl, v)
			} else {
				v, _ := st.AsString()
				tsr.SetString1D(trl, v)
			}
			return nil
		})
	}
	return logd
}

// ConfigAggLog adds a higher-level logging of lower-level into higher-level tensors
func (ss *Sim) ConfigAggLog(dir *datafs.Data, level string, from *datafs.Data, aggs ...stats.Stats) *datafs.Data {
	logd := errors.Log1(dir.Mkdir(level))
	sitems := ss.Stats.ItemsByTimeFunc(nil)
	nctr, _ := ss.Config.Item("N" + level).AsInt()
	for _, st := range sitems {
		if !st.IsNumeric() {
			continue
		}
		src := from.Item(st.Name()).AsTensor()
		if st.DataType() >= reflect.Float32 {
			dd := errors.Log1(logd.Mkdir(st.Name()))
			for _, ag := range aggs { // key advantage of dir structure: multiple stats per item
				dt := errors.Log1(datafs.NewData(dd, ag.String()))
				tsr := tensor.NewOfType(st.DataType(), []int{nctr}, "row")
				dt.Value = tsr
				dt.Meta.Copy(st.Meta)
				dt.SetCalcFunc(func() error {
					ctr, _ := ss.Stats.Item(level).AsInt()
					v := stats.StatTensor(src, ag)
					tsr.SetFloat1D(ctr, v)
					return nil
				})
			}
		} else {
			dt := errors.Log1(datafs.NewData(logd, st.Name()))
			tsr := tensor.NewOfType(st.DataType(), []int{nctr}, "row")
			// todo: set level counter as default x axis in plot config
			dt.Value = tsr
			dt.Meta.Copy(st.Meta)
			dt.SetCalcFunc(func() error {
				ctr, _ := ss.Stats.Item(level).AsInt()
				v, _ := st.AsFloat64()
				tsr.SetFloat1D(ctr, v)
				return nil
			})
		}
	}
	return logd
}

func (ss *Sim) Run() {
	nepc, _ := ss.Config.Item("NEpoch").AsInt()
	ntrl, _ := ss.Config.Item("NTrial").AsInt()
	for epc := range nepc {
		ss.Stats.Item("Epoch").SetInt(epc)
		for trl := range ntrl {
			ss.Stats.Item("Trial").SetInt(trl)
			ss.RunTrial(trl)
		}
		ss.EpochDone()
	}
}

func (ss *Sim) RunTrial(trl int) {
	ss.Stats.Item("TrialName").SetString("Trial_" + strconv.Itoa(trl))
	sse := rand.Float32()
	avgSSE := rand.Float32()
	ss.Stats.Item("SSE").SetFloat32(sse)
	ss.Stats.Item("AvgSSE").SetFloat32(avgSSE)
	trlErr := float32(1)
	if sse < 0.5 {
		trlErr = 0
	}
	ss.Stats.Item("TrlErr").SetFloat32(trlErr)
	ss.Logs.Item("Trial").CalcAll()
}

func (ss *Sim) EpochDone() {
	ss.Logs.Item("Epoch").CalcAll()
}

func main() {
	ss := &Sim{}
	ss.ConfigAll()
	ss.Run()

	databrowser.NewBrowserWindow(ss.Root, "Root")
	core.Wait()
}

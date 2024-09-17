// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math/rand/v2"
	"reflect"
	"strconv"

	"cogentcore.org/core/core"
	"cogentcore.org/core/plot/plotcore"
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
	ss.Root, _ = datafs.NewDir("Root")
	ss.Config, _ = ss.Root.Mkdir("Config")
	datafs.NewScalar[int](ss.Config, "NRun", "NEpoch", "NTrial")
	ss.Config.Item("NRun").SetInt(5)
	ss.Config.Item("NEpoch").SetInt(20)
	ss.Config.Item("NTrial").SetInt(25)

	ss.Stats = ss.ConfigStats(ss.Root)
	ss.Logs = ss.ConfigLogs(ss.Root)
}

// ConfigStats adds basic stats that we record for our simulation.
func (ss *Sim) ConfigStats(dir *datafs.Data) *datafs.Data {
	stats, _ := dir.Mkdir("Stats")
	datafs.NewScalar[int](stats, "Run", "Epoch", "Trial") // counters
	datafs.NewScalar[string](stats, "TrialName")
	datafs.NewScalar[float32](stats, "SSE", "AvgSSE", "TrlErr")
	z1, key := plotcore.PlotColumnZeroOne()
	stats.SetMetaItems(key, z1, "AvgErr", "TrlErr")
	zmax, _ := plotcore.PlotColumnZeroOne()
	zmax.Range.FixMax = false
	stats.SetMetaItems(key, z1, "SSE")
	return stats
}

// ConfigLogs adds first-level logging of stats into tensors
func (ss *Sim) ConfigLogs(dir *datafs.Data) *datafs.Data {
	logd, _ := dir.Mkdir("Log")
	trial := ss.ConfigTrialLog(logd)
	ss.ConfigAggLog(logd, "Epoch", trial, stats.Mean, stats.Sem, stats.Min)
	return logd
}

// ConfigTrialLog adds first-level logging of stats into tensors
func (ss *Sim) ConfigTrialLog(dir *datafs.Data) *datafs.Data {
	logd, _ := dir.Mkdir("Trial")
	ntrial := ss.Config.Item("NTrial").AsInt()
	sitems := ss.Stats.ValuesFunc(nil)
	for _, st := range sitems {
		nm := st.Tensor.Metadata().GetName()
		lt := logd.NewOfType(nm, st.Tensor.DataType(), ntrial)
		lt.Tensor.Metadata().Copy(*st.Tensor.Metadata()) // key affordance: we get meta data from source
		tensor.SetCalcFunc(lt.Tensor, func() error {
			trl := ss.Stats.Item("Trial").AsInt()
			if st.Tensor.IsString() {
				lt.SetStringRow(st.StringRow(0), trl)
			} else {
				lt.SetFloatRow(st.FloatRow(0), trl)
			}
			return nil
		})
	}
	alllogd, _ := dir.Mkdir("AllTrials")
	for _, st := range sitems {
		nm := st.Tensor.Metadata().GetName()
		// allocate full size
		lt := alllogd.NewOfType(nm, st.Tensor.DataType(), ntrial*ss.Config.Item("NEpoch").AsInt()*ss.Config.Item("NRun").AsInt())
		lt.Tensor.SetShape(0)                            // then truncate to 0
		lt.Tensor.Metadata().Copy(*st.Tensor.Metadata()) // key affordance: we get meta data from source
		tensor.SetCalcFunc(lt.Tensor, func() error {
			row := lt.Tensor.DimSize(0)
			lt.Tensor.SetShape(row + 1)
			if st.Tensor.IsString() {
				lt.SetStringRow(st.StringRow(0), row)
			} else {
				lt.SetFloatRow(st.FloatRow(0), row)
			}
			return nil
		})
	}
	return logd
}

// ConfigAggLog adds a higher-level logging of lower-level into higher-level tensors
func (ss *Sim) ConfigAggLog(dir *datafs.Data, level string, from *datafs.Data, aggs ...stats.Stats) *datafs.Data {
	logd, _ := dir.Mkdir(level)
	sitems := ss.Stats.ValuesFunc(nil)
	nctr := ss.Config.Item("N" + level).AsInt()
	stout := tensor.NewFloat64Scalar(0) // tmp stat output
	for _, st := range sitems {
		if st.Tensor.IsString() {
			continue
		}
		nm := st.Tensor.Metadata().GetName()
		src := from.Value(nm)
		if st.Tensor.DataType() >= reflect.Float32 {
			// todo: pct correct etc
			dd, _ := logd.Mkdir(nm)
			for _, ag := range aggs { // key advantage of dir structure: multiple stats per item
				lt := dd.NewOfType(ag.String(), st.Tensor.DataType(), nctr)
				lt.Tensor.Metadata().Copy(*st.Tensor.Metadata())
				tensor.SetCalcFunc(lt.Tensor, func() error {
					stats.Stat(ag, src, stout)
					ctr := ss.Stats.Item(level).AsInt()
					lt.SetFloatRow(stout.FloatRow(0), ctr)
					return nil
				})
			}
		} else {
			lt := logd.NewOfType(nm, st.Tensor.DataType(), nctr)
			lt.Tensor.Metadata().Copy(*st.Tensor.Metadata())
			tensor.SetCalcFunc(lt.Tensor, func() error {
				v := st.FloatRow(0)
				ctr := ss.Stats.Item(level).AsInt()
				lt.SetFloatRow(v, ctr)
				return nil
			})
		}
	}
	return logd
}

func (ss *Sim) Run() {
	nrun := ss.Config.Item("NRun").AsInt()
	nepc := ss.Config.Item("NEpoch").AsInt()
	ntrl := ss.Config.Item("NTrial").AsInt()
	for run := range nrun {
		ss.Stats.Item("Run").SetInt(run)
		for epc := range nepc {
			ss.Stats.Item("Epoch").SetInt(epc)
			for trl := range ntrl {
				ss.Stats.Item("Trial").SetInt(trl)
				ss.RunTrial(trl)
			}
			ss.EpochDone()
		}
	}
	alldt := ss.Logs.Item("AllTrials").GetDirTable(nil)
	dir, _ := ss.Logs.Mkdir("Stats")
	stats.TableGroups(dir, alldt, "Run", "Epoch", "Trial")
	sts := []string{"SSE", "AvgSSE", "TrlErr"}
	stats.TableGroupStats(dir, stats.Mean.FuncName(), alldt, sts...)
	stats.TableGroupStats(dir, stats.Sem.FuncName(), alldt, sts...)

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
	ss.Logs.Item("AllTrials").CalcAll()
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

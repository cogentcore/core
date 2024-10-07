// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package plotcore provides Cogent Core widgets for viewing and editing plots.
package plotcore

//go:generate core generate

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorcore"
	"cogentcore.org/core/tree"
)

// PlotEditor is a widget that provides an interactive 2D plot
// of selected columns of tabular data, represented by a [table.Table] into
// a [table.Table]. Other types of tabular data can be converted into this format.
// The user can change various options for the plot and also modify the underlying data.
type PlotEditor struct { //types:add
	core.Frame

	// table is the table of data being plotted.
	table *table.Table

	// Options are the overall plot options.
	Options PlotOptions

	// Columns are the options for each column of the table.
	Columns []*ColumnOptions `set:"-"`

	// plot is the plot object.
	plot *plot.Plot

	// current svg file
	svgFile core.Filename

	// current csv data file
	dataFile core.Filename

	// currently doing a plot
	inPlot bool

	columnsFrame *core.Frame
	plotWidget   *Plot
}

func (pl *PlotEditor) CopyFieldsFrom(frm tree.Node) {
	fr := frm.(*PlotEditor)
	pl.Frame.CopyFieldsFrom(&fr.Frame)
	pl.Options = fr.Options
	pl.setTable(fr.table)
	mx := min(len(pl.Columns), len(fr.Columns))
	for i := 0; i < mx; i++ {
		*pl.Columns[i] = *fr.Columns[i]
	}
}

// NewSubPlot returns a [PlotEditor] with its own separate [core.Toolbar],
// suitable for a tab or other element that is not the main plot.
func NewSubPlot(parent ...tree.Node) *PlotEditor {
	fr := core.NewFrame(parent...)
	tb := core.NewToolbar(fr)
	pl := NewPlotEditor(fr)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	tb.Maker(pl.MakeToolbar)
	return pl
}

func (pl *PlotEditor) Init() {
	pl.Frame.Init()

	pl.Options.defaults()
	pl.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		if pl.SizeClass() == core.SizeCompact {
			s.Direction = styles.Column
		}
	})

	pl.OnShow(func(e events.Event) {
		pl.UpdatePlot()
	})

	pl.Updater(func() {
		if pl.table != nil {
			pl.Options.fromMeta(pl.table)
		}
	})
	tree.AddChildAt(pl, "columns", func(w *core.Frame) {
		pl.columnsFrame = w
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Background = colors.Scheme.SurfaceContainerLow
			if w.SizeClass() == core.SizeCompact {
				s.Grow.Set(1, 0)
			} else {
				s.Grow.Set(0, 1)
				s.Overflow.Y = styles.OverflowAuto
			}
		})
		w.Maker(pl.makeColumns)
	})
	tree.AddChildAt(pl, "plot", func(w *Plot) {
		pl.plotWidget = w
		w.Plot = pl.plot
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
	})
}

// setTable sets the table to view and does Update
// to update the Column list, which will also trigger a Layout
// and updating of the plot on next render pass.
// This is safe to call from a different goroutine.
func (pl *PlotEditor) setTable(tab *table.Table) *PlotEditor {
	pl.table = tab
	pl.Update()
	return pl
}

// SetTable sets the table to view and does Update
// to update the Column list, which will also trigger a Layout
// and updating of the plot on next render pass.
// This is safe to call from a different goroutine.
func (pl *PlotEditor) SetTable(tab *table.Table) *PlotEditor {
	pl.table = table.NewView(tab)
	pl.Update()
	return pl
}

// SetSlice sets the table to a [table.NewSliceTable]
// from the given slice.
func (pl *PlotEditor) SetSlice(sl any) *PlotEditor {
	return pl.SetTable(errors.Log1(table.NewSliceTable(sl)))
}

// ColumnOptions returns the current column options by name
// (to access by index, just use Columns directly).
func (pl *PlotEditor) ColumnOptions(column string) *ColumnOptions {
	for _, co := range pl.Columns {
		if co.Column == column {
			return co
		}
	}
	return nil
}

// Bool constants for [PlotEditor.SetColumnOptions].
const (
	On       = true
	Off      = false
	FixMin   = true
	FloatMin = false
	FixMax   = true
	FloatMax = false
)

// SetColumnOptions sets the main parameters for one column.
func (pl *PlotEditor) SetColumnOptions(column string, on bool, fixMin bool, min float32, fixMax bool, max float32) *ColumnOptions {
	co := pl.ColumnOptions(column)
	if co == nil {
		slog.Error("plotcore.PlotEditor.SetColumnOptions: column not found", "column", column)
		return nil
	}
	co.On = on
	co.Range.FixMin = fixMin
	if fixMin {
		co.Range.Min = min
	}
	co.Range.FixMax = fixMax
	if fixMax {
		co.Range.Max = max
	}
	return co
}

// SaveSVG saves the plot to an svg -- first updates to ensure that plot is current
func (pl *PlotEditor) SaveSVG(fname core.Filename) { //types:add
	pl.UpdatePlot()
	// TODO: get plot svg saving working
	// pc := pl.PlotChild()
	// SaveSVGView(string(fname), pl.Plot, sv, 2)
	pl.svgFile = fname
}

// SavePNG saves the current plot to a png, capturing current render
func (pl *PlotEditor) SavePNG(fname core.Filename) { //types:add
	pl.UpdatePlot()
	imagex.Save(pl.plot.Pixels, string(fname))
}

// SaveCSV saves the Table data to a csv (comma-separated values) file with headers (any delim)
func (pl *PlotEditor) SaveCSV(fname core.Filename, delim tensor.Delims) { //types:add
	pl.table.SaveCSV(fsx.Filename(fname), delim, table.Headers)
	pl.dataFile = fname
}

// SaveAll saves the current plot to a png, svg, and the data to a tsv -- full save
// Any extension is removed and appropriate extensions are added
func (pl *PlotEditor) SaveAll(fname core.Filename) { //types:add
	fn := string(fname)
	fn = strings.TrimSuffix(fn, filepath.Ext(fn))
	pl.SaveCSV(core.Filename(fn+".tsv"), tensor.Tab)
	pl.SavePNG(core.Filename(fn + ".png"))
	pl.SaveSVG(core.Filename(fn + ".svg"))
}

// OpenCSV opens the Table data from a csv (comma-separated values) file (or any delim)
func (pl *PlotEditor) OpenCSV(filename core.Filename, delim tensor.Delims) { //types:add
	pl.table.OpenCSV(fsx.Filename(filename), delim)
	pl.dataFile = filename
	pl.UpdatePlot()
}

// OpenFS opens the Table data from a csv (comma-separated values) file (or any delim)
// from the given filesystem.
func (pl *PlotEditor) OpenFS(fsys fs.FS, filename core.Filename, delim tensor.Delims) {
	pl.table.OpenFS(fsys, string(filename), delim)
	pl.dataFile = filename
	pl.UpdatePlot()
}

// yLabel returns the Y-axis label
func (pl *PlotEditor) yLabel() string {
	if pl.Options.YAxisLabel != "" {
		return pl.Options.YAxisLabel
	}
	for _, cp := range pl.Columns {
		if cp.On {
			return cp.getLabel()
		}
	}
	return "Y"
}

// xLabel returns the X-axis label
func (pl *PlotEditor) xLabel() string {
	if pl.Options.XAxisLabel != "" {
		return pl.Options.XAxisLabel
	}
	if pl.Options.XAxis != "" {
		cp := pl.ColumnOptions(pl.Options.XAxis)
		if cp != nil {
			return cp.getLabel()
		}
		return pl.Options.XAxis
	}
	return "X"
}

// GoUpdatePlot updates the display based on current Indexed view into table.
// This version can be called from goroutines. It does Sequential() on
// the [table.Table], under the assumption that it is used for tracking a
// the latest updates of a running process.
func (pl *PlotEditor) GoUpdatePlot() {
	if pl == nil || pl.This == nil {
		return
	}
	if core.TheApp.Platform() == system.Web {
		time.Sleep(time.Millisecond) // critical to prevent hanging!
	}
	if !pl.IsVisible() || pl.table == nil || pl.inPlot {
		return
	}
	pl.Scene.AsyncLock()
	pl.table.Sequential()
	pl.genPlot()
	pl.NeedsRender()
	pl.Scene.AsyncUnlock()
}

// UpdatePlot updates the display based on current Indexed view into table.
// It does not automatically update the [table.Table] unless it is
// nil or out date.
func (pl *PlotEditor) UpdatePlot() {
	if pl == nil || pl.This == nil {
		return
	}
	if pl.table == nil || pl.inPlot {
		return
	}
	if len(pl.Children) != 2 || len(pl.Columns) != pl.table.NumColumns() {
		pl.Update()
	}
	if pl.table.NumRows() == 0 {
		pl.table.Sequential()
	}
	pl.genPlot()
}

// genPlot generates the plot and renders it to SVG
// It surrounds operation with InPlot true / false to prevent multiple updates
func (pl *PlotEditor) genPlot() {
	if pl.inPlot {
		slog.Error("plot: in plot already") // note: this never seems to happen -- could probably nuke
		return
	}
	pl.inPlot = true
	if pl.table == nil {
		pl.inPlot = false
		return
	}
	if len(pl.table.Indexes) == 0 {
		pl.table.Sequential()
	} else {
		lsti := pl.table.Indexes[pl.table.NumRows()-1]
		if lsti >= pl.table.NumRows() { // out of date
			pl.table.Sequential()
		}
	}
	pl.plot = nil
	switch pl.Options.Type {
	case XY:
		pl.genPlotXY()
	case Bar:
		pl.genPlotBar()
	}
	pl.plotWidget.Scale = pl.Options.Scale
	pl.plotWidget.SetRangesFunc = func() {
		plt := pl.plotWidget.Plot
		xi := pl.table.ColumnIndex(pl.Options.XAxis)
		if xi >= 0 {
			xp := pl.Columns[xi]
			if xp.Range.FixMin {
				plt.X.Min = math32.Min(plt.X.Min, float32(xp.Range.Min))
			}
			if xp.Range.FixMax {
				plt.X.Max = math32.Max(plt.X.Max, float32(xp.Range.Max))
			}
		}
		for _, cp := range pl.Columns { // key that this comes at the end, to actually stick
			if !cp.On || cp.IsString {
				continue
			}
			if cp.Range.FixMin {
				plt.Y.Min = math32.Min(plt.Y.Min, float32(cp.Range.Min))
			}
			if cp.Range.FixMax {
				plt.Y.Max = math32.Max(plt.Y.Max, float32(cp.Range.Max))
			}
		}
	}
	pl.plotWidget.SetPlot(pl.plot) // redraws etc
	pl.inPlot = false
}

// configPlot configures the given plot based on the plot options.
func (pl *PlotEditor) configPlot(plt *plot.Plot) {
	plt.Title.Text = pl.Options.Title
	plt.X.Label.Text = pl.xLabel()
	plt.Y.Label.Text = pl.yLabel()
	plt.Legend.Position = pl.Options.LegendPosition
	plt.X.TickText.Style.Rotation = float32(pl.Options.XAxisRotation)
}

// plotXAxis processes the XAxis and returns its index
func (pl *PlotEditor) plotXAxis(plt *plot.Plot, ixvw *table.Table) (xi int, xview *table.Table, err error) {
	xi = ixvw.ColumnIndex(pl.Options.XAxis)
	if xi < 0 {
		// log.Println("plot.PlotXAxis: " + err.Error())
		return
	}
	xview = ixvw
	xc := ixvw.ColumnByIndex(xi)
	xp := pl.Columns[xi]
	sz := 1
	if xp.Range.FixMin {
		plt.X.Min = math32.Min(plt.X.Min, float32(xp.Range.Min))
	}
	if xp.Range.FixMax {
		plt.X.Max = math32.Max(plt.X.Max, float32(xp.Range.Max))
	}
	if xc.Tensor.NumDims() > 1 {
		sz = xc.NumRows() / xc.Tensor.DimSize(0)
		if xp.TensorIndex > sz || xp.TensorIndex < 0 {
			slog.Error("plotcore.PlotEditor.plotXAxis: TensorIndex invalid -- reset to 0")
			xp.TensorIndex = 0
		}
	}
	return
}

const plotColumnsHeaderN = 2

// columnsListUpdate updates the list of columns
func (pl *PlotEditor) columnsListUpdate() {
	if pl.table == nil {
		pl.Columns = nil
		return
	}
	dt := pl.table
	nc := dt.NumColumns()
	if nc == len(pl.Columns) {
		return
	}
	pl.Columns = make([]*ColumnOptions, nc)
	clri := 0
	hasOn := false
	for ci := range dt.NumColumns() {
		cn := dt.ColumnName(ci)
		if pl.Options.XAxis == "" && ci == 0 {
			pl.Options.XAxis = cn // x-axis defaults to the first column
		}
		cp := &ColumnOptions{Column: cn}
		cp.defaults()
		tcol := dt.ColumnByIndex(ci)
		tc := tcol.Tensor
		if tc.IsString() {
			cp.IsString = true
		} else {
			cp.IsString = false
			// we enable the first non-string, non-x-axis, non-first column by default
			if !hasOn && cn != pl.Options.XAxis && ci != 0 {
				cp.On = true
				hasOn = true
			}
		}
		cp.fromMetaMap(pl.table.Meta)
		inc := 1
		if cn == pl.Options.XAxis || tc.IsString() || tc.DataType() == reflect.Int || tc.DataType() == reflect.Int64 || tc.DataType() == reflect.Int32 || tc.DataType() == reflect.Uint8 {
			inc = 0
		}
		cp.Color = colors.Uniform(colors.Spaced(clri))
		pl.Columns[ci] = cp
		clri += inc
	}
}

// ColumnsFromMetaMap updates all the column settings from given meta map
func (pl *PlotEditor) ColumnsFromMetaMap(meta metadata.Data) {
	for _, cp := range pl.Columns {
		cp.fromMetaMap(meta)
	}
}

// setAllColumns turns all Columns on or off (except X axis)
func (pl *PlotEditor) setAllColumns(on bool) {
	fr := pl.columnsFrame
	for i, cli := range fr.Children {
		if i < plotColumnsHeaderN {
			continue
		}
		ci := i - plotColumnsHeaderN
		cp := pl.Columns[ci]
		if cp.Column == pl.Options.XAxis {
			continue
		}
		cp.On = on
		cl := cli.(*core.Frame)
		sw := cl.Child(0).(*core.Switch)
		sw.SetChecked(cp.On)
	}
	pl.UpdatePlot()
	pl.NeedsRender()
}

// setColumnsByName turns columns on or off if their name contains
// the given string.
func (pl *PlotEditor) setColumnsByName(nameContains string, on bool) { //types:add
	fr := pl.columnsFrame
	for i, cli := range fr.Children {
		if i < plotColumnsHeaderN {
			continue
		}
		ci := i - plotColumnsHeaderN
		cp := pl.Columns[ci]
		if cp.Column == pl.Options.XAxis {
			continue
		}
		if !strings.Contains(cp.Column, nameContains) {
			continue
		}
		cp.On = on
		cl := cli.(*core.Frame)
		sw := cl.Child(0).(*core.Switch)
		sw.SetChecked(cp.On)
	}
	pl.UpdatePlot()
	pl.NeedsRender()
}

// makeColumns makes the Plans for columns
func (pl *PlotEditor) makeColumns(p *tree.Plan) {
	pl.columnsListUpdate()
	tree.Add(p, func(w *core.Frame) {
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Clear").SetIcon(icons.ClearAll).SetType(core.ButtonAction)
			w.SetTooltip("Turn all columns off")
			w.OnClick(func(e events.Event) {
				pl.setAllColumns(false)
			})
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Search").SetIcon(icons.Search).SetType(core.ButtonAction)
			w.SetTooltip("Select columns by column name")
			w.OnClick(func(e events.Event) {
				core.CallFunc(pl, pl.setColumnsByName)
			})
		})
	})
	tree.Add(p, func(w *core.Separator) {})
	for _, cp := range pl.Columns {
		tree.AddAt(p, cp.Column, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.CenterAll()
			})
			tree.AddChild(w, func(w *core.Switch) {
				w.SetType(core.SwitchCheckbox).SetTooltip("Turn this column on or off")
				w.OnChange(func(e events.Event) {
					cp.On = w.IsChecked()
					pl.UpdatePlot()
				})
				w.Updater(func() {
					xaxis := cp.Column == pl.Options.XAxis || cp.Column == pl.Options.Legend
					w.SetState(xaxis, states.Disabled, states.Indeterminate)
					if xaxis {
						cp.On = false
					} else {
						w.SetChecked(cp.On)
					}
				})
			})
			tree.AddChild(w, func(w *core.Button) {
				w.SetText(cp.Column).SetType(core.ButtonAction).SetTooltip("Edit column options including setting it as the x-axis or legend")
				w.OnClick(func(e events.Event) {
					update := func() {
						if core.TheApp.Platform().IsMobile() {
							pl.Update()
							return
						}
						// we must be async on multi-window platforms since
						// it is coming from a separate window
						pl.AsyncLock()
						pl.Update()
						pl.AsyncUnlock()
					}
					d := core.NewBody("Column options")
					core.NewForm(d).SetStruct(cp).
						OnChange(func(e events.Event) {
							update()
						})
					d.AddTopBar(func(bar *core.Frame) {
						core.NewToolbar(bar).Maker(func(p *tree.Plan) {
							tree.Add(p, func(w *core.Button) {
								w.SetText("Set x-axis").OnClick(func(e events.Event) {
									pl.Options.XAxis = cp.Column
									update()
								})
							})
							tree.Add(p, func(w *core.Button) {
								w.SetText("Set legend").OnClick(func(e events.Event) {
									pl.Options.Legend = cp.Column
									update()
								})
							})
						})
					})
					d.RunWindowDialog(pl)
				})
			})
		})
	}
}

func (pl *PlotEditor) MakeToolbar(p *tree.Plan) {
	if pl.table == nil {
		return
	}
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.PanTool).
			SetTooltip("toggle the ability to zoom and pan the view").OnClick(func(e events.Event) {
			pw := pl.plotWidget
			pw.SetReadOnly(!pw.IsReadOnly())
			pw.Restyle()
		})
	})
	// tree.Add(p, func(w *core.Button) {
	// 	w.SetIcon(icons.ArrowForward).
	// 		SetTooltip("turn on select mode for selecting Plot elements").
	// 		OnClick(func(e events.Event) {
	// 			fmt.Println("this will select select mode")
	// 		})
	// })
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Update").SetIcon(icons.Update).
			SetTooltip("update fully redraws display, reflecting any new settings etc").
			OnClick(func(e events.Event) {
				pl.UpdateWidget()
				pl.UpdatePlot()
			})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Options").SetIcon(icons.Settings).
			SetTooltip("Options for how the plot is rendered").
			OnClick(func(e events.Event) {
				d := core.NewBody("Plot options")
				core.NewForm(d).SetStruct(&pl.Options).
					OnChange(func(e events.Event) {
						pl.GoUpdatePlot()
					})
				d.RunWindowDialog(pl)
			})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Table").SetIcon(icons.Edit).
			SetTooltip("open a Table window of the data").
			OnClick(func(e events.Event) {
				d := core.NewBody(pl.Name + " Data")
				tv := tensorcore.NewTable(d).SetTable(pl.table)
				d.AddTopBar(func(bar *core.Frame) {
					core.NewToolbar(bar).Maker(tv.MakeToolbar)
				})
				d.RunWindowDialog(pl)
			})
	})
	tree.Add(p, func(w *core.Separator) {})

	tree.Add(p, func(w *core.Button) {
		w.SetText("Save").SetIcon(icons.Save).SetMenu(func(m *core.Scene) {
			core.NewFuncButton(m).SetFunc(pl.SaveSVG).SetIcon(icons.Save)
			core.NewFuncButton(m).SetFunc(pl.SavePNG).SetIcon(icons.Save)
			core.NewFuncButton(m).SetFunc(pl.SaveCSV).SetIcon(icons.Save)
			core.NewSeparator(m)
			core.NewFuncButton(m).SetFunc(pl.SaveAll).SetIcon(icons.Save)
		})
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(pl.OpenCSV).SetIcon(icons.Open)
	})
	tree.Add(p, func(w *core.Separator) {})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(pl.table.FilterString).SetText("Filter").SetIcon(icons.FilterAlt)
		w.SetAfterFunc(pl.UpdatePlot)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(pl.table.Sequential).SetText("Unfilter").SetIcon(icons.FilterAltOff)
		w.SetAfterFunc(pl.UpdatePlot)
	})
}

func (pt *PlotEditor) SizeFinal() {
	pt.Frame.SizeFinal()
	pt.UpdatePlot()
}

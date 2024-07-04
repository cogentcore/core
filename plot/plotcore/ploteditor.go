// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package plotcore provides Cogent Core widgets for viewing and editing plots.
package plotcore

//go:generate core generate

import (
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorcore"
	"cogentcore.org/core/tree"
)

// PlotEditor is a widget that provides an interactive 2D plot
// of selected columns of tabular data, represented by a [table.IndexView] into
// a [table.Table]. Other types of tabular data can be converted into this format.
// The user can change various options for the plot and also modify the underlying data.
type PlotEditor struct { //types:add
	core.Frame

	// Table is the table of data being plotted.
	Table *table.IndexView `set:"-"`

	// Options are the overall plot options.
	Options PlotOptions

	// Columns are the options for each column of the table.
	Columns []*ColumnOptions `set:"-"`

	// Plot is the plot object.
	Plot *plot.Plot `set:"-" edit:"-" json:"-" xml:"-"`

	// ConfigPlotFunc is a function to call to configure [PlotEditor.Plot], the plot.Plot that
	// actually does the plotting. It is called after [Plot] is generated, and properties
	// of [Plot] can be modified in it. Properties of [Plot] should not be modified outside
	// of this function, as doing so will have no effect.
	ConfigPlotFunc func() `json:"-" xml:"-"`

	// current svg file
	SVGFile core.Filename

	// current csv data file
	DataFile core.Filename

	// currently doing a plot
	InPlot bool `set:"-" edit:"-" json:"-" xml:"-"`
}

func (pl *PlotEditor) CopyFieldsFrom(frm tree.Node) {
	fr := frm.(*PlotEditor)
	pl.Frame.CopyFieldsFrom(&fr.Frame)
	pl.Options.CopyFrom(&fr.Options)
	pl.SetIndexView(fr.Table)
	mx := min(len(pl.Columns), len(fr.Columns))
	for i := 0; i < mx; i++ {
		pl.Columns[i].CopyFrom(fr.Columns[i])
	}
}

// NewSubPlot returns a PlotEditor with its own separate Toolbar,
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

	pl.Options.Plot = pl
	pl.Options.Defaults()
	pl.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 1)
	})

	pl.OnShow(func(e events.Event) {
		pl.UpdatePlot()
	})

	pl.Updater(func() {
		pl.Options.FromMeta(pl.Table.Table)
	})
	tree.AddChildAt(pl, "cols", func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(0, 1)
			s.Overflow.Y = styles.OverflowAuto
			s.Background = colors.Scheme.SurfaceContainerLow
		})
		w.Maker(pl.makeColumns)
	})
	tree.AddChildAt(pl, "plot", func(w *Plot) {
		w.Plot = pl.Plot
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
	})
}

// SetIndexView sets the table to view and does Update
// to update the Column list, which will also trigger a Layout
// and updating of the plot on next render pass.
// This is safe to call from a different goroutine.
func (pl *PlotEditor) SetIndexView(tab *table.IndexView) *PlotEditor {
	pl.Table = tab
	pl.Update()
	return pl
}

// SetTable sets the table to view and does Update
// to update the Column list, which will also trigger a Layout
// and updating of the plot on next render pass.
// This is safe to call from a different goroutine.
func (pl *PlotEditor) SetTable(tab *table.Table) *PlotEditor {
	pl.Table = table.NewIndexView(tab)
	pl.Update()
	return pl
}

// ColumnOptionsTry returns the current column options by name
// (to access by index, just use Columns directly).
// It returns an error message if not found.
func (pl *PlotEditor) ColumnOptionsTry(column string) (*ColumnOptions, error) {
	for _, co := range pl.Columns {
		if co.Column == column {
			return co, nil
		}
	}
	return nil, fmt.Errorf("plot: %v column named: %v not found", pl.Name, column)
}

// ColumnOptions returns the current column options by name
// (to access by index, just use Columns directly).
// It returns nil if not found.
func (pl *PlotEditor) ColumnOptions(column string) *ColumnOptions {
	co, _ := pl.ColumnOptionsTry(column)
	return co
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
	cp, err := pl.ColumnOptionsTry(column)
	if errors.Log(err) != nil {
		return nil
	}
	cp.On = on
	cp.Range.FixMin = fixMin
	if fixMin {
		cp.Range.Min = min
	}
	cp.Range.FixMax = fixMax
	if fixMax {
		cp.Range.Max = max
	}
	return cp
}

// SaveSVG saves the plot to an svg -- first updates to ensure that plot is current
func (pl *PlotEditor) SaveSVG(fname core.Filename) { //types:add
	pl.UpdatePlot()
	// TODO: get plot svg saving working
	// pc := pl.PlotChild()
	// SaveSVGView(string(fname), pl.Plot, sv, 2)
	pl.SVGFile = fname
}

// SavePNG saves the current plot to a png, capturing current render
func (pl *PlotEditor) SavePNG(fname core.Filename) { //types:add
	pl.UpdatePlot()
	imagex.Save(pl.Plot.Pixels, string(fname))
}

// SaveCSV saves the Table data to a csv (comma-separated values) file with headers (any delim)
func (pl *PlotEditor) SaveCSV(fname core.Filename, delim table.Delims) { //types:add
	pl.Table.SaveCSV(fname, delim, table.Headers)
	pl.DataFile = fname
}

// SaveAll saves the current plot to a png, svg, and the data to a tsv -- full save
// Any extension is removed and appropriate extensions are added
func (pl *PlotEditor) SaveAll(fname core.Filename) { //types:add
	fn := string(fname)
	fn = strings.TrimSuffix(fn, filepath.Ext(fn))
	pl.SaveCSV(core.Filename(fn+".tsv"), table.Tab)
	pl.SavePNG(core.Filename(fn + ".png"))
	pl.SaveSVG(core.Filename(fn + ".svg"))
}

// OpenCSV opens the Table data from a csv (comma-separated values) file (or any delim)
func (pl *PlotEditor) OpenCSV(filename core.Filename, delim table.Delims) { //types:add
	pl.Table.Table.OpenCSV(filename, delim)
	pl.DataFile = filename
	pl.UpdatePlot()
}

// OpenFS opens the Table data from a csv (comma-separated values) file (or any delim)
// from the given filesystem.
func (pl *PlotEditor) OpenFS(fsys fs.FS, filename core.Filename, delim table.Delims) {
	pl.Table.Table.OpenFS(fsys, string(filename), delim)
	pl.DataFile = filename
	pl.UpdatePlot()
}

// YLabel returns the Y-axis label
func (pl *PlotEditor) YLabel() string {
	if pl.Options.YAxisLabel != "" {
		return pl.Options.YAxisLabel
	}
	for _, cp := range pl.Columns {
		if cp.On {
			return cp.GetLabel()
		}
	}
	return "Y"
}

// XLabel returns the X-axis label
func (pl *PlotEditor) XLabel() string {
	if pl.Options.XAxisLabel != "" {
		return pl.Options.XAxisLabel
	}
	if pl.Options.XAxisColumn != "" {
		cp := pl.ColumnOptions(pl.Options.XAxisColumn)
		if cp != nil {
			return cp.GetLabel()
		}
		return pl.Options.XAxisColumn
	}
	return "X"
}

// GoUpdatePlot updates the display based on current IndexView into table.
// this version can be called from go routines. It does Sequential() on
// the [table.IndexView], under the assumption that it is used for tracking a
// the latest updates of a running process.
func (pl *PlotEditor) GoUpdatePlot() {
	if pl == nil || pl.This == nil {
		return
	}
	if !pl.IsVisible() || pl.Table == nil || pl.Table.Table == nil || pl.InPlot {
		return
	}
	pl.Scene.AsyncLock()
	pl.Table.Sequential()
	pl.GenPlot()
	pl.NeedsRender()
	pl.Scene.AsyncUnlock()
}

// UpdatePlot updates the display based on current IndexView into table.
// This version can only be called within main goroutine for
// window eventloop -- use GoUpdateUplot for other-goroutine updates.
// It does not automatically update the [table.IndexView] unless it is
// nil or out date.
func (pl *PlotEditor) UpdatePlot() {
	if pl == nil || pl.This == nil {
		return
	}
	if pl.Table == nil || pl.Table.Table == nil || pl.InPlot {
		return
	}
	if len(pl.Children) != 2 || len(pl.Columns) != pl.Table.Table.NumColumns() {
		pl.Update()
	}
	if pl.Table.Len() == 0 {
		pl.Table.Sequential()
	}
	pl.GenPlot()
}

// GenPlot generates the plot and renders it to SVG
// It surrounds operation with InPlot true / false to prevent multiple updates
func (pl *PlotEditor) GenPlot() {
	if pl.InPlot {
		slog.Error("plot: in plot already") // note: this never seems to happen -- could probably nuke
		return
	}
	pl.InPlot = true
	if pl.Table == nil || pl.Table.Table.NumRows() == 0 {
		// sv.DeleteChildren()
		pl.InPlot = false
		return
	}
	lsti := pl.Table.Indexes[pl.Table.Len()-1]
	if lsti >= pl.Table.Table.Rows { // out of date
		pl.Table.Sequential()
	}
	pl.Plot = nil
	switch pl.Options.Type {
	case XY:
		pl.GenPlotXY()
	case Bar:
		pl.GenPlotBar()
	}
	pl.PlotChild().Scale = pl.Options.Scale
	pl.PlotChild().SetPlot(pl.Plot) // redraws etc
	pl.InPlot = false
}

// ConfigPlot configures the given plot based on the plot options.
func (pl *PlotEditor) ConfigPlot(plt *plot.Plot) {
	plt.Title.Text = pl.Options.Title
	plt.X.Label.Text = pl.XLabel()
	plt.Y.Label.Text = pl.YLabel()

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

	plt.Legend.Position = pl.Options.LegendPosition
	plt.X.TickText.Style.Rotation = float32(pl.Options.XAxisRotation)
}

// PlotXAxis processes the XAxis and returns its index
func (pl *PlotEditor) PlotXAxis(plt *plot.Plot, ixvw *table.IndexView) (xi int, xview *table.IndexView, err error) {
	xi, err = ixvw.Table.ColumnIndexTry(pl.Options.XAxisColumn)
	if err != nil {
		// log.Println("plot.PlotXAxis: " + err.Error())
		return
	}
	xview = ixvw
	xc := ixvw.Table.Columns[xi]
	xp := pl.Columns[xi]
	sz := 1
	if xp.Range.FixMin {
		plt.X.Min = math32.Min(plt.X.Min, float32(xp.Range.Min))
	}
	if xp.Range.FixMax {
		plt.X.Max = math32.Max(plt.X.Max, float32(xp.Range.Max))
	}
	if xc.NumDims() > 1 {
		sz = xc.Len() / xc.DimSize(0)
		if xp.TensorIndex > sz || xp.TensorIndex < 0 {
			log.Printf("eplot.PlotXAxis: TensorIndex invalid -- reset to 0")
			xp.TensorIndex = 0
		}
	}
	return
}

func (pl *PlotEditor) ColumnsFrame() *core.Frame {
	return pl.ChildByName("cols", 0).(*core.Frame)
}

func (pl *PlotEditor) PlotChild() *Plot {
	return pl.ChildByName("plot", 1).(*Plot)
}

const PlotColumnsHeaderN = 2

// ColumnsListUpdate updates the list of columns
func (pl *PlotEditor) ColumnsListUpdate() {
	if pl.Table == nil || pl.Table.Table == nil {
		pl.Columns = nil
		return
	}
	dt := pl.Table.Table
	nc := dt.NumColumns()
	if nc == len(pl.Columns) {
		return
	}
	pl.Columns = make([]*ColumnOptions, nc)
	clri := 0
	for ci := range dt.NumColumns() {
		cn := dt.ColumnName(ci)
		cp := &ColumnOptions{Column: cn}
		cp.Defaults()
		tcol := dt.Columns[ci]
		if tcol.IsString() {
			cp.IsString = true
		} else {
			cp.IsString = false
		}
		cp.FromMetaMap(pl.Table.Table.MetaData)
		inc := 1
		if cn == pl.Options.XAxisColumn || tcol.IsString() || tcol.DataType() == reflect.Int || tcol.DataType() == reflect.Int64 || tcol.DataType() == reflect.Int32 || tcol.DataType() == reflect.Uint8 {
			inc = 0
		}
		cp.Color = colors.Uniform(colors.Spaced(clri))
		pl.Columns[ci] = cp
		clri += inc
	}
}

// ColumnsFromMetaMap updates all the column settings from given meta map
func (pl *PlotEditor) ColumnsFromMetaMap(meta map[string]string) {
	for _, cp := range pl.Columns {
		cp.FromMetaMap(meta)
	}
}

// SetAllColumns turns all Columns on or off (except X axis)
func (pl *PlotEditor) SetAllColumns(on bool) {
	fr := pl.ColumnsFrame()
	for i, cli := range fr.Children {
		if i < PlotColumnsHeaderN {
			continue
		}
		ci := i - PlotColumnsHeaderN
		cp := pl.Columns[ci]
		if cp.Column == pl.Options.XAxisColumn {
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

// SetColumnsByName turns cols On or Off if their name contains given string
func (pl *PlotEditor) SetColumnsByName(nameContains string, on bool) { //types:add
	fr := pl.ColumnsFrame()
	for i, cli := range fr.Children {
		if i < PlotColumnsHeaderN {
			continue
		}
		ci := i - PlotColumnsHeaderN
		cp := pl.Columns[ci]
		if cp.Column == pl.Options.XAxisColumn {
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
	pl.ColumnsListUpdate()
	tree.Add(p, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Grow.Set(0, 0)
		})
		tree.AddChild(w, func(w *core.Switch) {
			w.SetType(core.SwitchCheckbox).SetTooltip("Toggle off all columns")
			w.OnChange(func(e events.Event) {
				w.SetChecked(false)
				pl.SetAllColumns(false)
			})
		})
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Select Columns").SetType(core.ButtonAction).
				SetTooltip("click to select columns based on column name").
				OnClick(func(e events.Event) {
					core.CallFunc(pl, pl.SetColumnsByName)
				})
		})
	})
	tree.Add(p, func(w *core.Separator) {})
	for _, cp := range pl.Columns {
		cp.Plot = pl
		tree.AddAt(p, cp.Column, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Row
				s.Grow.Set(0, 0)
			})
			tree.AddChild(w, func(w *core.Switch) {
				w.SetType(core.SwitchCheckbox).SetTooltip("toggle plot on")
				w.OnChange(func(e events.Event) {
					cp.On = w.StateIs(states.Checked)
					pl.UpdatePlot()
				})
				w.Updater(func() {
					w.SetState(cp.On, states.Checked)
				})
			})
			tree.AddChild(w, func(w *core.Button) {
				w.SetText(cp.Column).SetType(core.ButtonAction).SetTooltip("Edit column options including setting it as the x-axis or legend")
				w.OnClick(func(e events.Event) {
					d := core.NewBody().AddTitle("Column options")
					core.NewForm(d).SetStruct(cp).
						OnChange(func(e events.Event) {
							pl.GoUpdatePlot() // note: because this is a separate window, need "Go" version
						})
					d.AddAppBar(func(p *tree.Plan) {
						tree.Add(p, func(w *core.Button) {
							w.SetText("Set X Axis").OnClick(func(e events.Event) {
								pl.Options.XAxisColumn = cp.Column
								pl.UpdatePlot()
							})
						})
						tree.Add(p, func(w *core.Button) {
							w.SetText("Set Legend").OnClick(func(e events.Event) {
								pl.Options.LegendColumn = cp.Column
								pl.UpdatePlot()
							})
						})
					})
					d.NewFullDialog(pl).SetNewWindow(true).Run()
				})
			})
		})
	}
}

func (pl *PlotEditor) MakeToolbar(p *tree.Plan) {
	if pl.Table == nil {
		return
	}
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.PanTool).
			SetTooltip("toggle the ability to zoom and pan the view").OnClick(func(e events.Event) {
			pc := pl.PlotChild()
			pc.SetReadOnly(!pc.IsReadOnly())
			pc.Restyle()
		})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.ArrowForward).
			SetTooltip("turn on select mode for selecting Plot elements").
			OnClick(func(e events.Event) {
				fmt.Println("this will select select mode")
			})
	})
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
				d := core.NewBody().AddTitle("Plot options")
				core.NewForm(d).SetStruct(&pl.Options).
					OnChange(func(e events.Event) {
						pl.GoUpdatePlot() // note: because this is a separate window, need "Go" version
					})
				d.NewFullDialog(pl).SetNewWindow(true).Run()
			})
	})
	tree.Add(p, func(w *core.Button) {
		w.SetText("Table").SetIcon(icons.Edit).
			SetTooltip("open a Table window of the data").
			OnClick(func(e events.Event) {
				d := core.NewBody().AddTitle(pl.Name + " Data")
				tv := tensorcore.NewTable(d).SetTable(pl.Table.Table)
				d.AddAppBar(tv.MakeToolbar)
				d.NewFullDialog(pl).SetNewWindow(true).Run()
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
		w.SetFunc(pl.Table.FilterColumnName).SetText("Filter").SetIcon(icons.FilterAlt)
		w.SetAfterFunc(pl.UpdatePlot)
	})
	tree.Add(p, func(w *core.FuncButton) {
		w.SetFunc(pl.Table.Sequential).SetText("Unfilter").SetIcon(icons.FilterAltOff)
		w.SetAfterFunc(pl.UpdatePlot)
	})
}

func (pt *PlotEditor) SizeFinal() {
	pt.Frame.SizeFinal()
	pt.UpdatePlot()
}

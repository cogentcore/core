// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotview

//go:generate core generate

import (
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"

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
	"cogentcore.org/core/tensor/tensorview"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

// PlotView is a Cogent Core Widget that provides an interactive 2D plot
// of selected columns of Tabular data, represented by an IndexView into
// a table.Table.  Other types of tabular data can be converted into this format.
type PlotView struct { //types:add
	core.Layout

	// the table of data being plotted
	Table *table.IndexView `set:"-"`

	// the overall plot parameters
	Params PlotParams

	// the parameters for each column of the table
	Columns []*ColumnParams `set:"-"`

	// the plot object
	Plot *plot.Plot `set:"-" edit:"-" json:"-" xml:"-"`

	// ConfigPlotFunc is a function to call to configure [PlotView.Plot], the plot.Plot that
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

func (pl *PlotView) CopyFieldsFrom(frm tree.Node) {
	fr := frm.(*PlotView)
	pl.Layout.CopyFieldsFrom(&fr.Layout)
	pl.Params.CopyFrom(&fr.Params)
	pl.SetTableView(fr.Table)
	mx := min(len(pl.Columns), len(fr.Columns))
	for i := 0; i < mx; i++ {
		pl.Columns[i].CopyFrom(fr.Columns[i])
	}
}

// NewSubPlot returns a PlotView with its own separate Toolbar,
// suitable for a tab or other element that is not the main plot.
func NewSubPlot(par core.Widget, name ...string) *PlotView {
	fr := core.NewFrame(par, name...)
	tb := core.NewToolbar(fr, "tbar")
	pl := NewPlotView(fr, "plot")
	fr.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	tb.ToolbarFuncs.Add(pl.ConfigToolbar)
	return pl
}

func (pl *PlotView) OnInit() {
	pl.Params.Plot = pl
	pl.Params.Defaults()
	pl.Style(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 1)
	})
}

func (pl *PlotView) OnAdd() {
	pl.Layout.OnAdd()
	pl.OnShow(func(e events.Event) {
		pl.UpdatePlot()
	})
}

// SetTableView sets the table to view and does Update
// to update the Column list, which will also trigger a Layout
// and updating of the plot on next render pass.
// This is safe to call from a different goroutine.
func (pl *PlotView) SetTableView(tab *table.IndexView) *PlotView {
	pl.Table = tab
	pl.DeleteColumns()
	pl.Update()
	return pl
}

// SetTable sets the table to view and does Update
// to update the Column list, which will also trigger a Layout
// and updating of the plot on next render pass.
// This is safe to call from a different goroutine.
func (pl *PlotView) SetTable(tab *table.Table) *PlotView {
	pl.Table = table.NewIndexView(tab)
	pl.DeleteColumns()
	pl.Update()
	return pl
}

// ColParamsTry returns the current column parameters by name (to access by index, just use Columns directly)
// Try version returns error message if not found.
func (pl *PlotView) ColParamsTry(colNm string) (*ColumnParams, error) {
	for _, cp := range pl.Columns {
		if cp.Column == colNm {
			return cp, nil
		}
	}
	return nil, fmt.Errorf("plot: %v column named: %v not found", pl.Nm, colNm)
}

// ColParams returns the current column parameters by name (to access by index, just use Columns directly)
// returns nil if not found
func (pl *PlotView) ColumnParams(colNm string) *ColumnParams {
	cp, _ := pl.ColParamsTry(colNm)
	return cp
}

// use these for SetColParams args
const (
	On       bool = true
	Off           = false
	FixMin        = true
	FloatMin      = false
	FixMax        = true
	FloatMax      = false
)

// SetColParams sets main parameters for one column
func (pl *PlotView) SetColParams(colNm string, on bool, fixMin bool, min float32, fixMax bool, max float32) *ColumnParams {
	cp, err := pl.ColParamsTry(colNm)
	if err != nil {
		log.Println(err)
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
func (pl *PlotView) SaveSVG(fname core.Filename) { //types:add
	pl.UpdatePlot()
	// pc := pl.PlotChild()
	// SaveSVGView(string(fname), pl.Plot, sv, 2)
	pl.SVGFile = fname
}

// SavePNG saves the current plot to a png, capturing current render
func (pl *PlotView) SavePNG(fname core.Filename) { //types:add
	pl.UpdatePlot()
	imagex.Save(pl.Plot.Pixels, string(fname))
}

// SaveCSV saves the Table data to a csv (comma-separated values) file with headers (any delim)
func (pl *PlotView) SaveCSV(fname core.Filename, delim table.Delims) { //types:add
	pl.Table.SaveCSV(fname, delim, table.Headers)
	pl.DataFile = fname
}

// SaveAll saves the current plot to a png, svg, and the data to a tsv -- full save
// Any extension is removed and appropriate extensions are added
func (pl *PlotView) SaveAll(fname core.Filename) { //types:add
	fn := string(fname)
	fn = strings.TrimSuffix(fn, filepath.Ext(fn))
	pl.SaveCSV(core.Filename(fn+".tsv"), table.Tab)
	pl.SavePNG(core.Filename(fn + ".png"))
	pl.SaveSVG(core.Filename(fn + ".svg"))
}

// OpenCSV opens the Table data from a csv (comma-separated values) file (or any delim)
func (pl *PlotView) OpenCSV(filename core.Filename, delim table.Delims) { //types:add
	pl.Table.Table.OpenCSV(filename, delim)
	pl.DataFile = filename
	pl.UpdatePlot()
}

// OpenFS opens the Table data from a csv (comma-separated values) file (or any delim)
// from the given filesystem.
func (pl *PlotView) OpenFS(fsys fs.FS, filename core.Filename, delim table.Delims) {
	pl.Table.Table.OpenFS(fsys, string(filename), delim)
	pl.DataFile = filename
	pl.UpdatePlot()
}

// YLabel returns the Y-axis label
func (pl *PlotView) YLabel() string {
	if pl.Params.YAxisLabel != "" {
		return pl.Params.YAxisLabel
	}
	for _, cp := range pl.Columns {
		if cp.On {
			return cp.Label()
		}
	}
	return "Y"
}

// XLabel returns the X-axis label
func (pl *PlotView) XLabel() string {
	if pl.Params.XAxisLabel != "" {
		return pl.Params.XAxisLabel
	}
	if pl.Params.XAxisColumn != "" {
		cp := pl.ColumnParams(pl.Params.XAxisColumn)
		if cp != nil {
			return cp.Label()
		}
		return pl.Params.XAxisColumn
	}
	return "X"
}

// GoUpdatePlot updates the display based on current IndexView into table.
// this version can be called from go routines.
func (pl *PlotView) GoUpdatePlot() {
	if pl == nil || pl.This() == nil {
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
func (pl *PlotView) UpdatePlot() {
	if pl == nil || pl.This() == nil {
		return
	}
	if pl.Table == nil || pl.Table.Table == nil || pl.InPlot {
		return
	}
	if len(pl.Kids) != 2 || len(pl.Columns) != pl.Table.Table.NumColumns() {
		pl.Update()
	}
	pl.Table.Sequential()
	pl.GenPlot()
}

// GenPlot generates the plot and renders it to SVG
// It surrounds operation with InPlot true / false to prevent multiple updates
func (pl *PlotView) GenPlot() {
	if pl.InPlot {
		slog.Error("plot: in plot already")
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
	switch pl.Params.Type {
	case XY:
		pl.GenPlotXY()
	case Bar:
		pl.GenPlotBar()
	}
	pl.PlotChild().Scale = pl.Params.Scale
	pl.PlotChild().SetPlot(pl.Plot) // redraws etc
	pl.InPlot = false
}

// ConfigPlot configures the plot with params
func (pl *PlotView) ConfigPlot(plt *plot.Plot) {
	plt.Title.Text = pl.Params.Title
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

	plt.Legend.Position = pl.Params.LegendPosition
	plt.X.TickText.Style.Rotation = float32(pl.Params.XAxisRotation)
}

// PlotXAxis processes the XAxis and returns its index
func (pl *PlotView) PlotXAxis(plt *plot.Plot, ixvw *table.IndexView) (xi int, xview *table.IndexView, err error) {
	xi, err = ixvw.Table.ColumnIndexTry(pl.Params.XAxisColumn)
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

func (pl *PlotView) Config() {
	if pl.Table != nil {
		pl.ConfigPlotView()
	}
}

// ConfigPlotView configures the overall view widget
func (pl *PlotView) ConfigPlotView() {
	pl.Params.FromMeta(pl.Table.Table)
	if !pl.HasChildren() {
		fr := core.NewFrame(pl, "cols")
		fr.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(0, 1)
			s.Overflow.Y = styles.OverflowAuto
			s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
		})
		pt := NewPlot(pl, "plot")
		pt.Plot = pl.Plot
		pt.Style(func(s *styles.Style) {
			s.Grow.Set(1, 1)
		})
	}

	pl.ColumnsConfig()
	pl.NeedsLayout()
}

// DeleteColumns deletes any existing cols, to ensure an update to new table
func (pl *PlotView) DeleteColumns() {
	pl.Columns = nil
	if pl.HasChildren() {
		vl := pl.ColumnsLay()
		vl.DeleteChildren()
	}
}

func (pl *PlotView) ColumnsLay() *core.Frame {
	return pl.ChildByName("cols", 0).(*core.Frame)
}

func (pl *PlotView) PlotChild() *Plot {
	return pl.ChildByName("plot", 1).(*Plot)
}

const PlotColumnsHeaderN = 2

// ColumnsListUpdate updates the list of columns
func (pl *PlotView) ColumnsListUpdate() {
	if pl.Table == nil || pl.Table.Table == nil {
		pl.Columns = nil
		return
	}
	dt := pl.Table.Table
	nc := dt.NumColumns()
	if nc == len(pl.Columns) {
		return
	}
	pl.Columns = make([]*ColumnParams, nc)
	clri := 0
	for ci := range dt.NumColumns() {
		cn := dt.ColumnName(ci)
		cp := &ColumnParams{Column: cn}
		cp.Defaults()
		tcol := dt.Columns[ci]
		if tcol.IsString() {
			cp.IsString = true
		} else {
			cp.IsString = false
		}
		cp.FromMetaMap(pl.Table.Table.MetaData)
		inc := 1
		if cn == pl.Params.XAxisColumn || tcol.IsString() || tcol.DataType() == reflect.Int || tcol.DataType() == reflect.Int64 || tcol.DataType() == reflect.Int32 || tcol.DataType() == reflect.Uint8 {
			inc = 0
		}
		cp.Color = colors.Spaced(clri)
		pl.Columns[ci] = cp
		clri += inc
	}
}

// ColumnsFromMetaMap updates all the column settings from given meta map
func (pl *PlotView) ColumnsFromMetaMap(meta map[string]string) {
	for _, cp := range pl.Columns {
		cp.FromMetaMap(meta)
	}
}

// ColumnsUpdate updates the display toggles for all the cols
func (pl *PlotView) ColumnsUpdate() {
	vl := pl.ColumnsLay()
	for i, cli := range *vl.Children() {
		if i < PlotColumnsHeaderN {
			continue
		}
		ci := i - PlotColumnsHeaderN
		cp := pl.Columns[ci]
		cl := cli.(*core.Layout)
		sw := cl.Child(0).(*core.Switch)
		if sw.StateIs(states.Checked) != cp.On {
			sw.SetChecked(cp.On)
			sw.NeedsRender()
		}
	}
}

// SetAllColumns turns all Columns on or off (except X axis)
func (pl *PlotView) SetAllColumns(on bool) {
	vl := pl.ColumnsLay()
	for i, cli := range *vl.Children() {
		if i < PlotColumnsHeaderN {
			continue
		}
		ci := i - PlotColumnsHeaderN
		cp := pl.Columns[ci]
		if cp.Column == pl.Params.XAxisColumn {
			continue
		}
		cp.On = on
		cl := cli.(*core.Layout)
		sw := cl.Child(0).(*core.Switch)
		sw.SetChecked(cp.On)
	}
	pl.UpdatePlot()
	pl.NeedsRender()
}

// SetColumnsByName turns cols On or Off if their name contains given string
func (pl *PlotView) SetColumnsByName(nameContains string, on bool) { //types:add
	vl := pl.ColumnsLay()
	for i, cli := range *vl.Children() {
		if i < PlotColumnsHeaderN {
			continue
		}
		ci := i - PlotColumnsHeaderN
		cp := pl.Columns[ci]
		if cp.Column == pl.Params.XAxisColumn {
			continue
		}
		if !strings.Contains(cp.Column, nameContains) {
			continue
		}
		cp.On = on
		cl := cli.(*core.Layout)
		sw := cl.Child(0).(*core.Switch)
		sw.SetChecked(cp.On)
	}
	pl.UpdatePlot()
	pl.NeedsRender()
}

// ColumnsConfig configures the column gui buttons
func (pl *PlotView) ColumnsConfig() {
	vl := pl.ColumnsLay()
	pl.ColumnsListUpdate()
	if len(vl.Kids) == len(pl.Columns)+PlotColumnsHeaderN {
		pl.ColumnsUpdate()
		return
	}
	vl.DeleteChildren()
	if len(pl.Columns) == 0 {
		return
	}
	sc := core.NewLayout(vl, "sel-cols")
	sw := core.NewSwitch(sc, "on").SetTooltip("Toggle off all columns")
	sw.OnChange(func(e events.Event) {
		sw.SetChecked(false)
		pl.SetAllColumns(false)
	})
	core.NewButton(sc, "col").SetText("Select Columns").SetType(core.ButtonAction).
		SetTooltip("click to select columns based on column name").
		OnClick(func(e events.Event) {
			views.CallFunc(pl, pl.SetColumnsByName)
		})
	core.NewSeparator(vl, "sep")

	for _, cp := range pl.Columns {
		cp := cp
		cp.Plot = pl
		cl := core.NewLayout(vl, cp.Column)
		cl.Style(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Grow.Set(0, 0)
		})
		sw := core.NewSwitch(cl, "on").SetType(core.SwitchCheckbox).SetTooltip("toggle plot on")
		sw.OnChange(func(e events.Event) {
			cp.On = sw.StateIs(states.Checked)
			pl.UpdatePlot()
		})
		sw.SetState(cp.On, states.Checked)
		bt := core.NewButton(cl, "col").SetText(cp.Column).SetType(core.ButtonAction)
		bt.SetMenu(func(m *core.Scene) {
			core.NewButton(m, "set-x").SetText("Set X Axis").OnClick(func(e events.Event) {
				pl.Params.XAxisColumn = cp.Column
				pl.UpdatePlot()
			})
			core.NewButton(m, "set-legend").SetText("Set Legend").OnClick(func(e events.Event) {
				pl.Params.LegendColumn = cp.Column
				pl.UpdatePlot()
			})
			core.NewButton(m, "edit").SetText("Edit").OnClick(func(e events.Event) {
				d := core.NewBody().AddTitle("Column Params")
				views.NewStructView(d).SetStruct(cp).
					OnChange(func(e events.Event) {
						pl.GoUpdatePlot() // note: because this is a separate window, need "Go" version
					})
				d.NewFullDialog(pl).SetNewWindow(true).Run()
			})
		})
	}
}

func (pl *PlotView) ConfigToolbar(tb *core.Toolbar) {
	if pl.Table == nil {
		return
	}
	core.NewButton(tb).SetIcon(icons.PanTool).
		SetTooltip("toggle the ability to zoom and pan the view").OnClick(func(e events.Event) {
		pc := pl.PlotChild()
		pc.SetReadOnly(!pc.IsReadOnly())
		pc.ApplyStyleUpdate()
	})
	core.NewButton(tb).SetIcon(icons.ArrowForward).
		SetTooltip("turn on select mode for selecting Plot elements").
		OnClick(func(e events.Event) {
			fmt.Println("this will select select mode")
		})
	core.NewSeparator(tb)
	core.NewButton(tb).SetText("Update").SetIcon(icons.Update).
		SetTooltip("update fully redraws display, reflecting any new settings etc").
		OnClick(func(e events.Event) {
			pl.ConfigPlotView()
			pl.UpdatePlot()
		})
	core.NewButton(tb).SetText("Config").SetIcon(icons.Settings).
		SetTooltip("set parameters that control display (font size etc)").
		OnClick(func(e events.Event) {
			d := core.NewBody().AddTitle(pl.Nm + " Params")
			views.NewStructView(d).SetStruct(&pl.Params).
				OnChange(func(e events.Event) {
					pl.GoUpdatePlot() // note: because this is a separate window, need "Go" version
				})
			d.NewFullDialog(pl).SetNewWindow(true).Run()
		})
	core.NewButton(tb).SetText("Table").SetIcon(icons.Edit).
		SetTooltip("open a TableView window of the data").
		OnClick(func(e events.Event) {
			d := core.NewBody().AddTitle(pl.Nm + " Data")
			etv := tensorview.NewTableView(d).SetTable(pl.Table.Table)
			d.AddAppBar(etv.ConfigToolbar)
			d.NewFullDialog(pl).SetNewWindow(true).Run()
		})
	core.NewSeparator(tb)

	core.NewButton(tb).SetText("Save").SetIcon(icons.Save).SetMenu(func(m *core.Scene) {
		views.NewFuncButton(m, pl.SaveSVG).SetIcon(icons.Save)
		views.NewFuncButton(m, pl.SavePNG).SetIcon(icons.Save)
		views.NewFuncButton(m, pl.SaveCSV).SetIcon(icons.Save)
		core.NewSeparator(m)
		views.NewFuncButton(m, pl.SaveAll).SetIcon(icons.Save)
	})
	views.NewFuncButton(tb, pl.OpenCSV).SetIcon(icons.Open)
	core.NewSeparator(tb)
	views.NewFuncButton(tb, pl.Table.FilterColumnName).SetText("Filter").SetIcon(icons.FilterAlt)
	views.NewFuncButton(tb, pl.Table.Sequential).SetText("Unfilter").SetIcon(icons.FilterAltOff)
}

func (pt *PlotView) SizeFinal() {
	pt.Layout.SizeFinal()
	pt.UpdatePlot()
}
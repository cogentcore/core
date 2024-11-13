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
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plots"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorcore"
	"cogentcore.org/core/tree"
	"golang.org/x/exp/maps"
)

// PlotEditor is a widget that provides an interactive 2D plot
// of selected columns of tabular data, represented by a [table.Table] into
// a [table.Table]. Other types of tabular data can be converted into this format.
// The user can change various options for the plot and also modify the underlying data.
type PlotEditor struct { //types:add
	core.Frame

	// table is the table of data being plotted.
	table *table.Table

	// PlotStyle has the overall plot style parameters.
	PlotStyle plot.PlotStyle

	// plot is the plot object.
	plot *plot.Plot

	// current svg file
	svgFile core.Filename

	// current csv data file
	dataFile core.Filename

	// currently doing a plot
	inPlot bool

	columnsFrame      *core.Frame
	plotWidget        *Plot
	plotStyleModified map[string]bool
}

func (pl *PlotEditor) CopyFieldsFrom(frm tree.Node) {
	fr := frm.(*PlotEditor)
	pl.Frame.CopyFieldsFrom(&fr.Frame)
	pl.PlotStyle = fr.PlotStyle
	pl.setTable(fr.table)
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

	pl.PlotStyle.Defaults()

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
			pl.plotStyleFromTable(pl.table)
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

// SetSlice sets the table to a [table.NewSliceTable] from the given slice.
// Optional styler functions are used for each struct field in sequence,
// and any can contain global plot style.
func (pl *PlotEditor) SetSlice(sl any, stylers ...func(s *plot.Style)) *PlotEditor {
	dt, err := table.NewSliceTable(sl)
	errors.Log(err)
	if dt == nil {
		return nil
	}
	mx := min(dt.NumColumns(), len(stylers))
	for i := range mx {
		plot.SetStylersTo(dt.Columns.Values[i], plot.Stylers{stylers[i]})
	}
	return pl.SetTable(dt)
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
	if len(pl.Children) != 2 { // || len(pl.Columns) != pl.table.NumColumns() { // todo:
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
	var err error
	pl.plot, err = plot.NewTablePlot(pl.table)
	if err != nil {
		core.ErrorSnackbar(pl, err)
	}
	pl.plotWidget.SetPlot(pl.plot) // redraws etc
	pl.inPlot = false
}

const plotColumnsHeaderN = 3

// allColumnsOff turns all columns off.
func (pl *PlotEditor) allColumnsOff() {
	fr := pl.columnsFrame
	for i, cli := range fr.Children {
		if i < plotColumnsHeaderN {
			continue
		}
		cl := cli.(*core.Frame)
		sw := cl.Child(0).(*core.Switch)
		sw.SetChecked(false)
		sw.SendChange()
	}
	pl.Update()
}

// setColumnsByName turns columns on or off if their name contains
// the given string.
func (pl *PlotEditor) setColumnsByName(nameContains string, on bool) { //types:add
	fr := pl.columnsFrame
	for i, cli := range fr.Children {
		if i < plotColumnsHeaderN {
			continue
		}
		cl := cli.(*core.Frame)
		if !strings.Contains(cl.Name, nameContains) {
			continue
		}
		sw := cl.Child(0).(*core.Switch)
		sw.SetChecked(on)
		sw.SendChange()
	}
	pl.Update()
}

// makeColumns makes the Plans for columns
func (pl *PlotEditor) makeColumns(p *tree.Plan) {
	tree.Add(p, func(w *core.Frame) {
		tree.AddChild(w, func(w *core.Button) {
			w.SetText("Clear").SetIcon(icons.ClearAll).SetType(core.ButtonAction)
			w.SetTooltip("Turn all columns off")
			w.OnClick(func(e events.Event) {
				pl.allColumnsOff()
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
	if pl.table == nil {
		return
	}
	colorIdx := 0 // index for color sequence -- skips various types
	for ci, cl := range pl.table.Columns.Values {
		cnm := pl.table.Columns.Keys[ci]
		psty := plot.GetStylersFrom(cl)
		cst, mods := pl.defaultColumnStyle(cl, ci, &colorIdx, psty)
		updateStyle := func() {
			if len(mods) == 0 {
				return
			}
			mf := modFields(mods)
			sty := psty
			sty = append(sty, func(s *plot.Style) {
				errors.Log(reflectx.CopyFields(s, cst, mf...))
				errors.Log(reflectx.CopyFields(&s.Plot, &pl.PlotStyle, modFields(pl.plotStyleModified)...))
			})
			plot.SetStylersTo(cl, sty)
		}
		updateStyle()
		tree.AddAt(p, cnm, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.CenterAll()
			})
			tree.AddChild(w, func(w *core.Switch) {
				w.SetType(core.SwitchCheckbox).SetTooltip("Turn this column on or off")
				w.Styler(func(s *styles.Style) {
					s.Color = cst.Line.Color
				})
				tree.AddChildInit(w, "stack", func(w *core.Frame) {
					f := func(name string) {
						tree.AddChildInit(w, name, func(w *core.Icon) {
							w.Styler(func(s *styles.Style) {
								s.Color = cst.Line.Color
							})
						})
					}
					f("icon-on")
					f("icon-off")
					f("icon-indeterminate")
				})
				w.OnChange(func(e events.Event) {
					mods["On"] = true
					cst.On = w.IsChecked()
					updateStyle()
					pl.UpdatePlot()
				})
				w.Updater(func() {
					xaxis := cst.Role == plot.X //  || cp.Column == pl.Options.Legend
					w.SetState(xaxis, states.Disabled, states.Indeterminate)
					if xaxis {
						cst.On = false
					} else {
						w.SetChecked(cst.On)
					}
				})
			})
			tree.AddChild(w, func(w *core.Button) {
				w.SetText(cnm).SetType(core.ButtonAction).SetTooltip("Edit all styling options for this column")
				w.OnClick(func(e events.Event) {
					update := func() {
						updateStyle()
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
					d := core.NewBody(cnm + " style properties")
					fm := core.NewForm(d).SetStruct(cst)
					fm.Modified = mods
					fm.OnChange(func(e events.Event) {
						update()
					})
					// d.AddTopBar(func(bar *core.Frame) {
					// 	core.NewToolbar(bar).Maker(func(p *tree.Plan) {
					// 		tree.Add(p, func(w *core.Button) {
					// 			w.SetText("Set x-axis").OnClick(func(e events.Event) {
					// 				pl.Options.XAxis = cp.Column
					// 				update()
					// 			})
					// 		})
					// 		tree.Add(p, func(w *core.Button) {
					// 			w.SetText("Set legend").OnClick(func(e events.Event) {
					// 				pl.Options.Legend = cp.Column
					// 				update()
					// 			})
					// 		})
					// 	})
					// })
					d.RunWindowDialog(pl)
				})
			})
		})
	}
}

// defaultColumnStyle initializes the column style with any existing stylers
// plus additional general defaults, returning the initially modified field names.
func (pl *PlotEditor) defaultColumnStyle(cl tensor.Values, ci int, colorIdx *int, psty plot.Stylers) (*plot.Style, map[string]bool) {
	cst := &plot.Style{}
	cst.Defaults()
	if psty != nil {
		psty.Run(cst)
	}
	mods := map[string]bool{}
	isfloat := reflectx.KindIsFloat(cl.DataType())
	if cst.Plotter == "" {
		if isfloat {
			cst.Plotter = plot.PlotterName(plots.XYType)
			mods["Plotter"] = true
		} else if cl.IsString() {
			cst.Plotter = plot.PlotterName(plots.LabelsType)
			mods["Plotter"] = true
		}
	}
	if cst.Role == plot.NoRole {
		mods["Role"] = true
		if isfloat {
			cst.Role = plot.Y
		} else if cl.IsString() {
			cst.Role = plot.Label
		} else {
			cst.Role = plot.X
		}
	}
	if cst.Line.Color == colors.Scheme.OnSurface {
		if cst.Role == plot.Y && isfloat {
			spclr := colors.Uniform(colors.Spaced(*colorIdx))
			cst.Line.Color = spclr
			mods["Line.Color"] = true
			cst.Point.Color = spclr
			mods["Point.Color"] = true
			if cst.Plotter == plots.BarType {
				cst.Line.Fill = spclr
				mods["Line.Fill"] = true
			}
			(*colorIdx)++
		}
	}
	return cst, mods
}

func (pl *PlotEditor) plotStyleFromTable(dt *table.Table) {
	if pl.plotStyleModified != nil { // already set
		return
	}
	pst := &pl.PlotStyle
	mods := map[string]bool{}
	pl.plotStyleModified = mods
	tst := &plot.Style{}
	tst.Defaults()
	tst.Plot.Defaults()
	for _, cl := range pl.table.Columns.Values {
		stl := plot.GetStylersFrom(cl)
		if stl == nil {
			continue
		}
		stl.Run(tst)
	}
	*pst = tst.Plot
	if pst.PointsOn == plot.Default {
		pst.PointsOn = plot.Off
		mods["PointsOn"] = true
	}
	if pst.Title == "" {
		pst.Title = metadata.Name(pl.table)
		if pst.Title != "" {
			mods["Title"] = true
		}
	}
}

// modFields returns the modified fields as field paths using . separators
func modFields(mods map[string]bool) []string {
	fns := maps.Keys(mods)
	rf := make([]string, 0, len(fns))
	for _, f := range fns {
		if mods[f] == false {
			continue
		}
		fc := strings.ReplaceAll(f, " • ", ".")
		rf = append(rf, fc)
	}
	slices.Sort(rf)
	return rf
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
		w.SetText("Style").SetIcon(icons.Settings).
			SetTooltip("Style for how the plot is rendered").
			OnClick(func(e events.Event) {
				d := core.NewBody("Plot style")
				fm := core.NewForm(d).SetStruct(&pl.PlotStyle)
				fm.Modified = pl.plotStyleModified
				fm.OnChange(func(e events.Event) {
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

// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"image"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/datafs"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/texteditor/diffbrowser"
	"cogentcore.org/core/tree"
)

// Treer is an interface for getting the Root node as a DataTree struct.
type Treer interface {
	AsDataTree() *DataTree
}

// AsDataTree returns the given value as a [DataTree] if it has
// an AsDataTree() method, or nil otherwise.
func AsDataTree(n tree.Node) *DataTree {
	if t, ok := n.(Treer); ok {
		return t.AsDataTree()
	}
	return nil
}

// DataTree is the databrowser version of [filetree.Tree],
// which provides the Tabber to show data editors.
type DataTree struct {
	filetree.Tree

	// Tabber is the [Tabber] for this tree.
	Tabber Tabber
}

func (ft *DataTree) AsDataTree() *DataTree {
	return ft
}

func (ft *DataTree) Init() {
	ft.Tree.Init()
	ft.Root = ft
}

// FileNode is databrowser version of FileNode for FileTree
type FileNode struct {
	filetree.Node
}

func (fn *FileNode) Init() {
	fn.Node.Init()
	fn.AddContextMenu(fn.ContextMenu)
}

// Tabber returns the [Tabber] for this filenode, from root tree.
func (fn *FileNode) Tabber() Tabber {
	fr := AsDataTree(fn.Root)
	if fr != nil {
		return fr.Tabber
	}
	return nil
}

func (fn *FileNode) WidgetTooltip(pos image.Point) (string, image.Point) {
	res := fn.Tooltip
	if fn.Info.Cat == fileinfo.Data {
		ofn := fn.AsNode()
		switch fn.Info.Known {
		case fileinfo.Number, fileinfo.String:
			dv := DataFS(ofn)
			v := dv.AsString()
			if res != "" {
				res += " "
			}
			res += v
		}
	}
	return res, fn.DefaultTooltipPos()
}

// DataFS returns the datafs representation of this item.
// returns nil if not a dataFS item.
func DataFS(fn *filetree.Node) *datafs.Data {
	dfs, ok := fn.FileRoot().FS.(*datafs.Data)
	if !ok {
		return nil
	}
	dfi, err := dfs.Stat(string(fn.Filepath))
	if errors.Log(err) != nil {
		return nil
	}
	return dfi.(*datafs.Data)
}

func (fn *FileNode) GetFileInfo() error {
	err := fn.InitFileInfo()
	if fn.FileRoot().FS == nil {
		return err
	}
	d := DataFS(fn.AsNode())
	if d != nil {
		fn.Info.Known = d.KnownFileInfo()
		fn.Info.Cat = fileinfo.Data
		switch fn.Info.Known {
		case fileinfo.Tensor:
			fn.Info.Ic = icons.BarChart
		case fileinfo.Table:
			fn.Info.Ic = icons.BarChart4Bars
		case fileinfo.Number:
			fn.Info.Ic = icons.Tag
		case fileinfo.String:
			fn.Info.Ic = icons.Title
		default:
			fn.Info.Ic = icons.BarChart
		}
	}
	return err
}

func (fn *FileNode) OpenFile() error {
	ofn := fn.AsNode()
	ts := fn.Tabber()
	if ts == nil {
		return nil
	}
	df := fsx.DirAndFile(string(fn.Filepath))
	switch {
	case fn.IsDir():
		d := DataFS(ofn)
		dt := d.GetDirTable(nil)
		ts.TensorTable(df, dt)
	case fn.Info.Cat == fileinfo.Data:
		switch fn.Info.Known {
		case fileinfo.Tensor:
			d := DataFS(ofn)
			ts.TensorEditor(df, d.Data)
		case fileinfo.Number:
			dv := DataFS(ofn)
			v := dv.AsFloat32()
			d := core.NewBody(df)
			core.NewText(d).SetType(core.TextSupporting).SetText(df)
			sp := core.NewSpinner(d).SetValue(v)
			d.AddBottomBar(func(bar *core.Frame) {
				d.AddCancel(bar)
				d.AddOK(bar).OnClick(func(e events.Event) {
					dv.SetFloat32(sp.Value)
				})
			})
			d.RunDialog(fn)
		case fileinfo.String:
			dv := DataFS(ofn)
			v := dv.AsString()
			d := core.NewBody(df)
			core.NewText(d).SetType(core.TextSupporting).SetText(df)
			tf := core.NewTextField(d).SetText(v)
			d.AddBottomBar(func(bar *core.Frame) {
				d.AddCancel(bar)
				d.AddOK(bar).OnClick(func(e events.Event) {
					dv.SetString(tf.Text())
				})
			})
			d.RunDialog(fn)

		default:
			dt := table.New()
			err := dt.OpenCSV(fsx.Filename(fn.Filepath), tensor.Tab) // todo: need more flexible data handling mode
			if err != nil {
				core.ErrorSnackbar(fn, err)
			} else {
				ts.TensorTable(df, dt)
			}
		}
	case fn.IsExec(): // todo: use exec?
		fn.OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Video: // todo: use our video viewer
		fn.OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Audio: // todo: use our audio viewer
		fn.OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Image: // todo: use our image viewer
		fn.OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Model: // todo: use xyz
		fn.OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Sheet: // todo: use our spreadsheet :)
		fn.OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Bin: // don't edit
		fn.OpenFilesDefault()
	case fn.Info.Cat == fileinfo.Archive || fn.Info.Cat == fileinfo.Backup: // don't edit
		fn.OpenFilesDefault()
	default:
		ts.EditorString(df, string(fn.Filepath))
	}
	return nil
}

// EditFiles calls EditFile on selected files
func (fn *FileNode) EditFiles() { //types:add
	fn.SelectedFunc(func(sn *filetree.Node) {
		sn.This.(*FileNode).EditFile()
	})
}

// EditFile pulls up this file in a texteditor
func (fn *FileNode) EditFile() {
	if fn.IsDir() {
		fn.OpenFile()
		return
	}
	ts := fn.Tabber()
	if ts == nil {
		return
	}
	if fn.Info.Cat == fileinfo.Data {
		fn.OpenFile()
		return
	}
	df := fsx.DirAndFile(string(fn.Filepath))
	ts.EditorString(df, string(fn.Filepath))
}

// PlotFiles calls PlotFile on selected files
func (fn *FileNode) PlotFiles() { //types:add
	fn.SelectedFunc(func(sn *filetree.Node) {
		if sfn, ok := sn.This.(*FileNode); ok {
			sfn.PlotFile()
		}
	})
}

// PlotFile pulls up this file in a texteditor.
func (fn *FileNode) PlotFile() {
	ts := fn.Tabber()
	if ts == nil {
		return
	}
	d := DataFS(fn.AsNode())
	df := fsx.DirAndFile(string(fn.Filepath))
	ptab := df + " Plot"
	var dt *table.Table
	switch {
	case fn.IsDir():
		dt = d.GetDirTable(nil)
	case fn.Info.Cat == fileinfo.Data:
		switch fn.Info.Known {
		case fileinfo.Tensor:
			tsr := d.Data
			dt = table.New(df)
			dt.Columns.Rows = tsr.DimSize(0)
			if ix, ok := tsr.(*tensor.Rows); ok {
				dt.Indexes = ix.Indexes
			}
			rc := dt.AddIntColumn("Row")
			for r := range dt.Columns.Rows {
				rc.Values[r] = r
			}
			dt.AddColumn(fn.Name, tsr.AsValues())
		// case fileinfo.Table:
		// 	dt = d.AsTable()
		default:
			dt = table.New(df)
			err := dt.OpenCSV(fsx.Filename(fn.Filepath), tensor.Tab) // todo: need more flexible data handling mode
			if err != nil {
				core.ErrorSnackbar(fn, err)
				dt = nil
			}
		}
	}
	if dt == nil {
		return
	}
	pl := ts.PlotTable(ptab, dt)
	_ = pl
	// pl.Options.Title = df
	// TODO: apply column and plot level options.
}

// DiffDirs displays a browser with differences between two selected directories
func (fn *FileNode) DiffDirs() { //types:add
	var da, db *filetree.Node
	fn.SelectedFunc(func(sn *filetree.Node) {
		if sn.IsDir() {
			if da == nil {
				da = sn
			} else if db == nil {
				db = sn
			}
		}
	})
	if da == nil || db == nil {
		core.MessageSnackbar(fn, "DiffDirs requires two selected directories")
		return
	}
	NewDiffBrowserDirs(string(da.Filepath), string(db.Filepath))
}

// NewDiffBrowserDirs returns a new diff browser for files that differ
// within the two given directories.  Excludes Job and .tsv data files.
func NewDiffBrowserDirs(pathA, pathB string) {
	brow, b := diffbrowser.NewBrowserWindow()
	brow.DiffDirs(pathA, pathB, func(fname string) bool {
		if IsTableFile(fname) {
			return true
		}
		if strings.HasPrefix(fname, "job.") || fname == "dbmeta.toml" {
			return true
		}
		return false
	})
	b.RunWindow()
}

func IsTableFile(fname string) bool {
	return strings.HasSuffix(fname, ".tsv") || strings.HasSuffix(fname, ".csv")
}

func (fn *FileNode) ContextMenu(m *core.Scene) {
	core.NewFuncButton(m).SetFunc(fn.EditFiles).SetText("Edit").SetIcon(icons.Edit).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection(), states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.PlotFiles).SetText("Plot").SetIcon(icons.Edit).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || fn.Info.Cat != fileinfo.Data, states.Disabled)
		})
	core.NewFuncButton(m).SetFunc(fn.DiffDirs).SetText("Diff Dirs").SetIcon(icons.Edit).
		Styler(func(s *styles.Style) {
			s.SetState(!fn.HasSelection() || !fn.IsDir(), states.Disabled)
		})
}

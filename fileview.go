// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

type FileSize datasize.ByteSize

func (fs FileSize) String() string {
	return (datasize.ByteSize)(fs).HumanReadable()
}

// Note: can get all the detailed birth, access, change times from this package
// 	"github.com/djherbis/times"

type FileTime time.Time

func (ft FileTime) String() string {
	return (time.Time)(ft).Format("Mon Jan  2 15:04:05 MST 2006")
}

// FileInfo represents the information about a given file / directory
type FileInfo struct {
	Name    string      `desc:"name of the file"`
	Size    FileSize    `desc:"size of the file in bytes"`
	Kind    string      `desc:"type of file / directory -- including MIME type"`
	Mode    os.FileMode `desc:"file mode bits"`
	ModTime FileTime    `desc:"time that contents (only) were last modified"`
}

// todo:
// * structtableview: sort by diff cols, keep col header pinned at top!
// * busy cursor when loading
// * NewFolder -- does everyone call it folder now?  Folder is the gui version of the name..
// * reset scroll position in structtableview when it rebuilds
// * tree view of directory on left of files view
// * prior paths, saved to prefs dir
// * favorites, with DND, saved to prefs dir
// * icons!  key in this kind of view -- shouldn't be too hard in terms of valueview type -- just define it!
// * filter(s) for types of files to highlight as selectable?  useful or not?

// FileView is a viewer onto files -- core of the file chooser dialog
type FileView struct {
	Frame
	DirPath string      `desc:"path to directory of files to display"`
	SelFile string      `desc:"selected file"`
	Files   []*FileInfo `desc:"files for current directory"`
	FileSig ki.Signal   `desc:"signal for file actions"`
}

var KiT_FileView = kit.Types.AddType(&FileView{}, FileViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetPathFile sets the path, initial select file (or "") and intializes the view
func (fv *FileView) SetPathFile(path, file string) {
	fv.DirPath = path
	fv.SelFile = file
	fv.UpdateFromPath()
}

// SelectedFile returns the full path to selected file
func (fv *FileView) SelectedFile() string {
	return filepath.Join(fv.DirPath, fv.SelFile)
}

func (fv *FileView) UpdateFromPath() {
	mods, updt := fv.StdConfig()
	fv.UpdateFiles()
	if mods {
		fv.UpdateEnd(updt)
	}
}

var FileViewProps = ki.Props{
	"background-color": &Prefs.BackgroundColor,
}

// FileViewKindColorMap translates file Kinds into different colors for the file viewer
var FileViewKindColorMap = map[string]string{
	"Folder":           "blue",
	"application/json": "purple",
}

func FileViewStyleFunc(slice interface{}, widg Node2D, row, col int, vv ValueView) {
	finf, ok := slice.(*[]*FileInfo)
	if ok {
		gi := widg.AsNode2D()
		if clr, got := FileViewKindColorMap[(*finf)[row].Kind]; got {
			gi.SetProp("color", clr)
		} else {
			gi.DeleteProp("color")
		}
	}
}

// SetFrame configures view as a frame
func (fv *FileView) SetFrame() {
	fv.Lay = LayoutCol
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (fv *FileView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(KiT_Layout, "path-row")
	config.Add(KiT_Space, "path-space")
	config.Add(KiT_Layout, "files-row")
	config.Add(KiT_Space, "files-space")
	config.Add(KiT_Layout, "sel-row")
	config.Add(KiT_Space, "sel-space")
	config.Add(KiT_Layout, "buttons")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (fv *FileView) StdConfig() (mods, updt bool) {
	fv.SetFrame()
	config := fv.StdFrameConfig()
	mods, updt = fv.ConfigChildren(config, false)
	if mods {
		fv.ConfigPathRow()
		fv.ConfigFilesRow()
		fv.ConfigSelRow()
		fv.ConfigButtons()
	}
	return
}

func (fv *FileView) ConfigPathRow() {
	pr := fv.ChildByName("path-row", 0).(*Layout)
	pr.Lay = LayoutRow
	pr.SetStretchMaxWidth()
	config := kit.TypeAndNameList{}
	config.Add(KiT_Label, "path-lbl")
	config.Add(KiT_TextField, "path")
	config.Add(KiT_Action, "path-up")
	pr.ConfigChildren(config, false) // already covered by parent update
	pl := pr.ChildByName("path-lbl", 0).(*Label)
	pl.Text = "Path:"
	pf := fv.PathField()
	pf.SetMinPrefWidth(units.NewValue(30.0, units.Em))
	pf.SetStretchMaxWidth()
	pf.TextFieldSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(TextFieldDone) {
			fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
			pff, _ := send.(*TextField)
			fvv.DirPath = pff.Text
			fvv.SetFullReRender()
			fvv.UpdateFiles()
		}
	})

	pu := pr.ChildByName("path-up", 0).(*Action)
	pu.Icon = IconName("widget-wedge-up")
	pu.SetProp("vertical-align", AlignMiddle)
	pu.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
		fvv.DirPathUp()
	})
}

func (fv *FileView) ConfigFilesRow() {
	fr := fv.ChildByName("files-row", 2).(*Layout)
	fr.SetStretchMaxHeight()
	fr.SetStretchMaxWidth()
	fr.Lay = LayoutRow
	config := kit.TypeAndNameList{}
	// todo: add favorites, dir tree
	config.Add(KiT_StructTableView, "files-view")
	fr.ConfigChildren(config, false) // already covered by parent update
	sv := fv.FilesView()
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetInactive() // select only
	sv.StyleFunc = FileViewStyleFunc
	sv.SetSlice(&fv.Files, nil)
	sv.SelectSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
		sv, _ := send.(*StructTableView)
		fvv.FileSelect(sv.SelectedIdx)
	})
}

func (fv *FileView) ConfigSelRow() {
	sr := fv.ChildByName("sel-row", 4).(*Layout)
	sr.Lay = LayoutRow
	sr.SetStretchMaxWidth()
	config := kit.TypeAndNameList{}
	config.Add(KiT_Label, "sel-lbl")
	config.Add(KiT_TextField, "sel")
	sr.ConfigChildren(config, false) // already covered by parent update
	sl := sr.ChildByName("sel-lbl", 0).(*Label)
	sl.Text = "File:"
	sf := fv.SelField()
	sf.SetMinPrefWidth(units.NewValue(30.0, units.Em))
	sf.SetStretchMaxWidth()
	sf.GrabFocus()
	sf.TextFieldSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(TextFieldDone) {
			fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
			pff, _ := send.(*TextField)
			fvv.SelFile = pff.Text
		}
	})
}

// PathField returns the TextField of the path
func (fv *FileView) PathField() *TextField {
	pr := fv.ChildByName("path-row", 0).(*Layout)
	return pr.ChildByName("path", 1).(*TextField)
}

// FilesView returns the StructTableView of the files
func (fv *FileView) FilesView() *StructTableView {
	fr := fv.ChildByName("files-row", 2).(*Layout)
	return fr.ChildByName("files-view", 0).(*StructTableView)
}

// SelField returns the TextField of the selected file
func (fv *FileView) SelField() *TextField {
	sr := fv.ChildByName("sel-row", 4).(*Layout)
	return sr.ChildByName("sel", 1).(*TextField)
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (fv *FileView) ButtonBox() (*Layout, int) {
	idx := fv.ChildIndexByName("buttons", 0)
	if idx < 0 {
		return nil, -1
	}
	return fv.Child(idx).(*Layout), idx
}

// UpdatePath ensures that path is in abs form and ready to be used..
func (fv *FileView) UpdatePath() {
	if fv.DirPath == "" {
		fv.DirPath, _ = os.Getwd()
	}
	fv.DirPath, _ = filepath.Abs(fv.DirPath)
}

// UpdateFiles updates list of files and other views for current path
func (fv *FileView) UpdateFiles() {
	updt := fv.UpdateStart()
	fv.SetFullReRender()
	fv.UpdatePath()
	pf := fv.PathField()
	pf.SetText(fv.DirPath)
	sf := fv.SelField()
	sf.SetText(fv.SelFile)

	fv.Files = make([]*FileInfo, 0, 1000)
	filepath.Walk(fv.DirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("Path %q: Error: %v", fv.DirPath, err)
			PromptDialog(fv.Viewport, "FileView UpdateFiles", emsg, true, false, nil, nil)
			return err
		}
		if path == fv.DirPath { // proceed..
			return nil
		}
		_, fn := filepath.Split(path)
		fi := FileInfo{
			Name:    fn,
			Size:    FileSize(info.Size()),
			Mode:    info.Mode(),
			ModTime: FileTime(info.ModTime()),
		}
		if info.IsDir() {
			fi.Kind = "Folder"
		} else {
			ext := filepath.Ext(fn)
			fi.Kind = mime.TypeByExtension(ext)
		}
		fv.Files = append(fv.Files, &fi)
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})

	sv := fv.FilesView()
	sv.UpdateFromSlice()
	fv.UpdateEnd(updt)
}

// DirPathUp moves up one directory in the path
func (fv *FileView) DirPathUp() {
	pdr, _ := filepath.Split(fv.DirPath)
	if pdr == "" {
		return
	}
	fv.DirPath = pdr
	fv.SetFullReRender()
	fv.UpdateFiles()
}

// FileSelect updates selection with given selected file
func (fv *FileView) FileSelect(idx int) {
	fi := fv.Files[idx]
	if fi.Kind == "Folder" {
		fv.DirPath = filepath.Join(fv.DirPath, fi.Name)
		fv.SetFullReRender()
		fv.UpdateFiles()
		return
	}
	fv.SelFile = fi.Name
	sf := fv.SelField()
	sf.SetText(fv.SelFile)
}

// ConfigButtons configures the buttons
func (fv *FileView) ConfigButtons() {
	// bb, _ := fv.ButtonBox()
	// config := kit.TypeAndNameList{}
	// config.Add(KiT_Button, "Add")
	// mods, updt := bb.ConfigChildren(config, false)
	// addb := bb.ChildByName("Add", 0).EmbeddedStruct(KiT_Button).(*Button)
	// addb.SetText("Add")
	// addb.ButtonSig.ConnectOnly(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
	// 	if sig == int64(ButtonClicked) {
	// 		fvv := recv.EmbeddedStruct(KiT_FileView).(*FileView)
	// 		fvv.SliceNewAt(-1)
	// 	}
	// })
	// if mods {
	// 	bb.UpdateEnd(updt)
	// }
}

func (fv *FileView) Render2D() {
	fv.ClearFullReRender()
	fv.Frame.Render2D()
}

func (fv *FileView) ReRender2D() (node Node2D, layout bool) {
	if fv.NeedsFullReRender() {
		node = nil
		layout = false
	} else {
		node = fv.This.(Node2D)
		layout = true
	}
	return
}

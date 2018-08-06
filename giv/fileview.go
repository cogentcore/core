// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/mitchellh/go-homedir"
)

////////////////////////////////////////////////////////////////////////////////
//  FileView

// todo:

// * search: use highlighting, not filtering -- < > arrows etc -- ext *.svg is
// just search pre-load -- add arg
// * popup menu to operate on files: trash, delete, rename, move
// * also simple search-while typing in grid?
// * busy cursor when loading
// * prior paths, saved to prefs dir -- easy -- add combobox with no label that displays them -- ideally would be an editable combobox..
// * in inactive select mode, it would be better to NOT immediately traverse into subdirs. and even in regular gui mode -- that should be more of a double-click.. although that would cause dialog to close now.. grr.

// FileView is a viewer onto files -- core of the file chooser dialog
type FileView struct {
	gi.Frame
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
	"color":            &gi.Prefs.FontColor,
	"background-color": &gi.Prefs.BackgroundColor,
}

// FileViewKindColorMap translates file Kinds into different colors for the file viewer
var FileViewKindColorMap = map[string]string{
	"Folder":           "blue",
	"application/json": "purple",
}

func FileViewStyleFunc(slice interface{}, widg gi.Node2D, row, col int, vv ValueView) {
	finf, ok := slice.([]*FileInfo)
	if ok {
		gi := widg.AsNode2D()
		if clr, got := FileViewKindColorMap[finf[row].Kind]; got {
			gi.SetProp("color", clr)
		} else {
			gi.DeleteProp("color")
		}
	}
}

// SetFrame configures view as a frame
func (fv *FileView) SetFrame() {
	fv.Lay = gi.LayoutVert
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (fv *FileView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Layout, "path-row")
	config.Add(gi.KiT_Space, "path-space")
	config.Add(gi.KiT_Layout, "files-row")
	config.Add(gi.KiT_Space, "files-space")
	config.Add(gi.KiT_Layout, "sel-row")
	config.Add(gi.KiT_Space, "sel-space")
	config.Add(gi.KiT_Layout, "buttons")
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
	pr := fv.KnownChildByName("path-row", 0).(*gi.Layout)
	pr.Lay = gi.LayoutHoriz
	pr.SetStretchMaxWidth()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "path-lbl")
	config.Add(gi.KiT_TextField, "path")
	config.Add(gi.KiT_Action, "path-up")
	config.Add(gi.KiT_Action, "path-fav")
	config.Add(gi.KiT_Action, "new-folder")
	pr.ConfigChildren(config, false) // already covered by parent update
	pl := pr.KnownChildByName("path-lbl", 0).(*gi.Label)
	pl.Text = "Path:"
	pf := fv.PathField()
	pf.SetMinPrefWidth(units.NewValue(60.0, units.Ex))
	pf.SetStretchMaxWidth()
	pf.TextFieldSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
			pff, _ := send.(*gi.TextField)
			fvv.DirPath = pff.Text()
			fvv.SetFullReRender()
			fvv.UpdateFiles()
		}
	})

	pu := pr.KnownChildByName("path-up", 0).(*gi.Action)
	pu.Icon = gi.IconName("widget-wedge-up")
	pu.Tooltip = "go up one level into the parent folder"
	pu.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
		fvv.DirPathUp()
	})

	pfv := pr.KnownChildByName("path-fav", 0).(*gi.Action)
	pfv.Icon = gi.IconName("heart")
	pfv.Tooltip = "save this path to the favorites list -- saves current Prefs"
	pfv.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
		fvv.AddPathToFavs()
	})

	nf := pr.KnownChildByName("new-folder", 0).(*gi.Action)
	nf.Icon = gi.IconName("folder-plus")
	nf.Tooltip = "Create a new folder in this folder"
	nf.ActionSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
		fvv.NewFolder()
	})
}

func (fv *FileView) ConfigFilesRow() {
	fr := fv.KnownChildByName("files-row", 2).(*gi.Layout)
	fr.SetStretchMaxHeight()
	fr.SetStretchMaxWidth()
	fr.Lay = gi.LayoutHoriz
	config := kit.TypeAndNameList{}
	config.Add(KiT_TableView, "favs-view")
	config.Add(KiT_TableView, "files-view")
	fr.ConfigChildren(config, false) // already covered by parent update

	sv := fv.FavsView()
	sv.CSS = ki.Props{
		"textfield": ki.Props{
			":inactive": ki.Props{
				"background-color": &gi.Prefs.ControlColor,
			},
		},
	}
	sv.SetStretchMaxHeight()
	sv.SetProp("index", false)
	sv.SetProp("inact-key-nav", false) // can only have one active -- files..
	sv.SetInactive()                   // select only
	sv.SelectedIdx = -1
	sv.SetSlice(&gi.Prefs.FavPaths, nil)
	sv.WidgetSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.WidgetSelected) {
			fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
			svv, _ := send.(*TableView)
			fvv.FavSelect(svv.SelectedIdx)
		}
	})

	sv = fv.FilesView()
	sv.CSS = ki.Props{
		"textfield": ki.Props{
			":inactive": ki.Props{
				"background-color": &gi.Prefs.ControlColor,
			},
		},
	}
	sv.SetProp("index", false) // no index
	sv.SetStretchMaxHeight()
	sv.SetStretchMaxWidth()
	sv.SetInactive() // select only
	sv.StyleFunc = FileViewStyleFunc
	sv.SetSlice(&fv.Files, nil)
	if gi.Prefs.FileViewSort != "" {
		sv.SetSortFieldName(gi.Prefs.FileViewSort)
	}
	sv.WidgetSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.WidgetSelected) {
			fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
			svv, _ := send.(*TableView)
			fvv.FileSelect(svv.SelectedIdx)
		}
	})
}

func (fv *FileView) ConfigSelRow() {
	sr := fv.KnownChildByName("sel-row", 4).(*gi.Layout)
	sr.Lay = gi.LayoutHoriz
	sr.SetStretchMaxWidth()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "sel-lbl")
	config.Add(gi.KiT_TextField, "sel")
	sr.ConfigChildren(config, false) // already covered by parent update
	sl := sr.KnownChildByName("sel-lbl", 0).(*gi.Label)
	sl.Text = "File:"
	sf := fv.SelField()
	sf.SetMinPrefWidth(units.NewValue(60.0, units.Ex))
	sf.SetStretchMaxWidth()
	sf.GrabFocus()
	sf.TextFieldSig.Connect(fv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			fvv, _ := recv.EmbeddedStruct(KiT_FileView).(*FileView)
			pff, _ := send.(*gi.TextField)
			fvv.SelFile = pff.Text()
		}
	})
}

// PathField returns the TextField of the path
func (fv *FileView) PathField() *gi.TextField {
	pr := fv.KnownChildByName("path-row", 0).(*gi.Layout)
	return pr.KnownChildByName("path", 1).(*gi.TextField)
}

// FavsView returns the TableView of the favorites
func (fv *FileView) FavsView() *TableView {
	fr := fv.KnownChildByName("files-row", 2).(*gi.Layout)
	return fr.KnownChildByName("favs-view", 1).(*TableView)
}

// FilesView returns the TableView of the files
func (fv *FileView) FilesView() *TableView {
	fr := fv.KnownChildByName("files-row", 2).(*gi.Layout)
	return fr.KnownChildByName("files-view", 1).(*TableView)
}

// SelField returns the TextField of the selected file
func (fv *FileView) SelField() *gi.TextField {
	sr := fv.KnownChildByName("sel-row", 4).(*gi.Layout)
	return sr.KnownChildByName("sel", 1).(*gi.TextField)
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (fv *FileView) ButtonBox() (*gi.Layout, int) {
	idx, ok := fv.Children().IndexByName("buttons", 0)
	if !ok {
		return nil, -1
	}
	return fv.KnownChild(idx).(*gi.Layout), idx
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
	oswin.TheApp.Cursor().Push(cursor.Wait)
	defer oswin.TheApp.Cursor().Pop()

	fv.Files = make([]*FileInfo, 0, 1000)
	filepath.Walk(fv.DirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("Path %q: Error: %v", fv.DirPath, err)
			// if fv.Viewport != nil {
			// 	gi.PromptDialog(fv.Viewport, "FileView UpdateFiles", emsg, true, false, nil, nil)
			// } else {
			log.Printf("gi.FileView error: %v\n", emsg)
			// }
			return nil // ignore
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
		fi.Ic = FileKindToIcon(fi.Kind, fi.Name)
		fv.Files = append(fv.Files, &fi)
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})

	sv := fv.FilesView()
	sv.CurSelField = "Name"
	sv.CurSelVal = fv.SelFile
	sv.UpdateFromSlice()
	fv.UpdateEnd(updt)
}

// UpdateFavs updates list of files and other views for current path
func (fv *FileView) UpdateFavs() {
	sv := fv.FavsView()
	sv.UpdateFromSlice()
}

// AddPathToFavs adds the current path to favorites
func (fv *FileView) AddPathToFavs() {
	dp := fv.DirPath
	if dp == "" {
		return
	}
	_, fnm := filepath.Split(dp)
	hd, _ := homedir.Dir()
	hd += string(filepath.Separator)
	if strings.HasPrefix(dp, hd) {
		dp = filepath.Join("~", strings.TrimPrefix(dp, hd))
	}
	if fnm == "" {
		fnm = dp
	}
	fi := gi.FavPathItem{"folder", fnm, dp}
	gi.Prefs.FavPaths = append(gi.Prefs.FavPaths, fi)
	gi.Prefs.Save()
	fv.UpdateFavs()
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

// NewFolder creates a new folder in current directory
func (fv *FileView) NewFolder() {
	dp := fv.DirPath
	if dp == "" {
		return
	}
	np := filepath.Join(dp, "NewFolder")
	err := os.MkdirAll(np, 0775)
	if err != nil {
		emsg := fmt.Sprintf("NewFolder at: %q: Error: %v", fv.DirPath, err)
		if fv.Viewport != nil {
			gi.PromptDialog(fv.Viewport, "FileView Error", emsg, true, false, nil, nil, nil)
		} else {
			log.Printf("gi.FileView NewFolder error: %v\n", emsg)
		}
	} else {
		fmt.Printf("gi.FileView made new folder: %v\n", np)
	}
	fv.SetFullReRender()
	fv.UpdateFiles()
}

// FileSelect updates selection with given selected file
func (fv *FileView) FileSelect(idx int) {
	if idx < 0 {
		return
	}
	fv.SaveSortPrefs()
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

// FavSelect selects a favorite path and goes there
func (fv *FileView) FavSelect(idx int) {
	if idx < 0 || idx >= len(gi.Prefs.FavPaths) {
		return
	}
	fi := gi.Prefs.FavPaths[idx]
	fv.DirPath, _ = homedir.Expand(fi.Path)
	fv.SetFullReRender()
	fv.UpdateFiles()
}

// SaveSortPrefs saves current sorting preferences
func (fv *FileView) SaveSortPrefs() {
	sv := fv.FilesView()
	if sv == nil {
		return
	}
	gi.Prefs.FileViewSort = sv.SortFieldName()
	gi.Prefs.Save()
}

// ConfigButtons configures the buttons
func (fv *FileView) ConfigButtons() {
	// bb, _ := fv.ButtonBox()
	// config := kit.TypeAndNameList{}
	// config.Add(gi.KiT_Button, "Add")
	// mods, updt := bb.ConfigChildren(config, false)
	// addb := bb.KnownChildByName("Add", 0).EmbeddedStruct(gi.KiT_Button).(*Button)
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

////////////////////////////////////////////////////////////////////////////////
//  FileInfo

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

// note: rendering icons taking a fair amount of extra time

// FileInfo represents the information about a given file / directory
type FileInfo struct {
	Ic      gi.IconName `desc:"icon for file"`
	Name    string      `width:"40" desc:"name of the file"`
	Size    FileSize    `desc:"size of the file in bytes"`
	Kind    string      `width:"20" max-width:"20" desc:"type of file / directory -- including MIME type"`
	Mode    os.FileMode `desc:"file mode bits"`
	ModTime FileTime    `desc:"time that contents (only) were last modified"`
}

// MimeToIconMap has special cases for mapping mime type to icon, for those that basic string doesn't work
var MimeToIconMap = map[string]string{
	"svg+xml": "svg",
}

// FileKindToIcon maps kinds to icon names, using extension directly from file as a last resort
func FileKindToIcon(kind, name string) gi.IconName {
	kind = strings.ToLower(kind)
	icn := gi.IconName(kind)
	if icn.IsValid() {
		return icn
	}
	if strings.Contains(kind, "/") {
		si := strings.IndexByte(kind, '/')
		typ := kind[:si]
		subtyp := kind[si+1:]
		if icn = "file-" + gi.IconName(subtyp); icn.IsValid() {
			return icn
		}
		if icn = gi.IconName(subtyp); icn.IsValid() {
			return icn
		}
		if ms, ok := MimeToIconMap[string(subtyp)]; ok {
			if icn = gi.IconName(ms); icn.IsValid() {
				return icn
			}
		}
		if icn = "file-" + gi.IconName(typ); icn.IsValid() {
			return icn
		}
		if icn = gi.IconName(typ); icn.IsValid() {
			return icn
		}
		if ms, ok := MimeToIconMap[string(typ)]; ok {
			if icn = gi.IconName(ms); icn.IsValid() {
				return icn
			}
		}
	}
	ext := filepath.Ext(name)
	if ext != "" {
		if icn = gi.IconName(ext[1:]); icn.IsValid() {
			return icn
		}
	}

	icn = gi.IconName("none")
	return icn
}

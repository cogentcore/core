// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"unicode"

	"github.com/goki/gi"
	"github.com/goki/gi/complete"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/mitchellh/go-homedir"
)

//////////////////////////////////////////////////////////////////////////
//  FileView

// todo:

// * search: use highlighting, not filtering -- < > arrows etc
// * also simple search-while typing in grid?
// * fileview selector DND is a file:/// url

// FileView is a viewer onto files -- core of the file chooser dialog
type FileView struct {
	gi.Frame
	DirPath     string             `desc:"path to directory of files to display"`
	SelFile     string             `desc:"selected file"`
	Ext         string             `desc:"target extension(s) (comma separated if multiple, including initial .), if any"`
	FilterFunc  FileViewFilterFunc `view:"-" json:"-" xml:"-" desc:"optional styling function"`
	ExtMap      map[string]string  `desc:"map of lower-cased extensions from Ext -- used for highlighting files with one of these extensions -- maps onto original ext value"`
	Files       []*FileInfo        `desc:"files for current directory"`
	SelectedIdx int                `desc:"index of currently-selected file in Files list (-1 if none)"`
	FileSig     ki.Signal          `desc:"signal for file actions"`
}

var KiT_FileView = kit.Types.AddType(&FileView{}, FileViewProps)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// FileViewFilterFunc is a filtering function for files -- returns true if the
// file should be visible in the view, and false if not
type FileViewFilterFunc func(fv *FileView, fi *FileInfo) bool

// FileViewDirOnlyFilter is a FileViewFilterFunc that only shows directories (folders).
func FileViewDirOnlyFilter(fv *FileView, fi *FileInfo) bool {
	return fi.IsDir()
}

// FileViewExtOnlyFilter is a FileViewFilterFunc that only shows files that
// match the target extensions, and directories.
func FileViewExtOnlyFilter(fv *FileView, fi *FileInfo) bool {
	if fi.IsDir() {
		return true
	}
	ext := strings.ToLower(filepath.Ext(fi.Name))
	_, has := fv.ExtMap[ext]
	return has
}

// SetFilename sets the initial filename (splitting out path and filename) and
// initializes the view
func (fv *FileView) SetFilename(filename, ext string) {
	fv.DirPath, fv.SelFile = filepath.Split(filename)
	fv.SetExt(ext)
	// fv.UpdateFromPath()
	fv.DoStdConfig()
}

// SetPathFile sets the path, initial select file (or "") and intializes the view
func (fv *FileView) SetPathFile(path, file, ext string) {
	fv.DirPath = path
	fv.SelFile = file
	fv.SetExt(ext)
	fv.DoStdConfig()
	// fv.UpdateFromPath()
}

// SelectedFile returns the full path to selected file
func (fv *FileView) SelectedFile() string {
	return filepath.Join(fv.DirPath, fv.SelFile)
}

// SelectedFileInfo returns the currently-selected fileinfo, returns
// false if none
func (fv *FileView) SelectedFileInfo() (*FileInfo, bool) {
	if fv.SelectedIdx < 0 || fv.SelectedIdx >= len(fv.Files) {
		return nil, false
	}
	return fv.Files[fv.SelectedIdx], true
}

// SelectFile selects the current file -- if a directory it opens
// the directory; if a file it selects the file and closes dialog
func (fv *FileView) SelectFile() {
	if fi, ok := fv.SelectedFileInfo(); ok {
		if fi.IsDir() {
			fv.DirPath = filepath.Join(fv.DirPath, fi.Name)
			fv.SelFile = ""
			fv.SelectedIdx = -1
			fv.UpdateFilesAction()
			return
		}
		fv.FileSig.Emit(fv.This(), int64(FileViewDoubleClicked), fv.SelectedFile())
	}
}

// UpdateFromPath will update view based on current DirPath
func (fv *FileView) UpdateFromPath() {
	mods, updt := fv.StdConfig()
	fv.UpdateFiles()
	if mods {
		fv.UpdateEnd(updt)
	}
}

var FileViewProps = ki.Props{
	"color":            &gi.Prefs.Colors.Font,
	"background-color": &gi.Prefs.Colors.Background,
	"max-width":        -1,
	"max-height":       -1,
}

// FileViewKindColorMap translates file Kinds into different colors for the file viewer
var FileViewKindColorMap = map[string]string{
	"folder": "pref(link)",
}

// FileViewSignals are signals that fileview sends based on user actions.
type FileViewSignals int64

const (
	// FileViewDoubleClicked emitted for double-click on a non-directory file
	// in table view (data is full selected file name w/ path) -- typically
	// closes dialog.
	FileViewDoubleClicked FileViewSignals = iota

	// FileViewWillUpdate emitted when list of files is about to be updated
	// based on user action (data is current path) -- current DirPath will be
	// used -- can intervene here if needed.
	FileViewWillUpdate

	// FileViewUpdated emitted after list of files has been updated (data is
	// current path).
	FileViewUpdated

	// FileViewNewFolder emitted when a new folder was created (data is
	// current path).
	FileViewNewFolder

	// FileViewFavAdded emitted when a new favorite was added (data is new
	// favorite path).
	FileViewFavAdded

	FileViewSignalsN
)

//go:generate stringer -type=FileViewSignals

func FileViewStyleFunc(tv *TableView, slice interface{}, widg gi.Node2D, row, col int, vv ValueView) {
	finf, ok := slice.([]*FileInfo)
	if ok {
		wi := widg.AsNode2D()
		if clr, got := FileViewKindColorMap[finf[row].Kind]; got {
			wi.SetProp("color", clr)
			return
		}
		if fvv, pok := tv.ParentByType(KiT_FileView, true); pok {
			fv := fvv.Embed(KiT_FileView).(*FileView)
			fn := finf[row].Name
			ext := strings.ToLower(filepath.Ext(fn))
			if _, has := fv.ExtMap[ext]; has {
				wi.SetProp("color", "pref(link)")
				return
			}
		}
		wi.DeleteProp("color")
	}
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (fv *FileView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "path-tbar")
	config.Add(gi.KiT_Layout, "files-row")
	config.Add(gi.KiT_Layout, "sel-row")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (fv *FileView) StdConfig() (mods, updt bool) {
	fv.Lay = gi.LayoutVert
	fv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := fv.StdFrameConfig()
	mods, updt = fv.ConfigChildren(config, true)
	if mods {
		fv.ConfigPathRow()
		fv.ConfigFilesRow()
		fv.ConfigSelRow()
	}
	return
}

// DoStdConfig does the standard configuration, but does not update files
func (fv *FileView) DoStdConfig() {
	mods, updt := fv.StdConfig()
	if mods {
		fv.UpdateEnd(updt)
	}
}

func (fv *FileView) ConfigPathRow() {
	pr := fv.KnownChildByName("path-tbar", 0).(*gi.ToolBar)
	pr.Lay = gi.LayoutHoriz
	pr.SetStretchMaxWidth()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "path-lbl")
	config.Add(gi.KiT_ComboBox, "path")
	config.Add(gi.KiT_Action, "path-up")
	config.Add(gi.KiT_Action, "path-ref")
	config.Add(gi.KiT_Action, "path-fav")
	config.Add(gi.KiT_Action, "new-folder")
	mods, updt := pr.ConfigChildren(config, false) // already covered by parent update
	if mods {
		pl := pr.KnownChildByName("path-lbl", 0).(*gi.Label)
		pl.Text = "Path:"
		pl.Tooltip = "Path to look for files in: can select from list of recent paths, or edit a value directly"
		pf := fv.PathField()
		pf.Editable = true
		pf.SetMinPrefWidth(units.NewValue(60.0, units.Ch))
		pf.SetStretchMaxWidth()
		pf.ConfigParts()
		pft, found := pf.TextField()
		if found {
			pft.SetCompleter(fv, fv.PathComplete, fv.PathCompleteEdit)
			pft.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.TextFieldDone) {
					fvv, _ := recv.Embed(KiT_FileView).(*FileView)
					pff, _ := send.(*gi.TextField)
					fvv.DirPath = pff.Text()
					fvv.UpdateFilesAction()
				}
			})
		}
		pf.ComboSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			pff := send.Embed(gi.KiT_ComboBox).(*gi.ComboBox)
			sp := data.(string)
			if sp == fileViewResetPaths {
				gi.SavedPaths = make(gi.FilePaths, 1, gi.Prefs.SavedPathsMax)
				gi.SavedPaths[0] = fvv.DirPath
				pff.ItemsFromStringList(([]string)(gi.SavedPaths), true, 0)
			} else {
				fvv.DirPath = sp
				fvv.UpdateFilesAction()
			}
		})

		pu := pr.KnownChildByName("path-up", 0).(*gi.Action)
		pu.Icon = gi.IconName("widget-wedge-up")
		pu.Tooltip = "go up one level into the parent folder"
		pu.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			fvv.DirPathUp()
		})

		prf := pr.KnownChildByName("path-ref", 0).(*gi.Action)
		prf.Icon = gi.IconName("update")
		prf.Tooltip = "Update directory view -- in case files might have changed"
		prf.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			fvv.UpdateFilesAction()
		})

		pfv := pr.KnownChildByName("path-fav", 0).(*gi.Action)
		pfv.Icon = gi.IconName("heart")
		pfv.Tooltip = "save this path to the favorites list -- saves current Prefs"
		pfv.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			fvv.AddPathToFavs()
		})

		nf := pr.KnownChildByName("new-folder", 0).(*gi.Action)
		nf.Icon = gi.IconName("folder-plus")
		nf.Tooltip = "Create a new folder in this folder"
		nf.ActionSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			fvv.NewFolder()
		})
		pr.UpdateEnd(updt)
	}
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
				"background-color": &gi.Prefs.Colors.Control,
			},
		},
	}
	sv.SetStretchMaxHeight()
	sv.SetProp("max-width", 0) // no stretch
	sv.SetProp("index", false)
	sv.SetProp("inact-key-nav", false) // can only have one active -- files..
	sv.SetInactive()                   // select only
	sv.SelectedIdx = -1
	sv.SetSlice(&gi.Prefs.FavPaths, nil)
	sv.WidgetSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.WidgetSelected) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			svv, _ := send.(*TableView)
			fvv.FavSelect(svv.SelectedIdx)
		}
	})

	sv = fv.FilesView()
	sv.CSS = ki.Props{
		"textfield": ki.Props{
			":inactive": ki.Props{
				"background-color": &gi.Prefs.Colors.Control,
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
	sv.WidgetSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.WidgetSelected) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			svv, _ := send.(*TableView)
			fvv.FileSelectAction(svv.SelectedIdx)
		}
	})
	sv.TableViewSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(TableViewDoubleClicked) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			fvv.SelectFile()
		}
	})
}

func (fv *FileView) ConfigSelRow() {
	sr := fv.KnownChildByName("sel-row", 4).(*gi.Layout)
	sr.Lay = gi.LayoutHoriz
	sr.SetProp("spacing", units.NewValue(4, units.Px))
	sr.SetStretchMaxWidth()
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "sel-lbl")
	config.Add(gi.KiT_TextField, "sel")
	config.Add(gi.KiT_Label, "ext-lbl")
	config.Add(gi.KiT_TextField, "ext")
	sr.ConfigChildren(config, false) // already covered by parent update

	sl := sr.KnownChildByName("sel-lbl", 0).(*gi.Label)
	sl.Text = "File:"
	sl.Tooltip = "enter file name here (or select from above list)"
	sf := fv.SelField()
	sf.Tooltip = fmt.Sprintf("enter file name.  special keys: up/down to move selection; %v to go up to parent folder; %v or %v to select current file (if directory, goes into it, if file, selects and closes); %v / %v for prev / next history item", gi.ShortcutForFun(gi.KeyFunWordLeft), gi.ShortcutForFun(gi.KeyFunInsert), gi.ShortcutForFun(gi.KeyFunMenuOpen), gi.ShortcutForFun(gi.KeyFunHistPrev), gi.ShortcutForFun(gi.KeyFunHistNext))
	sf.SetCompleter(fv, fv.FileComplete, fv.FileCompleteEdit)
	sf.SetMinPrefWidth(units.NewValue(60.0, units.Ch))
	sf.SetStretchMaxWidth()
	sf.SetText(fv.SelFile)
	sf.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			pff, _ := send.(*gi.TextField)
			fvv.SetSelFileAction(pff.Text())
		}
	})

	el := sr.KnownChildByName("ext-lbl", 0).(*gi.Label)
	el.Text = "Ext(s):"
	el.Tooltip = "target extension(s) to highlight -- if multiple, separate with commas, and do include the . at the start"
	ef := fv.ExtField()
	ef.SetText(fv.Ext)
	ef.SetMinPrefWidth(units.NewValue(10.0, units.Ch))
	ef.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			fvv, _ := recv.Embed(KiT_FileView).(*FileView)
			pff, _ := send.(*gi.TextField)
			fvv.SetExtAction(pff.Text())
		}
	})
}

// PathField returns the ComboBox of the path
func (fv *FileView) PathField() *gi.ComboBox {
	pr := fv.KnownChildByName("path-tbar", 0).(*gi.ToolBar)
	return pr.KnownChildByName("path", 1).(*gi.ComboBox)
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

// ExtField returns the TextField of the extension
func (fv *FileView) ExtField() *gi.TextField {
	sr := fv.KnownChildByName("sel-row", 4).(*gi.Layout)
	return sr.KnownChildByName("ext", 2).(*gi.TextField)
}

// UpdatePath ensures that path is in abs form and ready to be used..
func (fv *FileView) UpdatePath() {
	if fv.DirPath == "" {
		fv.DirPath, _ = os.Getwd()
	}
	fv.DirPath, _ = homedir.Expand(fv.DirPath)
	fv.DirPath, _ = filepath.Abs(fv.DirPath)
}

// UpdateFilesAction updates list of files and other views for current path,
// emitting FileSig signals around it -- this is for gui-generated actions only.
func (fv *FileView) UpdateFilesAction() {
	fv.FileSig.Emit(fv.This(), int64(FileViewWillUpdate), fv.DirPath)
	fv.SetFullReRender()
	fv.UpdateFiles()
	sf := fv.SelField()
	sf.GrabFocus()
	fv.FileSig.Emit(fv.This(), int64(FileViewUpdated), fv.DirPath)
}

var fileViewResetPaths = "<i>Reset Paths</i>"

// UpdateFiles updates list of files and other views for current path
func (fv *FileView) UpdateFiles() {
	updt := fv.UpdateStart()
	defer fv.UpdateEnd(updt)
	var win oswin.Window
	if fv.Viewport != nil && fv.Viewport.Win != nil && fv.Viewport.Win.OSWin != nil {
		win = fv.Viewport.Win.OSWin
	} else {
		win = oswin.TheApp.WindowInFocus()
	}

	fv.UpdatePath()
	pf := fv.PathField()
	if len(gi.SavedPaths) == 0 {
		gi.OpenPaths()
	}
	gi.SavedPaths.AddPath(fv.DirPath, gi.Prefs.SavedPathsMax)
	gi.SavePaths()
	sp := []string(gi.SavedPaths)
	sp = append(sp, fileViewResetPaths)
	pf.ItemsFromStringList(sp, true, 0)
	pf.SetText(fv.DirPath)
	sf := fv.SelField()
	sf.SetText(fv.SelFile)
	oswin.TheApp.Cursor(win).Push(cursor.Wait)
	defer oswin.TheApp.Cursor(win).Pop()

	effpath := fv.DirPath
	dpinfo, err := os.Lstat(effpath)
	if err != nil {
		log.Printf("gi.FileView Path: %v could not be opened -- error: %v\n", effpath, err)
		return
	}
	if dpinfo.Mode()&os.ModeSymlink != 0 {
		path, _ := filepath.Split(effpath)
		effpath, err = os.Readlink(effpath)
		if err != nil {
			log.Printf("gi.FileView Symbolic link path: %v could not be opened -- error: %v\n", effpath, err)
			return
		}
		effpath = filepath.Join(path, effpath)
	}

	fv.Files = make([]*FileInfo, 0, 1000)
	filepath.Walk(effpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("Path %q: Error: %v", effpath, err)
			// if fv.Viewport != nil {
			// 	gi.PromptDialog(fv.Viewport, "FileView UpdateFiles", emsg, true, false, nil, nil)
			// } else {
			log.Printf("gi.FileView error: %v\n", emsg)
			// }
			return nil // ignore
		}
		if path == effpath { // proceed..
			return nil
		}
		fi, ferr := NewFileInfo(path)
		keep := ferr == nil
		if fv.FilterFunc != nil {
			keep = fv.FilterFunc(fv, fi)
		}
		if keep {
			fv.Files = append(fv.Files, fi)
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})

	sv := fv.FilesView()
	sv.SelField = "Name"
	sv.SelVal = fv.SelFile
	sv.UpdateFromSlice()
	fv.SelectedIdx = sv.SelectedIdx
	if sv.SelectedIdx >= 0 {
		sv.ScrollToRow(sv.SelectedIdx)
	}
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
	if _, found := gi.Prefs.FavPaths.FindPath(dp); found {
		gi.PromptDialog(fv.Viewport, gi.DlgOpts{Title: "Add Path To Favorites", Prompt: fmt.Sprintf("Path is already on the favorites list: %v", dp)}, true, false, nil, nil)
		return
	}
	fi := gi.FavPathItem{"folder", fnm, dp}
	gi.Prefs.FavPaths = append(gi.Prefs.FavPaths, fi)
	gi.Prefs.Save()
	fv.FileSig.Emit(fv.This(), int64(FileViewFavAdded), fi)
	fv.UpdateFavs()
}

// DirPathUp moves up one directory in the path
func (fv *FileView) DirPathUp() {
	pdr, _ := filepath.Split(fv.DirPath)
	if pdr == "" {
		return
	}
	fv.DirPath = pdr
	fv.UpdateFilesAction()
}

// PathFieldHistPrev goes to the previous path in history
func (fv *FileView) PathFieldHistPrev() {
	pf := fv.PathField()
	pf.SelectItem(1) // todo: this doesn't quite work more than once, as history will update.
}

// PathFieldHistNext goes to the next path in history
func (fv *FileView) PathFieldHistNext() {
	pf := fv.PathField()
	pf.SelectItem(1) // todo: this doesn't work at all..
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
		gi.PromptDialog(fv.Viewport, gi.DlgOpts{Title: "FileView Error", Prompt: emsg}, true, false, nil, nil)
	}
	fv.FileSig.Emit(fv.This(), int64(FileViewNewFolder), fv.DirPath)
	fv.UpdateFilesAction()
}

// SetSelFileAction sets the currently selected file to given name, and sends
// selection action with current full file name, and updates selection in
// table view
func (fv *FileView) SetSelFileAction(sel string) {
	fv.SelFile = sel
	sv := fv.FilesView()
	ef := fv.ExtField()
	exts := ef.Text()
	if !sv.SelectFieldVal("Name", fv.SelFile) { // not found
		extl := strings.Split(exts, ",")
		if len(extl) == 1 {
			if !strings.HasSuffix(fv.SelFile, extl[0]) {
				fv.SelFile += extl[0]
			}
		}
	}
	fv.SelectedIdx = sv.SelectedIdx
	sf := fv.SelField()
	sf.SetText(fv.SelFile)
	fv.WidgetSig.Emit(fv.This(), int64(gi.WidgetSelected), fv.SelectedFile())
}

// FileSelectAction updates selection with given selected file and emits
// selected signal on WidgetSig with full name of selected item
func (fv *FileView) FileSelectAction(idx int) {
	if idx < 0 {
		return
	}
	fv.SaveSortPrefs()
	fi := fv.Files[idx]
	fv.SelectedIdx = idx
	fv.SelFile = fi.Name
	sf := fv.SelField()
	sf.SetText(fv.SelFile)
	fv.WidgetSig.Emit(fv.This(), int64(gi.WidgetSelected), fv.SelectedFile())
}

// SetExt updates the ext to given (list of, comma separated) extensions
func (fv *FileView) SetExt(ext string) {
	if ext == "" {
		if fv.SelFile != "" {
			ext = strings.ToLower(filepath.Ext(fv.SelFile))
		}
	}
	fv.Ext = ext
	exts := strings.Split(fv.Ext, ",")
	fv.ExtMap = make(map[string]string, len(exts))
	for _, ex := range exts {
		ex = strings.TrimSpace(ex)
		if len(ex) == 0 {
			continue
		}
		if ex[0] != '.' {
			ex = "." + ex
		}
		fv.ExtMap[strings.ToLower(ex)] = ex
	}
}

// SetExtAction sets the current extension to highlight, and redisplays files
func (fv *FileView) SetExtAction(ext string) {
	fv.SetExt(ext)
	fv.SetFullReRender()
	fv.UpdateFiles()
}

// FavSelect selects a favorite path and goes there
func (fv *FileView) FavSelect(idx int) {
	if idx < 0 || idx >= len(gi.Prefs.FavPaths) {
		return
	}
	fi := gi.Prefs.FavPaths[idx]
	fv.DirPath, _ = homedir.Expand(fi.Path)
	fv.UpdateFilesAction()
}

// SaveSortPrefs saves current sorting preferences
func (fv *FileView) SaveSortPrefs() {
	sv := fv.FilesView()
	if sv == nil {
		return
	}
	gi.Prefs.FileViewSort = sv.SortFieldName()
	// fmt.Printf("sort: %v\n", gi.Prefs.FileViewSort)
	gi.Prefs.Save()
}

func (fv *FileView) Style2D() {
	fv.Frame.Style2D()
	sf := fv.SelField()
	sf.StartFocus() // need to call this when window is actually active
	// if fv.Viewport != nil && fv.Viewport.IsDoingFullRender() {
	// 	fv.UpdateFiles()
	// }
}

func (fv *FileView) ConnectEvents2D() {
	fv.FileViewEvents()
}

func (fv *FileView) FileViewEvents() {
	fv.ConnectEvent(oswin.KeyChordEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		fvv := recv.Embed(KiT_FileView).(*FileView)
		kt := d.(*key.ChordEvent)
		fvv.KeyInput(kt)
	})
	fv.ConnectEvent(oswin.WindowShowEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		fvv := recv.Embed(KiT_FileView).(*FileView)
		fvv.UpdateFilesAction()
	})
}

func (fv *FileView) KeyInput(kt *key.ChordEvent) {
	if gi.KeyEventTrace {
		fmt.Printf("FileView KeyInput: %v\n", fv.PathUnique())
	}
	kf := gi.KeyFun(kt.Chord())
	switch {
	case kf == gi.KeyFunWordLeft:
		kt.SetProcessed()
		fv.DirPathUp()
	case kf == gi.KeyFunHistPrev:
		kt.SetProcessed()
		fv.PathFieldHistPrev()
	case kf == gi.KeyFunHistNext:
		kt.SetProcessed()
		fv.PathFieldHistNext()
	case kf == gi.KeyFunInsert:
		kt.SetProcessed()
		fv.SelectFile()
	case kf == gi.KeyFunMenuOpen:
		kt.SetProcessed()
		fv.SelectFile()
	}
}

func (fv *FileView) HasFocus2D() bool {
	return true // always.. we're typically a dialog anyway
}

////////////////////////////////////////////////////////////////////////////////
//  Completion

func (fv *FileView) FileComplete(data interface{}, text string, pos token.Position) (matches complete.Completions, seed string) {
	seedStart := 0
	for i := len(text) - 1; i >= 0; i-- {
		r := rune(text[i])
		if unicode.IsSpace(r) || r == filepath.Separator {
			seedStart = i + 1
			break
		}
	}
	seed = text[seedStart:]

	var files = []string{}
	for _, f := range fv.Files {
		files = append(files, f.Name)
	}

	if len(seed) > 0 { // return all directories
		files = complete.MatchSeedString(files, seed)
	}

	for _, d := range files {
		m := complete.Completion{Text: d}
		matches = append(matches, m)
	}
	return matches, seed
}

func (fv *FileView) PathComplete(data interface{}, path string, pos token.Position) (matches complete.Completions, seed string) {
	dir, seed := filepath.Split(path)
	d, err := os.Open(dir)
	if err != nil {
		return matches, seed
	}
	defer d.Close()

	files, err := ioutil.ReadDir(dir)
	var dirs = []string{}
	for _, f := range files {
		if f.IsDir() && !strings.HasPrefix(f.Name(), ".") {
			dirs = append(dirs, f.Name())
		}
	}

	if len(seed) > 0 { // return all directories
		dirs = complete.MatchSeedString(dirs, seed)
	}

	for _, d := range dirs {
		m := complete.Completion{Text: d}
		matches = append(matches, m)
	}
	return matches, seed
}

func (fv *FileView) PathCompleteEdit(data interface{}, text string, cursorPos int, completion string, seed string) (path string, delta int) {
	path, delta = complete.EditBasic(text, cursorPos, completion, seed)
	path = path + string(filepath.Separator)
	delta += 1
	return path, delta
}

func (fv *FileView) FileCompleteEdit(data interface{}, text string, cursorPos int, completion string, seed string) (file string, delta int) {
	file, delta = complete.EditWord(text, cursorPos, completion, seed)
	return file, delta
}

// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"unicode"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"
	"goki.dev/gi/v2/gi"
	"goki.dev/gicons"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/key"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/complete"
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

	// path to directory of files to display
	DirPath string `desc:"path to directory of files to display"`

	// selected file
	SelFile string `desc:"selected file"`

	// target extension(s) (comma separated if multiple, including initial .), if any
	Ext string `desc:"target extension(s) (comma separated if multiple, including initial .), if any"`

	// [view: -] optional styling function
	FilterFunc FileViewFilterFunc `view:"-" json:"-" xml:"-" desc:"optional styling function"`

	// map of lower-cased extensions from Ext -- used for highlighting files with one of these extensions -- maps onto original ext value
	ExtMap map[string]string `desc:"map of lower-cased extensions from Ext -- used for highlighting files with one of these extensions -- maps onto original ext value"`

	// files for current directory
	Files []*FileInfo `desc:"files for current directory"`

	// index of currently-selected file in Files list (-1 if none)
	SelectedIdx int `desc:"index of currently-selected file in Files list (-1 if none)"`

	// signal for file actions
	FileSig ki.Signal `desc:"signal for file actions"`

	// [view: -] change notify for current dir
	Watcher *fsnotify.Watcher `view:"-" desc:"change notify for current dir"`

	// [view: -] channel to close watcher watcher
	DoneWatcher chan bool `view:"-" desc:"channel to close watcher watcher"`

	// [view: -] UpdateFiles mutex
	UpdtMu sync.Mutex `view:"-" desc:"UpdateFiles mutex"`

	// [view: -] Previous path that was processed via UpdateFiles
	PrevPath string `view:"-" desc:"Previous path that was processed via UpdateFiles"`
}

func (fv *FileView) OnInit() {
	fv.Lay = gi.LayoutVert
	fv.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		fv.Spacing = gi.StdDialogVSpaceUnits
		s.SetStretchMax()
	})
}

func (fv *FileView) OnChildAdded(child ki.Ki) {
	if w := gi.KiAsWidget(child); w != nil {
		switch w.Name() {
		case "files-row":
			fr := child.(*gi.Layout)
			fr.Lay = gi.LayoutHoriz
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.SetStretchMax()
			})
		case "favs-view":
			fv := child.(*TableView)
			fv.ShowIndex = false
			fv.InactKeyNav = false // can only have one active -- files..
			fv.ShowToolBar = false
			fv.SetDisabled() // select only
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.SetStretchMaxHeight()
				s.MaxWidth.SetPx(0) // no stretch
			})
		case "files-view":
			fv := child.(*TableView)
			fv.ShowIndex = false // no index
			fv.ShowToolBar = false
			fv.SetDisabled() // select only
			fv.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.SetStretchMax()
			})
		case "sel-row":
			sr := child.(*gi.Layout)
			sr.Lay = gi.LayoutHoriz
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				sr.Spacing.SetPx(4 * gi.Prefs.DensityMul())
				s.SetStretchMaxWidth()
			})
		case "sel": // sel field
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.SetMinPrefWidth(units.Ch(60))
				s.SetStretchMaxWidth()
			})
		case "ext-label":
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.SetMinPrefWidth(units.Ch(10))
			})
		}
	}
}

func (fv *FileView) Disconnect() {
	if fv.Watcher != nil {
		fv.Watcher.Close()
		fv.Watcher = nil
	}
	if fv.DoneWatcher != nil {
		fv.DoneWatcher <- true
		close(fv.DoneWatcher)
		fv.DoneWatcher = nil
	}
	fv.Frame.Disconnect()
	fv.FileSig.DisconnectAll()
}

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
	fv.Config()
}

// SetPathFile sets the path, initial select file (or "") and initializes the view
func (fv *FileView) SetPathFile(path, file, ext string) {
	fv.DirPath = path
	fv.SelFile = file
	fv.SetExt(ext)
	fv.Config()
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

var FileViewProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}

// STYTODO: get rid of this or make it use actual color values
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

func FileViewStyleFunc(tv *TableView, slice any, widg gi.Node2D, row, col int, vv ValueView) {
	// STYTODO: get rid of this and move to OnChildAdded
	finf, ok := slice.([]*FileInfo)
	if ok {
		wi := widg.AsNode2D()
		if clr, got := FileViewKindColorMap[finf[row].Kind]; got {
			if _, err := wi.PropTry("color"); err != nil {
				wi.SetFullReRender()
			}
			wi.SetProp("color", clr)
			return
		}
		if fvv := tv.ParentByType(TypeFileView, ki.Embeds); fvv != nil {
			fv := fvv.Embed(TypeFileView).(*FileView)
			fn := finf[row].Name
			ext := strings.ToLower(filepath.Ext(fn))
			if _, has := fv.ExtMap[ext]; has {
				if _, err := wi.PropTry("color"); err != nil {
					wi.SetFullReRender()
				}
				wi.SetProp("color", "pref(link)")
				return
			}
		}
		if _, err := wi.PropTry("color"); err == nil {
			wi.SetFullReRender()
		}
		wi.DeleteProp("color")
	}
}

// Config configures the view
func (fv *FileView) Config() {
	config := ki.TypeAndNameList{}
	config.Add(gi.TypeToolBar, "path-tbar")
	config.Add(gi.TypeLayout, "files-row")
	config.Add(gi.TypeLayout, "sel-row")
	mods, updt := fv.ConfigChildren(config)
	if mods {
		fv.ConfigPathBar()
		fv.ConfigFilesRow()
		fv.ConfigSelRow()
		fv.UpdateFiles()
		fv.UpdateEnd(updt)
	}
}

func (fv *FileView) ConfigPathBar() {
	pr := fv.ChildByName("path-tbar", 0).(*gi.ToolBar)
	if pr.HasChildren() {
		return
	}
	pr.Lay = gi.LayoutHoriz
	pr.SetStretchMaxWidth()

	config := ki.TypeAndNameList{}
	config.Add(gi.TypeLabel, "path-lbl")
	config.Add(gi.TypeComboBox, "path")
	config.Add(gi.TypeAction, "path-up")
	config.Add(gi.TypeAction, "path-ref")
	config.Add(gi.TypeAction, "path-fav")
	config.Add(gi.TypeAction, "new-folder")

	pl := gi.NewLabel(pr, "path-lbl", "Path:")
	pl.Tooltip = "Path to look for files in: can select from list of recent paths, or edit a value directly"
	pf := gi.NewComboBox(pr, "path")
	pf.Editable = true
	pf.SetMinPrefWidth(units.Ch(60))
	pf.SetStretchMaxWidth()
	pf.ConfigParts()
	pft, found := pf.TextField()
	if found {
		pft.SetCompleter(fv, fv.PathComplete, fv.PathCompleteEdit)
		pft.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.TextFieldDone) {
				fvv, _ := recv.Embed(TypeFileView).(*FileView)
				pff, _ := send.(*gi.TextField)
				fvv.DirPath = pff.Text()
				fvv.UpdateFilesAction()
			}
		})
	}
	pf.ComboSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		fvv, _ := recv.Embed(TypeFileView).(*FileView)
		pff := send.Embed(gi.TypeComboBox).(*gi.ComboBox)
		sp := data.(string)
		if sp == gi.FileViewResetPaths {
			gi.SavedPaths = make(gi.FilePaths, 1, gi.Prefs.Params.SavedPathsMax)
			gi.SavedPaths[0] = fvv.DirPath
			pff.ItemsFromStringList(([]string)(gi.SavedPaths), true, 0)
			gi.StringsAddExtras((*[]string)(&gi.SavedPaths), gi.SavedPathsExtras)
			fv.UpdateFiles()
		} else if sp == gi.FileViewEditPaths {
			fv.EditPaths()
			pff.ItemsFromStringList(([]string)(gi.SavedPaths), true, 0)
		} else {
			fvv.DirPath = sp
			fvv.UpdateFilesAction()
		}
	})

	pr.AddAction(gi.ActOpts{Name: "path-up", Icon: gicons.ArrowUpward, Tooltip: "go up one level into the parent folder", ShortcutKey: gi.KeyFunJump}, fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		fvv, _ := recv.Embed(TypeFileView).(*FileView)
		fvv.DirPathUp()
	})

	pr.AddAction(gi.ActOpts{Name: "path-ref", Icon: gicons.Refresh, Tooltip: "Update directory view -- in case files might have changed"}, fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		fvv, _ := recv.Embed(TypeFileView).(*FileView)
		fvv.UpdateFilesAction()
	})

	pr.AddAction(gi.ActOpts{Name: "path-fav", Icon: gicons.Favorite, Tooltip: "save this path to the favorites list -- saves current Prefs"}, fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		fvv, _ := recv.Embed(TypeFileView).(*FileView)
		fvv.AddPathToFavs()
	})

	pr.AddAction(gi.ActOpts{Name: "new-folder", Icon: gicons.CreateNewFolder, Tooltip: "Create a new folder in this folder"},
		fv.This(), func(recv, send ki.Ki, sig int64, data any) {
			fvv, _ := recv.Embed(TypeFileView).(*FileView)
			fvv.NewFolder()
		})
}

func (fv *FileView) ConfigFilesRow() {
	fr := fv.FilesRow()
	config := ki.TypeAndNameList{}
	config.Add(TypeTableView, "favs-view")
	config.Add(TypeTableView, "files-view")
	fr.ConfigChildren(config) // already covered by parent update

	sv := fv.FavsView()
	sv.SelectedIdx = -1
	sv.SetSlice(&gi.Prefs.FavPaths)
	sv.WidgetSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.WidgetSelected) {
			fvv, _ := recv.Embed(TypeFileView).(*FileView)
			svv, _ := send.(*TableView)
			fvv.FavSelect(svv.SelectedIdx)
		}
	})

	sv = fv.FilesView()
	sv.StyleFunc = FileViewStyleFunc
	sv.SetSlice(&fv.Files)
	if gi.Prefs.FileViewSort != "" {
		sv.SetSortFieldName(gi.Prefs.FileViewSort)
	}
	sv.WidgetSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.WidgetSelected) {
			fvv, _ := recv.Embed(TypeFileView).(*FileView)
			svv, _ := send.(*TableView)
			fvv.FileSelectAction(svv.SelectedIdx)
		}
	})
	sv.SliceViewSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(SliceViewDoubleClicked) {
			fvv, _ := recv.Embed(TypeFileView).(*FileView)
			fvv.SelectFile()
		}
	})
}

func (fv *FileView) ConfigSelRow() {
	sr := fv.SelRow()
	config := ki.TypeAndNameList{}
	config.Add(gi.TypeLabel, "sel-lbl")
	config.Add(gi.TypeTextField, "sel")
	config.Add(gi.TypeLabel, "ext-lbl")
	config.Add(gi.TypeTextField, "ext")
	sr.ConfigChildren(config) // already covered by parent update

	sl := sr.ChildByName("sel-lbl", 0).(*gi.Label)
	sl.Text = "File:"
	sl.Tooltip = "enter file name here (or select from above list)"
	sf := fv.SelField()
	sf.Tooltip = fmt.Sprintf("enter file name.  special keys: up/down to move selection; %v or %v to go up to parent folder; %v or %v or %v or %v to select current file (if directory, goes into it, if file, selects and closes); %v or %v for prev / next history item; %s return to this field", gi.ShortcutForFun(gi.KeyFunWordLeft), gi.ShortcutForFun(gi.KeyFunJump), gi.ShortcutForFun(gi.KeyFunSelectMode), gi.ShortcutForFun(gi.KeyFunInsert), gi.ShortcutForFun(gi.KeyFunInsertAfter), gi.ShortcutForFun(gi.KeyFunMenuOpen), gi.ShortcutForFun(gi.KeyFunHistPrev), gi.ShortcutForFun(gi.KeyFunHistNext), gi.ShortcutForFun(gi.KeyFunSearch))
	sf.SetCompleter(fv, fv.FileComplete, fv.FileCompleteEdit)
	sf.SetText(fv.SelFile)
	sf.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			fvv, _ := recv.Embed(TypeFileView).(*FileView)
			pff, _ := send.(*gi.TextField)
			fvv.SetSelFileAction(pff.Text())
		}
	})
	sf.StartFocus()

	el := sr.ChildByName("ext-lbl", 0).(*gi.Label)
	el.Text = "Ext(s):"
	el.Tooltip = "target extension(s) to highlight -- if multiple, separate with commas, and do include the . at the start"
	ef := fv.ExtField()
	ef.SetText(fv.Ext)
	ef.TextFieldSig.Connect(fv.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			fvv, _ := recv.Embed(TypeFileView).(*FileView)
			pff, _ := send.(*gi.TextField)
			fvv.SetExtAction(pff.Text())
		}
	})
}

func (fv *FileView) ConfigWatcher() error {
	if fv.Watcher != nil {
		return nil
	}
	var err error
	fv.Watcher, err = fsnotify.NewWatcher()
	return err
}

func (fv *FileView) WatchWatcher() {
	if fv.Watcher == nil || fv.Watcher.Events == nil {
		return
	}
	if fv.DoneWatcher != nil {
		return
	}
	fv.DoneWatcher = make(chan bool)
	go func() {
		watch := fv.Watcher
		done := fv.DoneWatcher
		for {
			select {
			case <-done:
				return
			case event := <-watch.Events:
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Rename == fsnotify.Rename:
					fv.UpdateFiles()
				}
			case err := <-watch.Errors:
				_ = err
			}
		}
	}()
}

// PathField returns the ComboBox of the path
func (fv *FileView) PathField() *gi.ComboBox {
	pr := fv.ChildByName("path-tbar", 0).(*gi.ToolBar)
	return pr.ChildByName("path", 1).(*gi.ComboBox)
}

func (fv *FileView) FilesRow() *gi.Layout {
	return fv.ChildByName("files-row", 2).(*gi.Layout)
}

// FavsView returns the TableView of the favorites
func (fv *FileView) FavsView() *TableView {
	return fv.FilesRow().ChildByName("favs-view", 1).(*TableView)
}

// FilesView returns the TableView of the files
func (fv *FileView) FilesView() *TableView {
	return fv.FilesRow().ChildByName("files-view", 1).(*TableView)
}

func (fv *FileView) SelRow() *gi.Layout {
	return fv.ChildByName("sel-row", 4).(*gi.Layout)
}

// SelField returns the TextField of the selected file
func (fv *FileView) SelField() *gi.TextField {
	return fv.SelRow().ChildByName("sel", 1).(*gi.TextField)
}

// ExtField returns the TextField of the extension
func (fv *FileView) ExtField() *gi.TextField {
	return fv.SelRow().ChildByName("ext", 2).(*gi.TextField)
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

// UpdateFiles updates list of files and other views for current path
func (fv *FileView) UpdateFiles() {
	fv.UpdtMu.Lock()
	defer fv.UpdtMu.Unlock()

	updt := fv.UpdateStart()
	defer fv.UpdateEnd(updt)
	var owin goosi.Window
	win := fv.ParentWindow()
	if win != nil {
		owin = fv.Viewport.Win.OSWin
	} else {
		owin = goosi.TheApp.WindowInFocus()
	}

	fv.UpdatePath()
	pf := fv.PathField()
	if len(gi.SavedPaths) == 0 {
		gi.OpenPaths()
	}
	gi.SavedPaths.AddPath(fv.DirPath, gi.Prefs.Params.SavedPathsMax)
	gi.SavePaths()
	sp := []string(gi.SavedPaths)
	pf.ItemsFromStringList(sp, true, 0)
	pf.SetText(fv.DirPath)
	sf := fv.SelField()
	sf.SetText(fv.SelFile)
	goosi.TheApp.Cursor(owin).Push(cursor.Wait)
	defer goosi.TheApp.Cursor(owin).Pop()

	effpath, err := filepath.EvalSymlinks(fv.DirPath)
	if err != nil {
		log.Printf("gi.FileView Path: %v could not be opened -- error: %v\n", effpath, err)
		return
	}
	_, err = os.Lstat(effpath)
	if err != nil {
		log.Printf("gi.FileView Path: %v could not be opened -- error: %v\n", effpath, err)
		return
	}

	fv.Files = make([]*FileInfo, 0, 1000)
	filepath.Walk(effpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("Path %q: Error: %v", effpath, err)
			// if fv.Viewport != nil {
			// 	gi.PromptDialog(fv.Viewport, "FileView UpdateFiles", emsg, gi.AddOk, gi.NoCancel, nil, nil)
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

	fvv := fv.FavsView()
	fvv.ResetSelectedIdxs()

	sv := fv.FilesView()
	sv.ResetSelectedIdxs()
	sv.SelField = "Name"
	sv.SelVal = fv.SelFile
	sv.SortSlice()
	if !sv.IsConfiged() {
		sv.Config()
		sv.LayoutSliceGrid()
	}
	sv.UpdateSliceGrid()
	sv.LayoutHeader()
	fv.SelectedIdx = sv.SelectedIdx
	if sv.SelectedIdx >= 0 {
		sv.ScrollToIdx(sv.SelectedIdx)
	}

	if fv.PrevPath != fv.DirPath {
		if goosi.TheApp.Platform() != goosi.MacOS {
			// mac is not supported in a high-capacity fashion at this point
			if fv.PrevPath == "" {
				fv.ConfigWatcher()
			} else {
				fv.Watcher.Remove(fv.PrevPath)
			}
			fv.Watcher.Add(fv.DirPath)
			if fv.PrevPath == "" {
				fv.WatchWatcher()
			}
		}
		fv.PrevPath = fv.DirPath
	}
}

// UpdateFavs updates list of files and other views for current path
func (fv *FileView) UpdateFavs() {
	sv := fv.FavsView()
	sv.UpdateSliceGrid()
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
		gi.PromptDialog(fv.Viewport, gi.DlgOpts{Title: "Add Path To Favorites", Prompt: fmt.Sprintf("Path is already on the favorites list: %v", dp)}, gi.AddOk, gi.NoCancel, nil, nil)
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
	pf.SelectItemAction(1) // todo: this doesn't quite work more than once, as history will update.
}

// PathFieldHistNext goes to the next path in history
func (fv *FileView) PathFieldHistNext() {
	pf := fv.PathField()
	pf.SelectItemAction(1) // todo: this doesn't work at all..
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
		gi.PromptDialog(fv.Viewport, gi.DlgOpts{Title: "FileView Error", Prompt: emsg}, gi.AddOk, gi.NoCancel, nil, nil)
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

func (fv *FileView) SetStyle() {
	fv.Frame.SetStyle()
	sf := fv.SelField()
	sf.StartFocus() // need to call this when window is actually active
}

func (fv *FileView) ConnectEvents() {
	fv.FileViewEvents()
}

func (fv *FileView) FileViewEvents() {
	fv.ConnectEvent(goosi.KeyChordEvent, gi.LowPri, func(recv, send ki.Ki, sig int64, d any) {
		fvv := recv.Embed(TypeFileView).(*FileView)
		kt := d.(*key.ChordEvent)
		fvv.KeyInput(kt)
	})
}

func (fv *FileView) KeyInput(kt *key.ChordEvent) {
	if gi.KeyEventTrace {
		fmt.Printf("FileView KeyInput: %v\n", fv.Path())
	}
	kf := gi.KeyFun(kt.Chord())
	switch kf {
	case gi.KeyFunJump, gi.KeyFunWordLeft:
		kt.SetProcessed()
		fv.DirPathUp()
	case gi.KeyFunHistPrev:
		kt.SetProcessed()
		fv.PathFieldHistPrev()
	case gi.KeyFunHistNext:
		kt.SetProcessed()
		fv.PathFieldHistNext()
	case gi.KeyFunInsert, gi.KeyFunInsertAfter, gi.KeyFunMenuOpen, gi.KeyFunSelectMode:
		kt.SetProcessed()
		fv.SelectFile()
	case gi.KeyFunSearch:
		kt.SetProcessed()
		sf := fv.SelField()
		sf.GrabFocus()
	}
}

func (fv *FileView) HasFocus2D() bool {
	return true // always.. we're typically a dialog anyway
}

////////////////////////////////////////////////////////////////////////////////
//  Completion

// FileComplete finds the possible completions for the file field
func (fv *FileView) FileComplete(data any, text string, posLn, posCh int) (md complete.Matches) {
	seedStart := 0
	for i := len(text) - 1; i >= 0; i-- {
		r := rune(text[i])
		if unicode.IsSpace(r) || r == filepath.Separator {
			seedStart = i + 1
			break
		}
	}
	md.Seed = text[seedStart:]

	var files = []string{}
	for _, f := range fv.Files {
		files = append(files, f.Name)
	}

	if len(md.Seed) > 0 { // return all directories
		files = complete.MatchSeedString(files, md.Seed)
	}

	for _, d := range files {
		m := complete.Completion{Text: d}
		md.Matches = append(md.Matches, m)
	}
	return md
}

// PathComplete finds the possible completions for the path field
func (fv *FileView) PathComplete(data any, path string, posLn, posCh int) (md complete.Matches) {
	dir, seed := filepath.Split(path)
	md.Seed = seed
	d, err := os.Open(dir)
	if err != nil {
		return md
	}
	defer d.Close()

	files, err := ioutil.ReadDir(dir)
	var dirs = []string{}
	for _, f := range files {
		if f.IsDir() && !strings.HasPrefix(f.Name(), ".") {
			dirs = append(dirs, f.Name())
		}
	}

	if len(md.Seed) > 0 { // return all directories
		dirs = complete.MatchSeedString(dirs, md.Seed)
	}

	for _, d := range dirs {
		m := complete.Completion{Text: d}
		md.Matches = append(md.Matches, m)
	}
	return md
}

// PathCompleteEdit is the editing function called when inserting the completion selection in the path field
func (fv *FileView) PathCompleteEdit(data any, text string, cursorPos int, c complete.Completion, seed string) (ed complete.Edit) {
	ed = complete.EditWord(text, cursorPos, c.Text, seed)
	path := ed.NewText + string(filepath.Separator)
	ed.NewText = path
	ed.CursorAdjust += 1
	return ed
}

// FileCompleteEdit is the editing function called when inserting the completion selection in the file field
func (fv *FileView) FileCompleteEdit(data any, text string, cursorPos int, c complete.Completion, seed string) (ed complete.Edit) {
	ed = complete.EditWord(text, cursorPos, c.Text, seed)
	return ed
}

// EditPaths displays a dialog allowing user to delete paths from the path list
func (fv *FileView) EditPaths() {
	tmp := make([]string, len(gi.SavedPaths))
	copy(tmp, gi.SavedPaths)
	gi.StringsRemoveExtras((*[]string)(&tmp), gi.SavedPathsExtras)
	opts := DlgOpts{Title: "Recent File Paths", Prompt: "Delete paths you no longer use", Ok: true, Cancel: true, NoAdd: true}
	SliceViewDialog(fv.Viewport, &tmp, opts,
		nil, fv, func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				gi.SavedPaths = nil
				gi.SavedPaths = append(gi.SavedPaths, tmp...)
				// add back the reset/edit menu items
				gi.StringsAddExtras((*[]string)(&gi.SavedPaths), gi.SavedPathsExtras)
				gi.SavePaths()
				fv.UpdateFiles()
			}
		})
}

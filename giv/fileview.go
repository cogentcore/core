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
	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/complete"
	"goki.dev/pi/v2/filecat"
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
	DirPath string

	// selected file
	SelFile string

	// target extension(s) (comma separated if multiple, including initial .), if any
	Ext string `set:"-"`

	// optional styling function
	FilterFunc FileViewFilterFunc `view:"-" json:"-" xml:"-"`

	// map of lower-cased extensions from Ext -- used for highlighting files with one of these extensions -- maps onto original ext value
	ExtMap map[string]string

	// files for current directory
	Files []*filecat.FileInfo

	// index of currently-selected file in Files list (-1 if none)
	SelectedIdx int `set:"-" edit:"-"`

	// signal for file actions
	// FileSig ki.Signal `desc:"signal for file actions"`

	// change notify for current dir
	Watcher *fsnotify.Watcher `set:"-" view:"-"`

	// channel to close watcher watcher
	DoneWatcher chan bool `set:"-" view:"-"`

	// UpdateFiles mutex
	UpdtMu sync.Mutex `set:"-" view:"-"`

	// Previous path that was processed via UpdateFiles
	PrevPath string `set:"-" view:"-"`
}

func (fv *FileView) OnInit() {
	fv.HandleFileViewEvents()
	fv.FileViewStyles()
}

func (fv *FileView) FileViewStyles() {
	fv.Lay = gi.LayoutVert
	fv.Style(func(s *styles.Style) {
		fv.Spacing = gi.StdDialogVSpaceUnits
		s.SetStretchMax()
	})
	fv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(fv) {
		case "files-row":
			fr := w.(*gi.Layout)
			fr.Lay = gi.LayoutHoriz
			w.Style(func(s *styles.Style) {
				s.SetStretchMax()
			})
		case "files-row/favs-view":
			fv := w.(*TableView)
			fv.SetFlag(false, SliceViewShowIndex)
			fv.SetFlag(false, SliceViewReadOnlyKeyNav) // can only have one active -- files..
			fv.SetFlag(false, SliceViewShowToolbar)
			fv.SetState(true, states.ReadOnly)
			w.Style(func(s *styles.Style) {
				s.SetStretchMaxHeight()
				s.SetFixedWidth(units.Ch(25))
			})
		case "files-row/files-view":
			fv := w.(*TableView)
			fv.SetFlag(false, SliceViewShowIndex)
			fv.SetFlag(false, SliceViewShowToolbar)
			fv.SetState(true, states.ReadOnly)
			fv.Style(func(s *styles.Style) {
				s.SetStretchMax()
			})
		case "sel-row":
			sr := w.(*gi.Layout)
			sr.Lay = gi.LayoutHoriz
			w.Style(func(s *styles.Style) {
				sr.Spacing.Dp(4)
				s.SetStretchMaxWidth()
			})
		case "sel-row/sel": // sel field
			w.Style(func(s *styles.Style) {
				s.SetMinPrefWidth(units.Ch(60))
				s.SetStretchMaxWidth()
			})
		case "sel-row/ext-label":
			w.Style(func(s *styles.Style) {
				s.SetMinPrefWidth(units.Ch(10))
			})
		}
	})
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
}

// FileViewFilterFunc is a filtering function for files -- returns true if the
// file should be visible in the view, and false if not
type FileViewFilterFunc func(fv *FileView, fi *filecat.FileInfo) bool

// FileViewDirOnlyFilter is a FileViewFilterFunc that only shows directories (folders).
func FileViewDirOnlyFilter(fv *FileView, fi *filecat.FileInfo) bool {
	return fi.IsDir()
}

// FileViewExtOnlyFilter is a FileViewFilterFunc that only shows files that
// match the target extensions, and directories.
func FileViewExtOnlyFilter(fv *FileView, fi *filecat.FileInfo) bool {
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
}

// SetPathFile sets the path, initial select file (or "") and initializes the view
func (fv *FileView) SetPathFile(path, file, ext string) {
	fv.DirPath = path
	fv.SelFile = file
	fv.SetExt(ext)
}

// SelectedFile returns the full path to selected file
func (fv *FileView) SelectedFile() string {
	return filepath.Join(fv.DirPath, fv.SelFile)
}

// SelectedFileInfo returns the currently-selected fileinfo, returns
// false if none
func (fv *FileView) SelectedFileInfo() (*filecat.FileInfo, bool) {
	if fv.SelectedIdx < 0 || fv.SelectedIdx >= len(fv.Files) {
		return nil, false
	}
	return fv.Files[fv.SelectedIdx], true
}

// SelectFile selects the current file as the selection.
// if a directory it opens the directory and returns false.
// if a file it selects the file and returns true.
// if no selection, returns false.
func (fv *FileView) SelectFile() bool {
	if fi, ok := fv.SelectedFileInfo(); ok {
		if fi.IsDir() {
			fv.DirPath = filepath.Join(fv.DirPath, fi.Name)
			fv.SelFile = ""
			fv.SelectedIdx = -1
			fv.UpdateFilesAction()
			return false
		}
		return true
	}
	return false
}

// STYTODO: get rid of this or make it use actual color values
// FileViewKindColorMap translates file Kinds into different colors for the file viewer
var FileViewKindColorMap = map[string]string{
	"folder": "pref(link)",
}

func (fv *FileView) ConfigWidget(sc *gi.Scene) {
	fv.ConfigFileView(sc)
}

func (fv *FileView) ConfigFileView(sc *gi.Scene) {
	if fv.HasChildren() {
		return
	}
	config := ki.Config{}
	config.Add(gi.ToolbarType, "path-tbar")
	config.Add(gi.LayoutType, "files-row")
	config.Add(gi.LayoutType, "sel-row")
	mods, updt := fv.ConfigChildren(config)
	if mods {
		fv.ConfigPathBar()
		fv.ConfigFilesRow()
		fv.ConfigSelRow()
		fv.UpdateFiles()
		fv.UpdateEndLayout(updt)
		// fv.Update()
	}
}

func (fv *FileView) ConfigPathBar() {
	pr := fv.ChildByName("path-tbar", 0).(*gi.Toolbar)
	if pr.HasChildren() {
		return
	}
	pr.Lay = gi.LayoutHoriz
	pr.SetStretchMaxWidth()

	config := ki.Config{}
	config.Add(gi.LabelType, "path-lbl")
	config.Add(gi.ChooserType, "path")
	config.Add(gi.ButtonType, "path-up")
	config.Add(gi.ButtonType, "path-ref")
	config.Add(gi.ButtonType, "path-fav")
	config.Add(gi.ButtonType, "new-folder")

	pl := gi.NewLabel(pr, "path-lbl").SetText("Path:")
	pl.Tooltip = "Path to look for files in: can select from list of recent paths, or edit a value directly"
	pf := gi.NewChooser(pr, "path").SetEditable(true)
	pf.SetMinPrefWidth(units.Ch(60))
	pf.SetStretchMaxWidth()
	pf.ConfigParts(fv.Sc)
	pft, found := pf.TextField()
	if found {
		pft.SetCompleter(fv, fv.PathComplete, fv.PathCompleteEdit)
		pft.OnChange(func(e events.Event) {
			fv.DirPath = pft.Text()
			fv.UpdateFilesAction()
		})
	}
	pf.OnChange(func(e events.Event) {
		sp := pf.CurVal.(string)
		if sp == gi.FileViewResetPaths {
			gi.SavedPaths = make(gi.FilePaths, 1, gi.Prefs.Params.SavedPathsMax)
			gi.SavedPaths[0] = fv.DirPath
			pf.ItemsFromStringList(([]string)(gi.SavedPaths), true, 0)
			gi.StringsAddExtras((*[]string)(&gi.SavedPaths), gi.SavedPathsExtras)
			fv.UpdateFiles()
		} else if sp == gi.FileViewEditPaths {
			fv.EditPaths()
			pf.ItemsFromStringList(([]string)(gi.SavedPaths), true, 0)
		} else {
			fv.DirPath = sp
			fv.UpdateFilesAction()
		}
	})

	gi.NewButton(pr, "path-up").SetIcon(icons.ArrowUpward).SetKey(keyfun.Jump).SetTooltip("go up one level into the parent folder").
		OnClick(func(e events.Event) {
			fv.DirPathUp()
		})

	gi.NewButton(pr, "path-ref").SetIcon(icons.Refresh).SetTooltip("Update directory view -- in case files might have changed").
		OnClick(func(e events.Event) {
			fv.UpdateFilesAction()
		})

	gi.NewButton(pr, "path-fav").SetIcon(icons.Favorite).SetTooltip("save this path to the favorites list -- saves current Prefs").
		OnClick(func(e events.Event) {
			fv.AddPathToFavs()
		})

	gi.NewButton(pr, "new-folder").SetIcon(icons.CreateNewFolder).SetTooltip("Create a new folder in this folder").
		OnClick(func(e events.Event) {
			fv.NewFolder()
		})
}

func (fv *FileView) ConfigFilesRow() {
	fr := fv.FilesRow()
	config := ki.Config{}
	config.Add(TableViewType, "favs-view")
	config.Add(TableViewType, "files-view")
	fr.ConfigChildren(config) // already covered by parent update

	sv := fv.FavsView()
	sv.SelectedIdx = -1
	sv.SetState(true, states.ReadOnly)
	sv.SetSlice(&gi.Prefs.FavPaths)
	sv.OnSelect(func(e events.Event) {
		fv.FavSelect(sv.SelectedIdx)
	})

	fv.ReadFiles()
	fsv := fv.FilesView()
	fsv.SetState(true, states.ReadOnly)
	fsv.SetSlice(&fv.Files)
	fsv.StyleFunc = func(w gi.Widget, s *styles.Style, row, col int) {
		if clr, got := FileViewKindColorMap[fv.Files[row].Kind]; got {
			s.Color = grr.Log(colors.FromName(clr))
			return
		}
		fn := fv.Files[row].Name
		ext := strings.ToLower(filepath.Ext(fn))
		if _, has := fv.ExtMap[ext]; has {
			s.Color = colors.Scheme.Primary.Base
		} else {
			s.Color = colors.Scheme.OnSurface
		}
	}
	if gi.Prefs.FileViewSort != "" {
		fsv.SetSortFieldName(gi.Prefs.FileViewSort)
	}
	fsv.OnSelect(func(e events.Event) {
		fv.FileSelectAction(fsv.SelectedIdx)
	})
	fsv.OnDoubleClick(func(e events.Event) {
		if !fv.SelectFile() {
			e.SetHandled() // don't pass along; keep dialog open
		}
	})
}

func (fv *FileView) ConfigSelRow() {
	sr := fv.SelRow()
	config := ki.Config{}
	config.Add(gi.LabelType, "sel-lbl")
	config.Add(gi.TextFieldType, "sel")
	config.Add(gi.LabelType, "ext-lbl")
	config.Add(gi.TextFieldType, "ext")
	sr.ConfigChildren(config) // already covered by parent update

	sl := sr.ChildByName("sel-lbl", 0).(*gi.Label)
	sl.Text = "File:"
	sl.Tooltip = "enter file name here (or select from above list)"
	sf := fv.SelField()
	sf.Tooltip = fmt.Sprintf("enter file name.  special keys: up/down to move selection; %v or %v to go up to parent folder; %v or %v or %v or %v to select current file (if directory, goes into it, if file, selects and closes); %v or %v for prev / next history item; %s return to this field", keyfun.ShortcutFor(keyfun.WordLeft), keyfun.ShortcutFor(keyfun.Jump), keyfun.ShortcutFor(keyfun.SelectMode), keyfun.ShortcutFor(keyfun.Insert), keyfun.ShortcutFor(keyfun.InsertAfter), keyfun.ShortcutFor(keyfun.Open), keyfun.ShortcutFor(keyfun.HistPrev), keyfun.ShortcutFor(keyfun.HistNext), keyfun.ShortcutFor(keyfun.Search))
	sf.SetCompleter(fv, fv.FileComplete, fv.FileCompleteEdit)
	sf.SetText(fv.SelFile)
	sf.OnChange(func(e events.Event) {
		fv.SetSelFileAction(sf.Text())
	})
	sf.StartFocus()

	el := sr.ChildByName("ext-lbl", 0).(*gi.Label)
	el.Text = "Ext(s):"
	el.Tooltip = "target extension(s) to highlight -- if multiple, separate with commas, and do include the . at the start"
	ef := fv.ExtField()
	ef.SetText(fv.Ext)
	ef.OnChange(func(e events.Event) {
		fv.SetExtAction(ef.Text())
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

// PathField returns the chooser of the path
func (fv *FileView) PathField() *gi.Chooser {
	pr := fv.ChildByName("path-tbar", 0).(*gi.Toolbar)
	return pr.ChildByName("path", 1).(*gi.Chooser)
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
	fv.UpdateFiles()
	sf := fv.SelField()
	sf.GrabFocus()
}

func (fv *FileView) ReadFiles() {
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

	fv.Files = make([]*filecat.FileInfo, 0, 1000)
	filepath.Walk(effpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("Path %q: Error: %v", effpath, err)
			// if fv.Scene != nil {
			// 	gi.PromptDialog(fv, gi.DlgOpts{Title: "FileView UpdateFiles", emsg, Ok: true, Cancel: false}, nil)
			// } else {
			log.Printf("gi.FileView error: %v\n", emsg)
			// }
			return nil // ignore
		}
		if path == effpath { // proceed..
			return nil
		}
		fi, ferr := filecat.NewFileInfo(path)
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
}

// UpdateFiles updates list of files and other views for current path
func (fv *FileView) UpdateFiles() {
	fv.UpdtMu.Lock()
	defer fv.UpdtMu.Unlock()

	updt := fv.UpdateStart()
	defer fv.UpdateEnd(updt)

	fv.UpdatePath()
	pf := fv.PathField()
	if len(gi.SavedPaths) == 0 {
		gi.OpenPaths()
	}
	gi.SavedPaths.AddPath(fv.DirPath, gi.Prefs.Params.SavedPathsMax)
	gi.SavePaths()
	sp := []string(gi.SavedPaths)
	pf.ItemsFromStringList(sp, true, 0)
	pf.ShowCurVal(fv.DirPath)
	sf := fv.SelField()
	sf.SetText(fv.SelFile)

	// todo: wait cursor
	// goosi.TheApp.Cursor(owin).Push(cursor.Wait)
	// defer goosi.TheApp.Cursor(owin).Pop()

	fv.ReadFiles()

	fvv := fv.FavsView()
	fvv.ResetSelectedIdxs()

	sv := fv.FilesView()
	sv.ResetSelectedIdxs()
	sv.SelField = "Name"
	sv.SelVal = fv.SelFile
	sv.SortSlice()
	sv.Update()

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
	sv.Update()
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
		// TODO(kai/snack)
		// gi.PromptDialog(fv, gi.DlgOpts{Title: "Add Path To Favorites", Prompt: fmt.Sprintf("Path is already on the favorites list: %v", dp), Ok: true, Cancel: false}, nil)
		return
	}
	fi := gi.FavPathItem{"folder", fnm, dp}
	gi.Prefs.FavPaths = append(gi.Prefs.FavPaths, fi)
	gi.Prefs.Save()
	// fv.FileSig.Emit(fv.This(), int64(FileViewFavAdded), fi)
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
		// TODO(kai/snack)
		// emsg := fmt.Sprintf("NewFolder at: %q: Error: %v", fv.DirPath, err)
		// gi.PromptDialog(fv, gi.DlgOpts{Title: "FileView Error", Prompt: emsg, Ok: true, Cancel: false}, nil)
	}
	// fv.FileSig.Emit(fv.This(), int64(FileViewNewFolder), fv.DirPath)
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
	fv.Send(events.Select) // receiver needs to get selectedFile
	// fv.WidgetSig.Emit(fv.This(), int64(gi.WidgetSelected), fv.SelectedFile())
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
	fv.Send(events.Select)
	// fv.WidgetSig.Emit(fv.This(), int64(gi.WidgetSelected), fv.SelectedFile())
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

func (fv *FileView) ApplyStyle(sc *gi.Scene) {
	fv.Frame.ApplyStyle(sc)
	sf := fv.SelField()
	sf.StartFocus() // need to call this when window is actually active
}

func (fv *FileView) HandleFileViewEvents() {
	fv.OnKeyChord(func(e events.Event) {
		fv.KeyInput(e)
	})
}

func (fv *FileView) KeyInput(kt events.Event) {
	if gi.KeyEventTrace {
		fmt.Printf("FileView KeyInput: %v\n", fv.Path())
	}
	kf := keyfun.Of(kt.KeyChord())
	switch kf {
	case keyfun.Jump, keyfun.WordLeft:
		kt.SetHandled()
		fv.DirPathUp()
	case keyfun.HistPrev:
		kt.SetHandled()
		fv.PathFieldHistPrev()
	case keyfun.HistNext:
		kt.SetHandled()
		fv.PathFieldHistNext()
	case keyfun.Insert, keyfun.InsertAfter, keyfun.Open, keyfun.SelectMode:
		kt.SetHandled()
		if fv.SelectFile() {
			fv.Send(events.DoubleClick, kt) // will close dialog
		}
	case keyfun.Search:
		kt.SetHandled()
		sf := fv.SelField()
		sf.GrabFocus()
	}
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
	d := gi.NewDialog(fv).Title("Recent File Paths").Prompt("Delete paths you no longer use")
	NewSliceView(d).SetSlice(&tmp).SetFlag(true, SliceViewNoAdd)
	d.Cancel().Ok().OnAccept(func(e events.Event) {
		gi.SavedPaths = nil
		gi.SavedPaths = append(gi.SavedPaths, tmp...)
		// add back the reset/edit menu items
		gi.StringsAddExtras((*[]string)(&gi.SavedPaths), gi.SavedPathsExtras)
		gi.SavePaths()
		fv.UpdateFiles()
	}).Run()
}

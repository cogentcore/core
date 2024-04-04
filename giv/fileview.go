// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/pi/complete"
	"cogentcore.org/core/styles"
)

// FileViewDialog opens a dialog for selecting a file.
func FileViewDialog(ctx gi.Widget, filename, exts, title string, fun func(selfile string)) {
	d := gi.NewBody()
	if title != "" {
		d.SetTitle(title)
	}
	fv := NewFileView(d).SetFilename(filename, exts)
	d.AddAppBar(fv.ConfigToolbar)
	d.AddBottomBar(func(parent gi.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			fun(fv.SelectedFile())
		})
	})
	d.NewDialog(ctx).SetNewWindow(true).Run()
}

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
	DirPath string `set:"-"`

	// currently selected file
	CurrentSelectedFile string `set:"-"`

	// target extension(s) (comma separated if multiple, including initial .), if any
	Ext string `set:"-"`

	// optional styling function
	FilterFunc FileViewFilterFunc `view:"-" json:"-" xml:"-"`

	// map of lower-cased extensions from Ext -- used for highlighting files with one of these extensions -- maps onto original ext value
	ExtMap map[string]string

	// files for current directory
	Files []*fi.FileInfo

	// index of currently-selected file in Files list (-1 if none)
	SelectedIndex int `set:"-" edit:"-"`

	// change notify for current dir
	Watcher *fsnotify.Watcher `set:"-" view:"-"`

	// channel to close watcher watcher
	DoneWatcher chan bool `set:"-" view:"-"`

	// UpdateFiles mutex
	UpdateMu sync.Mutex `set:"-" view:"-"`

	// Previous path that was processed via UpdateFiles
	PrevPath string `set:"-" view:"-"`
}

func (fv *FileView) OnInit() {
	fv.Frame.OnInit()
	fv.HandleEvents()
	fv.SetStyles()
}

func (fv *FileView) SetStyles() {
	fv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	fv.OnWidgetAdded(func(w gi.Widget) {
		pfrom := w.PathFrom(fv)
		switch pfrom {
		case "path-tbar":
			fr := w.(*gi.Frame)
			gi.ToolbarStyles(fr)
			w.Style(func(s *styles.Style) {
				s.Gap.X.Dp(4)
			})
		case "path-tbar/path-lbl":
			w.Style(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
		case "path-tbar/path":
			w.Style(func(s *styles.Style) {
				s.Min.X.Ch(60)
				s.Max.X.Zero()
				s.Grow.Set(1, 0)
			})
		case "files-row":
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
		case "files-row/favs-view":
			fv := w.(*TableView)
			fv.SetFlag(false, SliceViewShowIndex)
			fv.SetFlag(false, SliceViewReadOnlyKeyNav) // can only have one active -- files..
			fv.SetReadOnly(true)
			w.Style(func(s *styles.Style) {
				s.Grow.Set(0, 1)
				s.Min.X.Ch(25)
				s.Overflow.X = styles.OverflowHidden
			})
		case "files-row/files-view":
			fv := w.(*TableView)
			fv.SetFlag(false, SliceViewShowIndex)
			fv.SetReadOnly(true)
			fv.Style(func(s *styles.Style) {
				// s.Grow.Set(1, 1)
			})
		case "sel-row":
			w.Style(func(s *styles.Style) {
				s.Gap.X.Dp(4)
			})
		case "sel-row/sel-lbl":
			w.Style(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
		case "sel-row/sel": // sel field
			w.Style(func(s *styles.Style) {
				s.Min.X.Ch(60)
				s.Max.X.Zero()
				s.Grow.Set(1, 0)
			})
		case "sel-row/ext-lbl":
			w.Style(func(s *styles.Style) {
				s.SetTextWrap(false)
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
type FileViewFilterFunc func(fv *FileView, fi *fi.FileInfo) bool

// FileViewDirOnlyFilter is a FileViewFilterFunc that only shows directories (folders).
func FileViewDirOnlyFilter(fv *FileView, fi *fi.FileInfo) bool {
	return fi.IsDir()
}

// FileViewExtOnlyFilter is a FileViewFilterFunc that only shows files that
// match the target extensions, and directories.
func FileViewExtOnlyFilter(fv *FileView, fi *fi.FileInfo) bool {
	if fi.IsDir() {
		return true
	}
	ext := strings.ToLower(filepath.Ext(fi.Name))
	_, has := fv.ExtMap[ext]
	return has
}

// SetFilename sets the initial filename (splitting out path and filename) and
// initializes the view
func (fv *FileView) SetFilename(filename, ext string) *FileView {
	fv.DirPath, fv.CurrentSelectedFile = filepath.Split(filename)
	return fv.SetExt(ext)
}

// SetPathFile sets the path, initial select file (or "") and initializes the view
func (fv *FileView) SetPathFile(path, file, ext string) *FileView {
	fv.DirPath = path
	fv.CurrentSelectedFile = file
	return fv.SetExt(ext)
}

// SelectedFile returns the full path to selected file
func (fv *FileView) SelectedFile() string {
	sf := fv.SelectField()
	sf.EditDone()
	return filepath.Join(fv.DirPath, fv.CurrentSelectedFile)
}

// SelectedFileInfo returns the currently-selected fileinfo, returns
// false if none
func (fv *FileView) SelectedFileInfo() (*fi.FileInfo, bool) {
	if fv.SelectedIndex < 0 || fv.SelectedIndex >= len(fv.Files) {
		return nil, false
	}
	return fv.Files[fv.SelectedIndex], true
}

// SelectFile selects the current file as the selection.
// if a directory it opens the directory and returns false.
// if a file it selects the file and returns true.
// if no selection, returns false.
func (fv *FileView) SelectFile() bool {
	if fi, ok := fv.SelectedFileInfo(); ok {
		if fi.IsDir() {
			fv.DirPath = filepath.Join(fv.DirPath, fi.Name)
			fv.CurrentSelectedFile = ""
			fv.SelectedIndex = -1
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

func (fv *FileView) Config() {
	fv.ConfigFileView()
}

func (fv *FileView) ConfigFileView() {
	if fv.HasChildren() {
		return
	}
	gi.NewLayout(fv, "files-row")
	gi.NewLayout(fv, "sel-row")

	fv.ConfigFilesRow()
	fv.ConfigSelRow()
	fv.UpdateFiles()
}

// ConfigToolbar configures the given toolbar to have file view
// actions and completions.
func (fv *FileView) ConfigToolbar(tb *gi.Toolbar) {
	NewFuncButton(tb, fv.DirPathUp).SetIcon(icons.ArrowUpward).SetKey(keyfun.Jump).SetText("Up")
	NewFuncButton(tb, fv.AddPathToFavs).SetIcon(icons.Favorite).SetText("Favorite")
	NewFuncButton(tb, fv.UpdateFilesAction).SetIcon(icons.Refresh).SetText("Update")
	NewFuncButton(tb, fv.NewFolder).SetIcon(icons.CreateNewFolder)

	ch := tb.AppChooser()
	if ch == nil {
		return
	}
	ch.ItemsFuncs = slices.Insert(ch.ItemsFuncs, 0, func() {
		for _, sp := range gi.RecentPaths {
			ch.Items = append(ch.Items, gi.ChooserItem{
				Value: sp,
				Icon:  icons.Folder,
				Func: func() {
					fv.DirPath = sp
					fv.UpdateFilesAction()
				},
			})
		}
		ch.Items = append(ch.Items, gi.ChooserItem{
			Value:           "Reset recent paths",
			Icon:            icons.Refresh,
			SeparatorBefore: true,
			Func: func() {
				gi.RecentPaths = make(gi.FilePaths, 1, gi.SystemSettings.SavedPathsMax)
				gi.RecentPaths[0] = fv.DirPath
				fv.UpdateFiles()
			},
		})
		ch.Items = append(ch.Items, gi.ChooserItem{
			Value: "Edit recent paths",
			Icon:  icons.Edit,
			Func: func() {
				fv.EditRecentPaths()
			},
		})
	})
}

func (fv *FileView) ConfigFilesRow() {
	fr := fv.FilesRow()
	sv := NewTableView(fr, "favs-view")
	fsv := NewTableView(fr, "files-view")

	sv.SelectedIndex = -1
	sv.SetReadOnly(true)
	sv.SetSlice(&gi.SystemSettings.FavPaths)
	sv.OnSelect(func(e events.Event) {
		fv.FavSelect(sv.SelectedIndex)
	})

	fsv.ContextMenus = nil
	fsv.AddContextMenu(func(m *gi.Scene) {
		gi.NewButton(m).SetText("Open").SetIcon(icons.Open).
			SetTooltip("Open the selected file using the default app").
			OnClick(func(e events.Event) {
				gi.TheApp.OpenURL(fv.SelectedFile())
			})
		gi.NewSeparator(m)
		gi.NewButton(m).SetText("Duplicate").SetIcon(icons.FileCopy).
			SetTooltip("Make a copy of the selected file").
			OnClick(func(e events.Event) {
				fn := fv.Files[fsv.SelectedIndex]
				fn.Duplicate()
				fv.UpdateFilesAction()
			})
		tip := "Delete moves the selected file to the trash / recycling bin"
		if gi.TheApp.Platform().IsMobile() {
			tip = "Delete deletes the selected file"
		}
		gi.NewButton(m).SetText("Delete").SetIcon(icons.Delete).
			SetTooltip(tip).
			OnClick(func(e events.Event) {
				fn := fv.Files[fsv.SelectedIndex]
				NewSoloFuncButton(fsv, fn.Delete).SetTooltip(tip).SetConfirm(true).
					SetAfterFunc(fv.UpdateFilesAction).CallFunc()
			})
		gi.NewButton(m).SetText("Rename").SetIcon(icons.EditNote).
			SetTooltip("Rename the selected file").
			OnClick(func(e events.Event) {
				fn := fv.Files[fsv.SelectedIndex]
				NewSoloFuncButton(fsv, fn.Rename).SetAfterFunc(fv.UpdateFilesAction).CallFunc()
			})
		gi.NewButton(m).SetText("Info").SetIcon(icons.Info).
			SetTooltip("View information about the selected file").
			OnClick(func(e events.Event) {
				fn := fv.Files[fsv.SelectedIndex]
				d := gi.NewBody().AddTitle("Info: " + fn.Name)
				NewStructView(d).SetStruct(&fn).SetReadOnly(true)
				d.AddOKOnly().NewFullDialog(fsv).Run()
			})
		gi.NewSeparator(m)
		NewFuncButton(m, fv.NewFolder).SetIcon(icons.CreateNewFolder)
	})
	fv.ReadFiles()
	fsv.SetReadOnly(true)
	fsv.SetSlice(&fv.Files)
	fsv.StyleFunc = func(w gi.Widget, s *styles.Style, row, col int) {
		if clr, got := FileViewKindColorMap[fv.Files[row].Kind]; got {
			s.Color = grr.Log1(gradient.FromString(clr))
			return
		}
		fn := fv.Files[row].Name
		ext := strings.ToLower(filepath.Ext(fn))
		if _, has := fv.ExtMap[ext]; has {
			s.Color = colors.C(colors.Scheme.Primary.Base)
		} else {
			s.Color = colors.C(colors.Scheme.OnSurface)
		}
	}

	if gi.SystemSettings.FileViewSort != "" {
		fsv.SetSortFieldName(gi.SystemSettings.FileViewSort)
	}
	fsv.Style(func(s *styles.Style) {
		s.Cursor = cursors.Pointer
	})
	fsv.OnSelect(func(e events.Event) {
		fv.FileSelectAction(fsv.SelectedIndex)
	})
	fsv.OnDoubleClick(func(e events.Event) {
		if fsv.ClickSelectEvent(e) {
			if !fv.SelectFile() {
				e.SetHandled() // don't pass along; keep dialog open
			} else {
				fv.Scene.SendKeyFun(keyfun.Accept, e) // activates Ok button code
			}
		}
	})
}

func (fv *FileView) ConfigSelRow() {
	sr := fv.SelectRow()
	gi.NewLabel(sr, "sel-lbl").SetText("File: ").
		SetTooltip("enter file name here (or select from above list)")

	sf := gi.NewTextField(sr, "sel").SetText(fv.CurrentSelectedFile).
		SetTooltip(fmt.Sprintf("Enter the file name. Special keys: up/down to move selection; %s or %s to go up to parent folder; %s or %s or %s or %s to select current file (if directory, goes into it, if file, selects and closes); %s or %s for prev / next history item; %s return to this field", keyfun.WordLeft.Label(), keyfun.Jump.Label(), keyfun.SelectMode.Label(), keyfun.Insert.Label(), keyfun.InsertAfter.Label(), keyfun.Open.Label(), keyfun.HistPrev.Label(), keyfun.HistNext.Label(), keyfun.Search.Label()))
	sf.SetCompleter(fv, fv.FileComplete, fv.FileCompleteEdit)
	sf.OnChange(func(e events.Event) {
		fv.SetSelFileAction(sf.Text())
	})
	sf.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Accept {
			fv.SetSelFileAction(sf.Text())
		}
	})
	sf.StartFocus()

	gi.NewLabel(sr, "ext-lbl").SetText("Extension(s):").
		SetTooltip("target extension(s) to highlight; if multiple, separate with commas, and do include the . at the start")
	ef := gi.NewTextField(sr, "ext").SetText(fv.Ext)
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

func (fv *FileView) SelectRow() *gi.Layout {
	return fv.ChildByName("sel-row", 4).(*gi.Layout)
}

// SelectField returns the TextField of the select file
func (fv *FileView) SelectField() *gi.TextField {
	return fv.SelectRow().ChildByName("sel", 1).(*gi.TextField)
}

// ExtField returns the TextField of the extension
func (fv *FileView) ExtField() *gi.TextField {
	return fv.SelectRow().ChildByName("ext", 2).(*gi.TextField)
}

// UpdatePath ensures that path is in abs form and ready to be used..
func (fv *FileView) UpdatePath() {
	if fv.DirPath == "" {
		fv.DirPath, _ = os.Getwd()
	}
	fv.DirPath, _ = homedir.Expand(fv.DirPath)
	fv.DirPath, _ = filepath.Abs(fv.DirPath)
}

// UpdateFilesAction updates the list of files and other views for the current path.
func (fv *FileView) UpdateFilesAction() { //gti:add
	fv.UpdateFiles()
	sf := fv.SelectField()
	sf.SetFocusEvent()
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

	fv.Files = make([]*fi.FileInfo, 0, 1000)
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
		fi, ferr := fi.NewFileInfo(path)
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
	fv.UpdateMu.Lock()
	defer fv.UpdateMu.Unlock()

	fv.UpdatePath()
	if len(gi.RecentPaths) == 0 {
		gi.OpenRecentPaths()
	}
	gi.RecentPaths.AddPath(fv.DirPath, gi.SystemSettings.SavedPathsMax)
	gi.SaveRecentPaths()
	sf := fv.SelectField()
	sf.SetText(fv.CurrentSelectedFile)

	fv.Scene.UpdateTitle("Files: " + fv.DirPath)

	fv.ReadFiles()

	fvv := fv.FavsView()
	fvv.ResetSelectedIndexs()

	sv := fv.FilesView()
	sv.ResetSelectedIndexs()
	sv.SelectedField = "Name"
	sv.SelectedValue = fv.CurrentSelectedFile
	sv.SortSlice()
	sv.Update()

	fv.SelectedIndex = sv.SelectedIndex
	if sv.SelectedIndex >= 0 {
		sv.ScrollToIndex(sv.SelectedIndex)
	}

	if fv.PrevPath != fv.DirPath {
		if gi.TheApp.Platform() != goosi.MacOS {
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
func (fv *FileView) AddPathToFavs() { //gti:add
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
	if _, found := gi.SystemSettings.FavPaths.FindPath(dp); found {
		gi.MessageSnackbar(fv, "Error: path is already on the favorites list")
		return
	}
	fi := gi.FavPathItem{"folder", fnm, dp}
	gi.SystemSettings.FavPaths = append(gi.SystemSettings.FavPaths, fi)
	gi.ErrorSnackbar(fv, gi.SaveSettings(gi.SystemSettings), "Error saving settings")
	// fv.FileSig.Emit(fv.This(), int64(FileViewFavAdded), fi)
	fv.UpdateFavs()
}

// DirPathUp moves up one directory in the path
func (fv *FileView) DirPathUp() { //gti:add
	pdr, _ := filepath.Split(fv.DirPath)
	if pdr == "" {
		return
	}
	fv.DirPath = pdr
	fv.UpdateFilesAction()
}

// NewFolder creates a new folder with the given name in the current directory.
func (fv *FileView) NewFolder(name string) error { //gti:add
	dp := fv.DirPath
	if dp == "" {
		return nil
	}
	np := filepath.Join(dp, name)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		return err
	}
	fv.UpdateFilesAction()
	return nil
}

// SetSelFileAction sets the currently selected file to given name, and sends
// selection action with current full file name, and updates selection in
// table view
func (fv *FileView) SetSelFileAction(sel string) {
	fv.CurrentSelectedFile = sel
	sv := fv.FilesView()
	ef := fv.ExtField()
	exts := ef.Text()
	if !sv.SelectFieldVal("Name", fv.CurrentSelectedFile) { // not found
		extl := strings.Split(exts, ",")
		if len(extl) == 1 {
			if !strings.HasSuffix(fv.CurrentSelectedFile, extl[0]) {
				fv.CurrentSelectedFile += extl[0]
			}
		}
	}
	fv.SelectedIndex = sv.SelectedIndex
	sf := fv.SelectField()
	sf.SetText(fv.CurrentSelectedFile) // make sure
	fv.Send(events.Select)             // receiver needs to get selectedFile
}

// FileSelectAction updates selection with given selected file and emits
// selected signal on WidgetSig with full name of selected item
func (fv *FileView) FileSelectAction(idx int) {
	if idx < 0 {
		return
	}
	fv.SaveSortPrefs()
	fi := fv.Files[idx]
	fv.SelectedIndex = idx
	fv.CurrentSelectedFile = fi.Name
	sf := fv.SelectField()
	sf.SetText(fv.CurrentSelectedFile)
	fv.Send(events.Select)
	// fv.WidgetSig.Emit(fv.This(), int64(gi.WidgetSelected), fv.SelectedFile())
}

// SetExt updates the ext to given (list of, comma separated) extensions
func (fv *FileView) SetExt(ext string) *FileView {
	if ext == "" {
		if fv.CurrentSelectedFile != "" {
			ext = strings.ToLower(filepath.Ext(fv.CurrentSelectedFile))
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
	return fv
}

// SetExtAction sets the current extension to highlight, and redisplays files
func (fv *FileView) SetExtAction(ext string) *FileView {
	fv.SetExt(ext)
	fv.UpdateFiles()
	return fv
}

// FavSelect selects a favorite path and goes there
func (fv *FileView) FavSelect(idx int) {
	if idx < 0 || idx >= len(gi.SystemSettings.FavPaths) {
		return
	}
	fi := gi.SystemSettings.FavPaths[idx]
	fv.DirPath, _ = homedir.Expand(fi.Path)
	fv.UpdateFilesAction()
}

// SaveSortPrefs saves current sorting preferences
func (fv *FileView) SaveSortPrefs() {
	sv := fv.FilesView()
	if sv == nil {
		return
	}
	gi.SystemSettings.FileViewSort = sv.SortFieldName()
	// fmt.Printf("sort: %v\n", gi.Prefs.FileViewSort)
	gi.ErrorSnackbar(fv, gi.SaveSettings(gi.SystemSettings), "Error saving settings")
}

func (fv *FileView) ApplyStyle() {
	fv.Frame.ApplyStyle()
	sf := fv.SelectField()
	sf.StartFocus() // need to call this when window is actually active
}

func (fv *FileView) HandleEvents() {
	fv.OnKeyChord(func(e events.Event) {
		fv.KeyInput(e)
	})
}

func (fv *FileView) KeyInput(kt events.Event) {
	kf := keyfun.Of(kt.KeyChord())
	if gi.DebugSettings.KeyEventTrace {
		slog.Info("FileView KeyInput", "widget", fv, "keyfun", kf)
	}
	switch kf {
	case keyfun.Jump, keyfun.WordLeft:
		kt.SetHandled()
		fv.DirPathUp()
	case keyfun.Insert, keyfun.InsertAfter, keyfun.Open, keyfun.SelectMode:
		kt.SetHandled()
		if fv.SelectFile() {
			fv.Send(events.DoubleClick, kt) // will close dialog
		}
	case keyfun.Search:
		kt.SetHandled()
		sf := fv.SelectField()
		sf.SetFocusEvent()
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

	files, err := os.ReadDir(dir)
	if err != nil {
		return md
	}
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

// EditRecentPaths displays a dialog allowing the user to
// edit the recent paths list.
func (fv *FileView) EditRecentPaths() {
	d := gi.NewBody().AddTitle("Recent file paths").AddText("You can delete paths you no longer use")
	NewSliceView(d).SetSlice(&gi.RecentPaths)
	d.AddBottomBar(func(parent gi.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			gi.SaveRecentPaths()
			fv.UpdateFiles()
		})
	})
	d.NewDialog(fv).Run()
}

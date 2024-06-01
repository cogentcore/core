// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
)

// FileViewDialog opens a dialog for selecting a file.
func FileViewDialog(ctx core.Widget, filename, exts, title string, fun func(selfile string)) {
	d := core.NewBody()
	if title != "" {
		d.SetTitle(title)
	}
	fv := NewFileView(d) // .SetFilename(filename, exts)
	d.AddAppBar(fv.MakeToolbar)
	d.AddAppChooser(fv.ConfigAppChooser)
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			fun(fv.SelectedFile())
		})
	})
	d.RunWindowDialog(ctx)
}

//////////////////////////////////////////////////////////////////////////
//  FileView

// todo:

// * search: use highlighting, not filtering -- < > arrows etc
// * also simple search-while typing in grid?
// * fileview selector DND is a file:/// url

// FileView is a viewer onto files -- core of the file chooser dialog
type FileView struct {
	core.Frame

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
	Files []*fileinfo.FileInfo

	// index of currently selected file in Files list (-1 if none)
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
	fv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	fv.OnKeyChord(func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		if core.DebugSettings.KeyEventTrace {
			slog.Info("FileView KeyInput", "widget", fv, "keyFunction", kf)
		}
		switch kf {
		case keymap.Jump, keymap.WordLeft:
			e.SetHandled()
			fv.DirPathUp()
		case keymap.Insert, keymap.InsertAfter, keymap.Open, keymap.SelectMode:
			e.SetHandled()
			if fv.SelectFile() {
				fv.Send(events.DoubleClick, e) // will close dialog
			}
		case keymap.Search:
			e.SetHandled()
			sf := fv.SelectField()
			sf.SetFocusEvent()
		}
	})

	fv.Maker(func(p *core.Plan) {
		if len(core.RecentPaths) == 0 {
			core.OpenRecentPaths()
		}
		fv.Scene.UpdateTitle("Files: " + fv.DirPath)
		core.RecentPaths.AddPath(fv.DirPath, core.SystemSettings.SavedPathsMax)
		core.SaveRecentPaths()
		fv.ReadFiles()

		if fv.PrevPath != fv.DirPath {
			if core.TheApp.Platform() != system.MacOS {
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

		core.AddAt(p, "files", func(w *core.Frame) {
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
			w.Maker(fv.makeFilesRow)
		})
		core.AddAt(p, "sel", func(w *core.Frame) {
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Gap.X.Dp(4)
			})
			w.Maker(fv.makeSelRow)
		})
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
type FileViewFilterFunc func(fv *FileView, fi *fileinfo.FileInfo) bool

// FileViewDirOnlyFilter is a FileViewFilterFunc that only shows directories (folders).
func FileViewDirOnlyFilter(fv *FileView, fi *fileinfo.FileInfo) bool {
	return fi.IsDir()
}

// FileViewExtOnlyFilter is a FileViewFilterFunc that only shows files that
// match the target extensions, and directories.
func FileViewExtOnlyFilter(fv *FileView, fi *fileinfo.FileInfo) bool {
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

// SelectedFileInfo returns the currently selected fileinfo, returns
// false if none
func (fv *FileView) SelectedFileInfo() (*fileinfo.FileInfo, bool) {
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

func (fv *FileView) MakeToolbar(p *core.Plan) {
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(fv.DirPathUp).SetIcon(icons.ArrowUpward).SetKey(keymap.Jump).SetText("Up")
	})
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(fv.AddPathToFavorites).SetIcon(icons.Favorite).SetText("Favorite")
	})
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(fv.UpdateFilesAction).SetIcon(icons.Refresh).SetText("Update")
	})
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(fv.NewFolder).SetIcon(icons.CreateNewFolder)
	})
}

// ConfigAppChooser configures given app chooser
func (fv *FileView) ConfigAppChooser(ch *core.Chooser) {
	ch.ItemsFuncs = slices.Insert(ch.ItemsFuncs, 0, func() {
		for _, sp := range core.RecentPaths {
			ch.Items = append(ch.Items, core.ChooserItem{
				Value: sp,
				Icon:  icons.Folder,
				Func: func() {
					fv.DirPath = sp
					fv.UpdateFilesAction()
				},
			})
		}
		ch.Items = append(ch.Items, core.ChooserItem{
			Value:           "Reset recent paths",
			Icon:            icons.Refresh,
			SeparatorBefore: true,
			Func: func() {
				core.RecentPaths = make(core.FilePaths, 1, core.SystemSettings.SavedPathsMax)
				core.RecentPaths[0] = fv.DirPath
				fv.Update()
			},
		})
		ch.Items = append(ch.Items, core.ChooserItem{
			Value: "Edit recent paths",
			Icon:  icons.Edit,
			Func: func() {
				fv.EditRecentPaths()
			},
		})
	})
}

func (fv *FileView) makeFilesRow(p *core.Plan) {
	core.AddAt(p, "favorites", func(w *TableView) {
		w.SelectedIndex = -1
		w.SetReadOnly(true)
		w.SetFlag(false, SliceViewShowIndex)
		w.SetFlag(false, SliceViewReadOnlyKeyNav) // can only have one active -- files..
		w.Style(func(s *styles.Style) {
			s.Grow.Set(0, 1)
			s.Min.X.Ch(25)
			s.Overflow.X = styles.OverflowHidden
		})
		w.SetSlice(&core.SystemSettings.FavPaths)
		w.OnSelect(func(e events.Event) {
			fv.FavoritesSelect(w.SelectedIndex)
		})
		w.Updater(func() {
			w.ResetSelectedIndexes()
		})
	})
	core.AddAt(p, "files", func(w *TableView) {
		w.SetFlag(false, SliceViewShowIndex)
		w.SetReadOnly(true)
		w.SetSlice(&fv.Files)
		w.SelectedField = "Name"
		w.SelectedValue = fv.CurrentSelectedFile
		if core.SystemSettings.FileViewSort != "" {
			w.SetSortFieldName(core.SystemSettings.FileViewSort)
		}
		w.StyleFunc = func(w core.Widget, s *styles.Style, row, col int) {
			if clr, got := FileViewKindColorMap[fv.Files[row].Kind]; got {
				s.Color = errors.Log1(gradient.FromString(clr))
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
		w.Style(func(s *styles.Style) {
			s.Cursor = cursors.Pointer
		})
		w.OnSelect(func(e events.Event) {
			fv.FileSelectAction(w.SelectedIndex)
		})
		w.OnDoubleClick(func(e events.Event) {
			if w.ClickSelectEvent(e) {
				if !fv.SelectFile() {
					e.SetHandled() // don't pass along; keep dialog open
				} else {
					fv.Scene.SendKey(keymap.Accept, e) // activates Ok button code
				}
			}
		})
		w.ContextMenus = nil
		w.AddContextMenu(func(m *core.Scene) {
			core.NewButton(m).SetText("Open").SetIcon(icons.Open).
				SetTooltip("Open the selected file using the default app").
				OnClick(func(e events.Event) {
					core.TheApp.OpenURL(fv.SelectedFile())
				})
			core.NewSeparator(m)
			core.NewButton(m).SetText("Duplicate").SetIcon(icons.FileCopy).
				SetTooltip("Make a copy of the selected file").
				OnClick(func(e events.Event) {
					fn := fv.Files[w.SelectedIndex]
					fn.Duplicate()
					fv.UpdateFilesAction()
				})
			tip := "Delete moves the selected file to the trash / recycling bin"
			if core.TheApp.Platform().IsMobile() {
				tip = "Delete deletes the selected file"
			}
			core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).
				SetTooltip(tip).
				OnClick(func(e events.Event) {
					fn := fv.Files[w.SelectedIndex]
					NewSoloFuncButton(w, fn.Delete).SetTooltip(tip).SetConfirm(true).
						SetAfterFunc(fv.UpdateFilesAction).CallFunc()
				})
			core.NewButton(m).SetText("Rename").SetIcon(icons.EditNote).
				SetTooltip("Rename the selected file").
				OnClick(func(e events.Event) {
					fn := fv.Files[w.SelectedIndex]
					NewSoloFuncButton(w, fn.Rename).SetAfterFunc(fv.UpdateFilesAction).CallFunc()
				})
			core.NewButton(m).SetText("Info").SetIcon(icons.Info).
				SetTooltip("View information about the selected file").
				OnClick(func(e events.Event) {
					fn := fv.Files[w.SelectedIndex]
					d := core.NewBody().AddTitle("Info: " + fn.Name)
					NewStructView(d).SetStruct(&fn).SetReadOnly(true)
					d.AddOKOnly().RunFullDialog(w)
				})
			core.NewSeparator(m)
			NewFuncButton(m, fv.NewFolder).SetIcon(icons.CreateNewFolder)
		})
		// w.Updater(func() {})
	})
}

func (fv *FileView) makeSelRow(sel *core.Plan) {
	core.AddAt(sel, "file-text", func(w *core.Text) {
		w.SetText("File: ")
		w.SetTooltip("Enter file name here (or select from list above)")
		w.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
	})

	core.AddAt(sel, "file", func(w *core.TextField) {
		w.SetText(fv.CurrentSelectedFile)
		w.SetTooltip(fmt.Sprintf("Enter the file name. Special keys: up/down to move selection; %s or %s to go up to parent folder; %s or %s or %s or %s to select current file (if directory, goes into it, if file, selects and closes); %s or %s for prev / next history item; %s return to this field", keymap.WordLeft.Label(), keymap.Jump.Label(), keymap.SelectMode.Label(), keymap.Insert.Label(), keymap.InsertAfter.Label(), keymap.Open.Label(), keymap.HistPrev.Label(), keymap.HistNext.Label(), keymap.Search.Label()))
		w.SetCompleter(fv, fv.FileComplete, fv.FileCompleteEdit)
		w.Style(func(s *styles.Style) {
			s.Min.X.Ch(60)
			s.Max.X.Zero()
			s.Grow.Set(1, 0)
		})
		w.OnChange(func(e events.Event) {
			fv.SetSelFileAction(w.Text())
		})
		w.OnKeyChord(func(e events.Event) {
			kf := keymap.Of(e.KeyChord())
			if kf == keymap.Accept {
				fv.SetSelFileAction(w.Text())
			}
		})
		w.StartFocus()
		w.Updater(func() {
			w.SetText(fv.CurrentSelectedFile)
		})
	})

	core.AddAt(sel, "ext-text", func(w *core.Text) {
		w.SetText("Extension(s):").SetTooltip("target extension(s) to highlight; if multiple, separate with commas, and include the . at the start")
		w.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
	})

	core.AddAt(sel, "ext", func(w *core.TextField) {
		w.SetText(fv.Ext)
		w.OnChange(func(e events.Event) {
			fv.SetExtAction(w.Text())
		})
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
					fv.Update()
				}
			case err := <-watch.Errors:
				_ = err
			}
		}
	}()
}

// FavoritesView returns the TableView of the favorites
func (fv *FileView) FavoritesView() *TableView {
	return fv.FindPath("files/favorites").(*TableView)
}

// FilesView returns the TableView of the files
func (fv *FileView) FilesView() *TableView {
	return fv.FindPath("files/files").(*TableView)
}

// SelectField returns the TextField of the select file
func (fv *FileView) SelectField() *core.TextField {
	return fv.FindPath("sel/file").(*core.TextField)
}

// ExtField returns the TextField of the extension
func (fv *FileView) ExtField() *core.TextField {
	return fv.FindPath("sel/ext").(*core.TextField)
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
func (fv *FileView) UpdateFilesAction() { //types:add
	fv.ReadFiles()
	fv.Update()
	// sf := fv.SelectField()
	// sf.SetFocusEvent()
}

func (fv *FileView) ReadFiles() {
	effpath, err := filepath.EvalSymlinks(fv.DirPath)
	if err != nil {
		log.Printf("core.FileView Path: %v could not be opened -- error: %v\n", effpath, err)
		return
	}
	_, err = os.Lstat(effpath)
	if err != nil {
		log.Printf("core.FileView Path: %v could not be opened -- error: %v\n", effpath, err)
		return
	}

	fv.Files = make([]*fileinfo.FileInfo, 0, 1000)
	filepath.Walk(effpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("Path %q: Error: %v", effpath, err)
			// if fv.Scene != nil {
			// 	core.PromptDialog(fv, core.DlgOpts{Title: "FileView UpdateFiles", emsg, Ok: true, Cancel: false}, nil)
			// } else {
			log.Printf("core.FileView error: %v\n", emsg)
			// }
			return nil // ignore
		}
		if path == effpath { // proceed..
			return nil
		}
		fi, ferr := fileinfo.NewFileInfo(path)
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

// UpdateFavorites updates list of files and other views for current path
func (fv *FileView) UpdateFavorites() {
	sv := fv.FavoritesView()
	sv.Update()
}

// AddPathToFavorites adds the current path to favorites
func (fv *FileView) AddPathToFavorites() { //types:add
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
	if _, found := core.SystemSettings.FavPaths.FindPath(dp); found {
		core.MessageSnackbar(fv, "Error: path is already on the favorites list")
		return
	}
	fi := core.FavPathItem{"folder", fnm, dp}
	core.SystemSettings.FavPaths = append(core.SystemSettings.FavPaths, fi)
	core.ErrorSnackbar(fv, core.SaveSettings(core.SystemSettings), "Error saving settings")
	// fv.FileSig.Emit(fv.This(), int64(FileViewFavAdded), fi)
	fv.UpdateFavorites()
}

// DirPathUp moves up one directory in the path
func (fv *FileView) DirPathUp() { //types:add
	pdr := filepath.Dir(fv.DirPath)
	if pdr == "" {
		return
	}
	fv.DirPath = pdr
	fv.UpdateFilesAction()
}

// NewFolder creates a new folder with the given name in the current directory.
func (fv *FileView) NewFolder(name string) error { //types:add
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
	fv.SaveSortSettings()
	fi := fv.Files[idx]
	fv.SelectedIndex = idx
	fv.CurrentSelectedFile = fi.Name
	sf := fv.SelectField()
	sf.SetText(fv.CurrentSelectedFile)
	fv.Send(events.Select)
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
	fv.Update()
	return fv
}

// FavoritesSelect selects a favorite path and goes there
func (fv *FileView) FavoritesSelect(idx int) {
	if idx < 0 || idx >= len(core.SystemSettings.FavPaths) {
		return
	}
	fi := core.SystemSettings.FavPaths[idx]
	fv.DirPath, _ = homedir.Expand(fi.Path)
	fv.UpdateFilesAction()
}

// SaveSortSettings saves current sorting preferences
func (fv *FileView) SaveSortSettings() {
	sv := fv.FilesView()
	if sv == nil {
		return
	}
	core.SystemSettings.FileViewSort = sv.SortFieldName()
	// fmt.Printf("sort: %v\n", core.Settings.FileViewSort)
	core.ErrorSnackbar(fv, core.SaveSettings(core.SystemSettings), "Error saving settings")
}

// FileComplete finds the possible completions for the file field
func (fv *FileView) FileComplete(data any, text string, posLine, posChar int) (md complete.Matches) {
	md.Seed = complete.SeedPath(text)

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
func (fv *FileView) PathComplete(data any, path string, posLine, posChar int) (md complete.Matches) {
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
	d := core.NewBody().AddTitle("Recent file paths").AddText("You can delete paths you no longer use")
	NewSliceView(d).SetSlice(&core.RecentPaths)
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			core.SaveRecentPaths()
			fv.Update()
		})
	})
	d.RunDialog(fv)
}

// FileButton represents a filename value with a button
// that opens a [FileView].
type FileButton struct {
	core.Button
	Filename string
}

func (fb *FileButton) WidgetValue() any { return &fb.Filename }

func (fb *FileButton) OnInit() {
	fb.Button.OnInit()
	fb.SetType(core.ButtonTonal).SetIcon(icons.File)
	fb.Updater(func() {
		if fb.Filename == "" {
			fb.SetText("Select file")
		} else {
			fb.SetText(fb.Filename)
		}
	})
	var fv *FileView
	core.InitValueButton(fb, false, func(d *core.Body) {
		fb.SetFlag(true, core.ValueDialogNewWindow) // default to new window on supported platforms
		// ext, _ := v.Tag("ext") // TODO(config)
		fv = NewFileView(d).SetFilename(fb.Filename, "")
		d.AddAppBar(fv.MakeToolbar)
	}, func() {
		fb.Filename = fv.SelectedFile()
	})
}

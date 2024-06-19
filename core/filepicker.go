// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// FilePickerDialog opens a dialog for selecting a file.
func FilePickerDialog(ctx Widget, filename, exts, title string, fun func(selfile string)) {
	d := NewBody()
	if title != "" {
		d.SetTitle(title)
	}
	fv := NewFilePicker(d).SetFilename(filename).SetExtensions(exts)
	d.AddAppBar(fv.MakeToolbar)
	d.AddBottomBar(func(parent Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			fun(fv.SelectedFile())
		})
	})
	d.RunWindowDialog(ctx)
}

// todo:

// * search: use highlighting, not filtering -- < > arrows etc
// * also simple search-while typing in grid?
// * filepicker selector DND is a file:/// url

// FilePicker is a widget for selecting files.
type FilePicker struct {
	Frame

	// Directory is the absolute path to the directory of files to display.
	Directory string `set:"-"`

	// SelectedFilename is the name of the currently selected file, not including the directory.
	// See [FilePicker.SelectedFile] for the full path.
	SelectedFilename string `set:"-"`

	// Extensions is a list of the target file extensions.
	// If there are multiple, they must be comma separated.
	// The extensions must include the dot (".") at the start.
	// They must be set using [FilePicker.SetExtensions].
	Extensions string `set:"-"`

	// FilterFunc is an optional filtering function for which files to display.
	FilterFunc FilePickerFilterFunc `display:"-" json:"-" xml:"-"`

	// extensionMap is a map of lower-cased extensions from Extensions.
	// It used for highlighting files with one of these extensions;
	// maps onto original Extensions value.
	extensionMap map[string]string

	// files for current directory
	files []*fileinfo.FileInfo

	// index of currently selected file in Files list (-1 if none)
	selectedIndex int

	// change notify for current dir
	watcher *fsnotify.Watcher

	// channel to close watcher watcher
	doneWatcher chan bool

	// Previous path that was processed via UpdateFiles
	prevPath string
}

func (fp *FilePicker) Init() {
	fp.Frame.Init()
	fp.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	fp.OnKeyChord(func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		if DebugSettings.KeyEventTrace {
			slog.Info("FilePicker KeyInput", "widget", fp, "keyFunction", kf)
		}
		switch kf {
		case keymap.Jump, keymap.WordLeft:
			e.SetHandled()
			fp.DirectoryUp()
		case keymap.Insert, keymap.InsertAfter, keymap.Open, keymap.SelectMode:
			e.SetHandled()
			if fp.SelectFile() {
				fp.Send(events.DoubleClick, e) // will close dialog
			}
		case keymap.Search:
			e.SetHandled()
			sf := fp.SelectField()
			sf.SetFocusEvent()
		}
	})

	fp.Maker(func(p *tree.Plan) {
		if fp.Directory == "" {
			fp.SetFilename("") // default to current directory
		}
		if len(RecentPaths) == 0 {
			OpenRecentPaths()
		}
		// if we update the title before the scene is shown, it may incorrectly
		// override the title of the window of the context widget
		if fp.Scene.HasShown {
			fp.Scene.UpdateTitle("Files: " + fp.Directory)
		}
		RecentPaths.AddPath(fp.Directory, SystemSettings.SavedPathsMax)
		SaveRecentPaths()
		fp.ReadFiles()

		if fp.prevPath != fp.Directory {
			if TheApp.Platform() != system.MacOS {
				// mac is not supported in a high-capacity fashion at this point
				if fp.prevPath == "" {
					fp.ConfigWatcher()
				} else {
					fp.watcher.Remove(fp.prevPath)
				}
				fp.watcher.Add(fp.Directory)
				if fp.prevPath == "" {
					fp.WatchWatcher()
				}
			}
			fp.prevPath = fp.Directory
		}

		tree.AddAt(p, "files", func(w *Frame) {
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
			w.Maker(fp.makeFilesRow)
		})
		tree.AddAt(p, "sel", func(w *Frame) {
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Gap.X.Dp(4)
			})
			w.Maker(fp.makeSelRow)
		})
	})
}

func (fp *FilePicker) Disconnect() {
	if fp.watcher != nil {
		fp.watcher.Close()
		fp.watcher = nil
	}
	if fp.doneWatcher != nil {
		fp.doneWatcher <- true
		close(fp.doneWatcher)
		fp.doneWatcher = nil
	}
}

// FilePickerFilterFunc is a filtering function for files; returns true if the
// file should be visible in the picker, and false if not
type FilePickerFilterFunc func(fp *FilePicker, fi *fileinfo.FileInfo) bool

// FilePickerDirOnlyFilter is a FilePickerFilterFunc that only shows directories (folders).
func FilePickerDirOnlyFilter(fp *FilePicker, fi *fileinfo.FileInfo) bool {
	return fi.IsDir()
}

// FilePickerExtOnlyFilter is a FilePickerFilterFunc that only shows files that
// match the target extensions, and directories.
func FilePickerExtOnlyFilter(fp *FilePicker, fi *fileinfo.FileInfo) bool {
	if fi.IsDir() {
		return true
	}
	ext := strings.ToLower(filepath.Ext(fi.Name))
	_, has := fp.extensionMap[ext]
	return has
}

// SetFilename sets the directory and filename of the file picker
// from the given filepath.
func (fp *FilePicker) SetFilename(filename string) *FilePicker {
	fp.Directory, fp.SelectedFilename = filepath.Split(filename)
	fp.Directory = errors.Log1(filepath.Abs(fp.Directory))
	return fp
}

// SelectedFile returns the full path to the currently selected file.
func (fp *FilePicker) SelectedFile() string {
	sf := fp.SelectField()
	sf.EditDone()
	return filepath.Join(fp.Directory, fp.SelectedFilename)
}

// SelectedFileInfo returns the currently selected fileinfo, returns
// false if none
func (fp *FilePicker) SelectedFileInfo() (*fileinfo.FileInfo, bool) {
	if fp.selectedIndex < 0 || fp.selectedIndex >= len(fp.files) {
		return nil, false
	}
	return fp.files[fp.selectedIndex], true
}

// SelectFile selects the current file as the selection.
// if a directory it opens the directory and returns false.
// if a file it selects the file and returns true.
// if no selection, returns false.
func (fp *FilePicker) SelectFile() bool {
	if fi, ok := fp.SelectedFileInfo(); ok {
		if fi.IsDir() {
			fp.Directory = filepath.Join(fp.Directory, fi.Name)
			fp.SelectedFilename = ""
			fp.selectedIndex = -1
			fp.UpdateFilesAction()
			return false
		}
		return true
	}
	return false
}

// STYTODO: get rid of this or make it use actual color values
// FilePickerKindColorMap translates file Kinds into different colors for the file picker
var FilePickerKindColorMap = map[string]string{
	"folder": "pref(link)",
}

func (fp *FilePicker) MakeToolbar(p *tree.Plan) {
	tree.AddInit(p, "app-chooser", func(w *Chooser) {
		fp.AddChooserPaths(w)
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(fp.DirectoryUp).SetIcon(icons.ArrowUpward).SetKey(keymap.Jump).SetText("Up")
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(fp.AddPathToFavorites).SetIcon(icons.Favorite).SetText("Favorite")
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(fp.UpdateFilesAction).SetIcon(icons.Refresh).SetText("Update")
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(fp.NewFolder).SetIcon(icons.CreateNewFolder)
	})
}

// AddChooserPaths adds paths to the app chooser
func (fp *FilePicker) AddChooserPaths(ch *Chooser) {
	ch.ItemsFuncs = slices.Insert(ch.ItemsFuncs, 0, func() {
		for _, sp := range RecentPaths {
			ch.Items = append(ch.Items, ChooserItem{
				Value: sp,
				Icon:  icons.Folder,
				Func: func() {
					fp.Directory = sp
					fp.UpdateFilesAction()
				},
			})
		}
		ch.Items = append(ch.Items, ChooserItem{
			Value:           "Reset recent paths",
			Icon:            icons.Refresh,
			SeparatorBefore: true,
			Func: func() {
				RecentPaths = make(FilePaths, 1, SystemSettings.SavedPathsMax)
				RecentPaths[0] = fp.Directory
				fp.Update()
			},
		})
		ch.Items = append(ch.Items, ChooserItem{
			Value: "Edit recent paths",
			Icon:  icons.Edit,
			Func: func() {
				fp.EditRecentPaths()
			},
		})
	})
}

func (fp *FilePicker) makeFilesRow(p *tree.Plan) {
	tree.AddAt(p, "favorites", func(w *Table) {
		w.SelectedIndex = -1
		w.SetReadOnly(true)
		w.ReadOnlyKeyNav = false // keys must go to files, not favorites
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(0, 1)
			s.Min.X.Ch(25)
			s.Overflow.X = styles.OverflowHidden
		})
		w.SetSlice(&SystemSettings.FavPaths)
		w.OnSelect(func(e events.Event) {
			fp.FavoritesSelect(w.SelectedIndex)
		})
		w.Updater(func() {
			w.ResetSelectedIndexes()
		})
	})
	tree.AddAt(p, "files", func(w *Table) {
		w.SetReadOnly(true)
		w.SetSlice(&fp.files)
		w.SelectedField = "Name"
		w.SelectedValue = fp.SelectedFilename
		if SystemSettings.FilePickerSort != "" {
			w.SetSortFieldName(SystemSettings.FilePickerSort)
		}
		w.StyleFunc = func(w Widget, s *styles.Style, row, col int) {
			if clr, got := FilePickerKindColorMap[fp.files[row].Kind]; got {
				s.Color = errors.Log1(gradient.FromString(clr))
				return
			}
			fn := fp.files[row].Name
			ext := strings.ToLower(filepath.Ext(fn))
			if _, has := fp.extensionMap[ext]; has {
				s.Color = colors.C(colors.Scheme.Primary.Base)
			} else {
				s.Color = colors.C(colors.Scheme.OnSurface)
			}
		}
		w.Styler(func(s *styles.Style) {
			s.Cursor = cursors.Pointer
		})
		w.OnSelect(func(e events.Event) {
			fp.FileSelect(w.SelectedIndex)
		})
		w.OnDoubleClick(func(e events.Event) {
			if w.ClickSelectEvent(e) {
				if !fp.SelectFile() {
					e.SetHandled() // don't pass along; keep dialog open
				} else {
					fp.Scene.SendKey(keymap.Accept, e) // activates Ok button code
				}
			}
		})
		w.ContextMenus = nil
		w.AddContextMenu(func(m *Scene) {
			NewButton(m).SetText("Open").SetIcon(icons.Open).
				SetTooltip("Open the selected file using the default app").
				OnClick(func(e events.Event) {
					TheApp.OpenURL(fp.SelectedFile())
				})
			NewSeparator(m)
			NewButton(m).SetText("Duplicate").SetIcon(icons.FileCopy).
				SetTooltip("Make a copy of the selected file").
				OnClick(func(e events.Event) {
					fn := fp.files[w.SelectedIndex]
					fn.Duplicate()
					fp.UpdateFilesAction()
				})
			tip := "Delete moves the selected file to the trash / recycling bin"
			if TheApp.Platform().IsMobile() {
				tip = "Delete deletes the selected file"
			}
			NewButton(m).SetText("Delete").SetIcon(icons.Delete).
				SetTooltip(tip).
				OnClick(func(e events.Event) {
					fn := fp.files[w.SelectedIndex]
					fb := NewSoloFuncButton(w).SetFunc(fn.Delete).SetConfirm(true).SetAfterFunc(fp.UpdateFilesAction)
					fb.SetTooltip(tip)
					fb.CallFunc()
				})
			NewButton(m).SetText("Rename").SetIcon(icons.EditNote).
				SetTooltip("Rename the selected file").
				OnClick(func(e events.Event) {
					fn := fp.files[w.SelectedIndex]
					NewSoloFuncButton(w).SetFunc(fn.Rename).SetAfterFunc(fp.UpdateFilesAction).CallFunc()
				})
			NewButton(m).SetText("Info").SetIcon(icons.Info).
				SetTooltip("View information about the selected file").
				OnClick(func(e events.Event) {
					fn := fp.files[w.SelectedIndex]
					d := NewBody().AddTitle("Info: " + fn.Name)
					NewForm(d).SetStruct(&fn).SetReadOnly(true)
					d.AddOKOnly().RunFullDialog(w)
				})
			NewSeparator(m)
			NewFuncButton(m).SetFunc(fp.NewFolder).SetIcon(icons.CreateNewFolder)
		})
		// w.Updater(func() {})
	})
}

func (fp *FilePicker) makeSelRow(sel *tree.Plan) {
	tree.AddAt(sel, "file-text", func(w *Text) {
		w.SetText("File: ")
		w.SetTooltip("Enter file name here (or select from list above)")
		w.Styler(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
	})

	tree.AddAt(sel, "file", func(w *TextField) {
		w.SetText(fp.SelectedFilename)
		w.SetTooltip(fmt.Sprintf("Enter the file name. Special keys: up/down to move selection; %s or %s to go up to parent folder; %s or %s or %s or %s to select current file (if directory, goes into it, if file, selects and closes); %s or %s for prev / next history item; %s return to this field", keymap.WordLeft.Label(), keymap.Jump.Label(), keymap.SelectMode.Label(), keymap.Insert.Label(), keymap.InsertAfter.Label(), keymap.Open.Label(), keymap.HistPrev.Label(), keymap.HistNext.Label(), keymap.Search.Label()))
		w.SetCompleter(fp, fp.FileComplete, fp.FileCompleteEdit)
		w.Styler(func(s *styles.Style) {
			s.Min.X.Ch(60)
			s.Max.X.Zero()
			s.Grow.Set(1, 0)
		})
		w.OnChange(func(e events.Event) {
			fp.SetSelectedFile(w.Text())
		})
		w.OnKeyChord(func(e events.Event) {
			kf := keymap.Of(e.KeyChord())
			if kf == keymap.Accept {
				fp.SetSelectedFile(w.Text())
			}
		})
		w.StartFocus()
		w.Updater(func() {
			w.SetText(fp.SelectedFilename)
		})
	})

	tree.AddAt(sel, "extension-text", func(w *Text) {
		w.SetText("Extension(s):").SetTooltip("target extension(s) to highlight; if multiple, separate with commas, and include the . at the start")
		w.Styler(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
	})

	tree.AddAt(sel, "extension", func(w *TextField) {
		w.SetText(fp.Extensions)
		w.OnChange(func(e events.Event) {
			fp.SetExtensions(w.Text()).Update()
		})
	})
}

func (fp *FilePicker) ConfigWatcher() error {
	if fp.watcher != nil {
		return nil
	}
	var err error
	fp.watcher, err = fsnotify.NewWatcher()
	return err
}

func (fp *FilePicker) WatchWatcher() {
	if fp.watcher == nil || fp.watcher.Events == nil {
		return
	}
	if fp.doneWatcher != nil {
		return
	}
	fp.doneWatcher = make(chan bool)
	go func() {
		watch := fp.watcher
		done := fp.doneWatcher
		for {
			select {
			case <-done:
				return
			case event := <-watch.Events:
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Rename == fsnotify.Rename:
					fp.Update()
				}
			case err := <-watch.Errors:
				_ = err
			}
		}
	}()
}

// FavoritesView returns the Table of the favorites
func (fp *FilePicker) FavoritesView() *Table {
	return fp.FindPath("files/favorites").(*Table)
}

// FilesView returns the Table of the files
func (fp *FilePicker) FilesView() *Table {
	return fp.FindPath("files/files").(*Table)
}

// SelectField returns the TextField of the select file
func (fp *FilePicker) SelectField() *TextField {
	return fp.FindPath("sel/file").(*TextField)
}

// ExtField returns the TextField of the extension
func (fp *FilePicker) ExtField() *TextField {
	return fp.FindPath("sel/extension").(*TextField)
}

// UpdatePath ensures that path is in abs form and ready to be used..
func (fp *FilePicker) UpdatePath() {
	if fp.Directory == "" {
		fp.Directory, _ = os.Getwd()
	}
	fp.Directory, _ = homedir.Expand(fp.Directory)
	fp.Directory, _ = filepath.Abs(fp.Directory)
}

// UpdateFilesAction updates the list of files and other views for the current path.
func (fp *FilePicker) UpdateFilesAction() { //types:add
	fp.ReadFiles()
	fp.Update()
	// sf := fv.SelectField()
	// sf.SetFocusEvent()
}

func (fp *FilePicker) ReadFiles() {
	effpath, err := filepath.EvalSymlinks(fp.Directory)
	if err != nil {
		log.Printf("FilePicker Path: %v could not be opened -- error: %v\n", effpath, err)
		return
	}
	_, err = os.Lstat(effpath)
	if err != nil {
		log.Printf("FilePicker Path: %v could not be opened -- error: %v\n", effpath, err)
		return
	}

	fp.files = make([]*fileinfo.FileInfo, 0, 1000)
	filepath.Walk(effpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("Path %q: Error: %v", effpath, err)
			// if fv.Scene != nil {
			// 	PromptDialog(fv, DlgOpts{Title: "FilePicker UpdateFiles", emsg, Ok: true, Cancel: false}, nil)
			// } else {
			log.Printf("FilePicker error: %v\n", emsg)
			// }
			return nil // ignore
		}
		if path == effpath { // proceed..
			return nil
		}
		fi, ferr := fileinfo.NewFileInfo(path)
		keep := ferr == nil
		if fp.FilterFunc != nil {
			keep = fp.FilterFunc(fp, fi)
		}
		if keep {
			fp.files = append(fp.files, fi)
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
}

// UpdateFavorites updates list of files and other views for current path
func (fp *FilePicker) UpdateFavorites() {
	sv := fp.FavoritesView()
	sv.Update()
}

// AddPathToFavorites adds the current path to favorites
func (fp *FilePicker) AddPathToFavorites() { //types:add
	dp := fp.Directory
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
	if _, found := SystemSettings.FavPaths.FindPath(dp); found {
		MessageSnackbar(fp, "Error: path is already on the favorites list")
		return
	}
	fi := FavPathItem{"folder", fnm, dp}
	SystemSettings.FavPaths = append(SystemSettings.FavPaths, fi)
	ErrorSnackbar(fp, SaveSettings(SystemSettings), "Error saving settings")
	// fv.FileSig.Emit(fv.This, int64(FilePickerFavAdded), fi)
	fp.UpdateFavorites()
}

// DirectoryUp moves up one directory in the path
func (fp *FilePicker) DirectoryUp() { //types:add
	pdr := filepath.Dir(fp.Directory)
	if pdr == "" {
		return
	}
	fp.Directory = pdr
	fp.UpdateFilesAction()
}

// NewFolder creates a new folder with the given name in the current directory.
func (fp *FilePicker) NewFolder(name string) error { //types:add
	dp := fp.Directory
	if dp == "" {
		return nil
	}
	np := filepath.Join(dp, name)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		return err
	}
	fp.UpdateFilesAction()
	return nil
}

// SetSelectedFile sets the currently selected file to the given name, sends
// a selection event, and updates the selection in the table.
func (fp *FilePicker) SetSelectedFile(file string) {
	fp.SelectedFilename = file
	sv := fp.FilesView()
	ef := fp.ExtField()
	exts := ef.Text()
	if !sv.SelectFieldVal("Name", fp.SelectedFilename) { // not found
		extl := strings.Split(exts, ",")
		if len(extl) == 1 {
			if !strings.HasSuffix(fp.SelectedFilename, extl[0]) {
				fp.SelectedFilename += extl[0]
			}
		}
	}
	fp.selectedIndex = sv.SelectedIndex
	sf := fp.SelectField()
	sf.SetText(fp.SelectedFilename) // make sure
	fp.Send(events.Select)          // receiver needs to get selectedFile
}

// FileSelect updates the selection with the given selected file index and
// sends a select event.
func (fp *FilePicker) FileSelect(idx int) {
	if idx < 0 {
		return
	}
	fp.SaveSortSettings()
	fi := fp.files[idx]
	fp.selectedIndex = idx
	fp.SelectedFilename = fi.Name
	sf := fp.SelectField()
	sf.SetText(fp.SelectedFilename)
	fp.Send(events.Select)
}

// SetExtensions sets the [FilePicker.Extensions] to the given comma separated
// list of file extensions, which each must start with a dot (".").
func (fp *FilePicker) SetExtensions(ext string) *FilePicker {
	if ext == "" {
		if fp.SelectedFilename != "" {
			ext = strings.ToLower(filepath.Ext(fp.SelectedFilename))
		}
	}
	fp.Extensions = ext
	exts := strings.Split(fp.Extensions, ",")
	fp.extensionMap = make(map[string]string, len(exts))
	for _, ex := range exts {
		ex = strings.TrimSpace(ex)
		if len(ex) == 0 {
			continue
		}
		if ex[0] != '.' {
			ex = "." + ex
		}
		fp.extensionMap[strings.ToLower(ex)] = ex
	}
	return fp
}

// FavoritesSelect selects a favorite path and goes there
func (fp *FilePicker) FavoritesSelect(idx int) {
	if idx < 0 || idx >= len(SystemSettings.FavPaths) {
		return
	}
	fi := SystemSettings.FavPaths[idx]
	fp.Directory, _ = homedir.Expand(fi.Path)
	fp.UpdateFilesAction()
}

// SaveSortSettings saves current sorting preferences
func (fp *FilePicker) SaveSortSettings() {
	sv := fp.FilesView()
	if sv == nil {
		return
	}
	SystemSettings.FilePickerSort = sv.SortFieldName()
	// fmt.Printf("sort: %v\n", Settings.FilePickerSort)
	ErrorSnackbar(fp, SaveSettings(SystemSettings), "Error saving settings")
}

// FileComplete finds the possible completions for the file field
func (fp *FilePicker) FileComplete(data any, text string, posLine, posChar int) (md complete.Matches) {
	md.Seed = complete.SeedPath(text)

	var files = []string{}
	for _, f := range fp.files {
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
func (fp *FilePicker) PathComplete(data any, path string, posLine, posChar int) (md complete.Matches) {
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
func (fp *FilePicker) PathCompleteEdit(data any, text string, cursorPos int, c complete.Completion, seed string) (ed complete.Edit) {
	ed = complete.EditWord(text, cursorPos, c.Text, seed)
	path := ed.NewText + string(filepath.Separator)
	ed.NewText = path
	ed.CursorAdjust += 1
	return ed
}

// FileCompleteEdit is the editing function called when inserting the completion selection in the file field
func (fp *FilePicker) FileCompleteEdit(data any, text string, cursorPos int, c complete.Completion, seed string) (ed complete.Edit) {
	ed = complete.EditWord(text, cursorPos, c.Text, seed)
	return ed
}

// EditRecentPaths displays a dialog allowing the user to
// edit the recent paths list.
func (fp *FilePicker) EditRecentPaths() {
	d := NewBody().AddTitle("Recent file paths").AddText("You can delete paths you no longer use")
	NewList(d).SetSlice(&RecentPaths)
	d.AddBottomBar(func(parent Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			SaveRecentPaths()
			fp.Update()
		})
	})
	d.RunDialog(fp)
}

// Filename is used to specify an file path.
// It results in a [FileButton] [Value].
type Filename string

// FileButton represents a [Filename] value with a button
// that opens a [FilePicker].
type FileButton struct {
	Button
	Filename string
}

func (fb *FileButton) WidgetValue() any { return &fb.Filename }

func (fb *FileButton) Init() {
	fb.Button.Init()
	fb.SetType(ButtonTonal).SetIcon(icons.File)
	fb.Updater(func() {
		if fb.Filename == "" {
			fb.SetText("Select file")
		} else {
			fb.SetText(fb.Filename)
		}
	})
	var fp *FilePicker
	InitValueButton(fb, false, func(d *Body) {
		// ext, _ := v.Tag("ext") // TODO(config) (also rename to extension)
		fp = NewFilePicker(d).SetFilename(fb.Filename)
		fb.ValueNewWindow = true
		d.AddAppBar(fp.MakeToolbar)
	}, func() {
		fb.Filename = fp.SelectedFile()
	})
}

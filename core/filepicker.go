// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"

	"cogentcore.org/core/base/elide"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/tree"
)

// todo:

// * search: use highlighting, not filtering -- < > arrows etc
// * also simple search-while typing in grid?
// * filepicker selector DND is a file:/// url

// FilePicker is a widget for selecting files.
type FilePicker struct {
	Frame

	// Filterer is an optional filtering function for which files to display.
	Filterer FilePickerFilterer `display:"-" json:"-" xml:"-"`

	// directory is the absolute path to the directory of files to display.
	directory string

	// selectedFilename is the name of the currently selected file,
	// not including the directory. See [FilePicker.SelectedFile]
	// for the full path.
	selectedFilename string

	// extensions is a list of the target file extensions.
	// If there are multiple, they must be comma separated.
	// The extensions must include the dot (".") at the start.
	// They must be set using [FilePicker.SetExtensions].
	extensions string

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

	favoritesTable, filesTable  *Table
	selectField, extensionField *TextField
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
			fp.directoryUp()
		case keymap.Insert, keymap.InsertAfter, keymap.Open, keymap.SelectMode:
			e.SetHandled()
			if fp.selectFile() {
				fp.Send(events.DoubleClick, e) // will close dialog
			}
		case keymap.Search:
			e.SetHandled()
			sf := fp.selectField
			sf.SetFocus()
		}
	})

	fp.Maker(func(p *tree.Plan) {
		if fp.directory == "" {
			fp.SetFilename("") // default to current directory
		}
		if len(recentPaths) == 0 {
			openRecentPaths()
		}
		recentPaths.AddPath(fp.directory, SystemSettings.SavedPathsMax)
		saveRecentPaths()
		fp.readFiles()

		if fp.prevPath != fp.directory {
			// TODO(#424): disable for all platforms for now; causing issues
			if false && TheApp.Platform() != system.MacOS {
				// mac is not supported in a high-capacity fashion at this point
				if fp.prevPath == "" {
					fp.configWatcher()
				} else {
					fp.watcher.Remove(fp.prevPath)
				}
				fp.watcher.Add(fp.directory)
				if fp.prevPath == "" {
					fp.watchWatcher()
				}
			}
			fp.prevPath = fp.directory
		}

		tree.AddAt(p, "path", func(w *Chooser) {
			Bind(&fp.directory, w)
			w.SetEditable(true).SetDefaultNew(true)
			w.AddItemsFunc(func() {
				fp.addRecentPathItems(&w.Items)
			})
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 0)
			})
			w.OnChange(func(e events.Event) {
				fp.updateFilesEvent()
			})
		})
		tree.AddAt(p, "files", func(w *Frame) {
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
			w.Maker(fp.makeFilesRow)
		})
		tree.AddAt(p, "selected", func(w *Frame) {
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Gap.X.Dp(4)
			})
			w.Maker(fp.makeSelectedRow)
		})
	})
}

func (fp *FilePicker) Destroy() {
	fp.Frame.Destroy()
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

// FilePickerFilterer is a filtering function for files; returns true if the
// file should be visible in the picker, and false if not
type FilePickerFilterer func(fp *FilePicker, fi *fileinfo.FileInfo) bool

// FilePickerDirOnlyFilter is a [FilePickerFilterer] that only shows directories (folders).
func FilePickerDirOnlyFilter(fp *FilePicker, fi *fileinfo.FileInfo) bool {
	return fi.IsDir()
}

// FilePickerExtensionOnlyFilter is a [FilePickerFilterer] that only shows files that
// match the target extensions, and directories.
func FilePickerExtensionOnlyFilter(fp *FilePicker, fi *fileinfo.FileInfo) bool {
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
	fp.directory, fp.selectedFilename = filepath.Split(filename)
	fp.directory = errors.Log1(filepath.Abs(fp.directory))
	return fp
}

// SelectedFile returns the full path to the currently selected file.
func (fp *FilePicker) SelectedFile() string {
	sf := fp.selectField
	sf.editDone()
	return filepath.Join(fp.directory, fp.selectedFilename)
}

// SelectedFileInfo returns the currently selected [fileinfo.FileInfo] or nil.
func (fp *FilePicker) SelectedFileInfo() *fileinfo.FileInfo {
	if fp.selectedIndex < 0 || fp.selectedIndex >= len(fp.files) {
		return nil
	}
	return fp.files[fp.selectedIndex]
}

// selectFile selects the current file as the selection.
// if a directory it opens the directory and returns false.
// if a file it selects the file and returns true.
// if no selection, returns false.
func (fp *FilePicker) selectFile() bool {
	if fi := fp.SelectedFileInfo(); fi != nil {
		if fi.IsDir() {
			fp.directory = filepath.Join(fp.directory, fi.Name)
			fp.selectedFilename = ""
			fp.selectedIndex = -1
			fp.updateFilesEvent()
			return false
		}
		return true
	}
	return false
}

func (fp *FilePicker) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(fp.directoryUp).SetIcon(icons.ArrowUpward).SetKey(keymap.Jump).SetText("Up")
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(fp.addPathToFavorites).SetIcon(icons.Favorite).SetText("Favorite")
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(fp.updateFilesEvent).SetIcon(icons.Refresh).SetText("Update")
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(fp.newFolder).SetIcon(icons.CreateNewFolder)
	})
}

func (fp *FilePicker) addRecentPathItems(items *[]ChooserItem) {
	for _, sp := range recentPaths {
		*items = append(*items, ChooserItem{
			Value: sp,
		})
	}
	// TODO: file picker reset and edit recent paths buttons not working
	// *items = append(*items, ChooserItem{
	// 	Value:           "Reset recent paths",
	// 	Icon:            icons.Refresh,
	// 	SeparatorBefore: true,
	// 	Func: func() {
	// 		recentPaths = make(FilePaths, 1, SystemSettings.SavedPathsMax)
	// 		recentPaths[0] = fp.directory
	// 		fp.Update()
	// 	},
	// })
	// *items = append(*items, ChooserItem{
	// 	Value: "Edit recent paths",
	// 	Icon:  icons.Edit,
	// 	Func: func() {
	// 		fp.editRecentPaths()
	// 	},
	// })
}

func (fp *FilePicker) makeFilesRow(p *tree.Plan) {
	tree.AddAt(p, "favorites", func(w *Table) {
		fp.favoritesTable = w
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
			fp.favoritesSelect(w.SelectedIndex)
		})
		w.Updater(func() {
			w.ResetSelectedIndexes()
		})
	})
	tree.AddAt(p, "files", func(w *Table) {
		fp.filesTable = w
		w.SetReadOnly(true)
		w.SetSlice(&fp.files)
		w.SelectedField = "Name"
		w.SelectedValue = fp.selectedFilename
		if SystemSettings.FilePickerSort != "" {
			w.setSortFieldName(SystemSettings.FilePickerSort)
		}
		w.TableStyler = func(w Widget, s *styles.Style, row, col int) {
			fn := fp.files[row].Name
			ext := strings.ToLower(filepath.Ext(fn))
			if _, has := fp.extensionMap[ext]; has {
				s.Color = colors.Scheme.Primary.Base
			} else {
				s.Color = colors.Scheme.OnSurface
			}
		}
		w.Styler(func(s *styles.Style) {
			s.Cursor = cursors.Pointer
		})
		w.OnSelect(func(e events.Event) {
			fp.fileSelect(w.SelectedIndex)
		})
		w.OnDoubleClick(func(e events.Event) {
			if w.clickSelectEvent(e) {
				if !fp.selectFile() {
					e.SetHandled() // don't pass along; keep dialog open
				} else {
					fp.Scene.sendKey(keymap.Accept, e) // activates Ok button code
				}
			}
		})
		w.ContextMenus = nil
		w.AddContextMenu(func(m *Scene) {
			open := NewButton(m).SetText("Open").SetIcon(icons.Open)
			open.SetTooltip("Open the selected file using the default app")
			open.OnClick(func(e events.Event) {
				TheApp.OpenURL("file://" + fp.SelectedFile())
			})
			if TheApp.Platform() == system.Web {
				open.SetText("Download").SetIcon(icons.Download).SetTooltip("Download this file to your device")
			}
			NewSeparator(m)
			NewButton(m).SetText("Duplicate").SetIcon(icons.FileCopy).
				SetTooltip("Make a copy of the selected file").
				OnClick(func(e events.Event) {
					fn := fp.files[w.SelectedIndex]
					fn.Duplicate()
					fp.updateFilesEvent()
				})
			tip := "Delete moves the selected file to the trash / recycling bin"
			if TheApp.Platform().IsMobile() {
				tip = "Delete deletes the selected file"
			}
			NewButton(m).SetText("Delete").SetIcon(icons.Delete).
				SetTooltip(tip).
				OnClick(func(e events.Event) {
					fn := fp.files[w.SelectedIndex]
					fb := NewSoloFuncButton(w).SetFunc(fn.Delete).SetConfirm(true).SetAfterFunc(fp.updateFilesEvent)
					fb.SetTooltip(tip)
					fb.CallFunc()
				})
			NewButton(m).SetText("Rename").SetIcon(icons.EditNote).
				SetTooltip("Rename the selected file").
				OnClick(func(e events.Event) {
					fn := fp.files[w.SelectedIndex]
					NewSoloFuncButton(w).SetFunc(fn.Rename).SetAfterFunc(fp.updateFilesEvent).CallFunc()
				})
			NewButton(m).SetText("Info").SetIcon(icons.Info).
				SetTooltip("View information about the selected file").
				OnClick(func(e events.Event) {
					fn := fp.files[w.SelectedIndex]
					d := NewBody("Info: " + fn.Name)
					NewForm(d).SetStruct(&fn).SetReadOnly(true)
					d.AddOKOnly().RunWindowDialog(w)
				})
			NewSeparator(m)
			NewFuncButton(m).SetFunc(fp.newFolder).SetIcon(icons.CreateNewFolder)
		})
		// w.Updater(func() {})
	})
}

func (fp *FilePicker) makeSelectedRow(selected *tree.Plan) {
	tree.AddAt(selected, "file-text", func(w *Text) {
		w.SetText("File: ")
		w.SetTooltip("Enter file name here (or select from list above)")
		w.Styler(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
	})

	tree.AddAt(selected, "file", func(w *TextField) {
		fp.selectField = w
		w.SetText(fp.selectedFilename)
		w.SetTooltip(fmt.Sprintf("Enter the file name. Special keys: up/down to move selection; %s or %s to go up to parent folder; %s or %s or %s or %s to select current file (if directory, goes into it, if file, selects and closes); %s or %s for prev / next history item; %s return to this field", keymap.WordLeft.Label(), keymap.Jump.Label(), keymap.SelectMode.Label(), keymap.Insert.Label(), keymap.InsertAfter.Label(), keymap.Open.Label(), keymap.HistPrev.Label(), keymap.HistNext.Label(), keymap.Search.Label()))
		w.SetCompleter(fp, fp.fileComplete, fp.fileCompleteEdit)
		w.Styler(func(s *styles.Style) {
			s.Min.X.Ch(60)
			s.Max.X.Zero()
			s.Grow.Set(1, 0)
		})
		w.OnChange(func(e events.Event) {
			fp.setSelectedFile(w.Text())
		})
		w.OnKeyChord(func(e events.Event) {
			kf := keymap.Of(e.KeyChord())
			if kf == keymap.Accept {
				fp.setSelectedFile(w.Text())
			}
		})
		w.StartFocus()
		w.Updater(func() {
			w.SetText(fp.selectedFilename)
		})
	})

	tree.AddAt(selected, "extension-text", func(w *Text) {
		w.SetText("Extension(s):").SetTooltip("target extension(s) to highlight; if multiple, separate with commas, and include the . at the start")
		w.Styler(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
	})

	tree.AddAt(selected, "extension", func(w *TextField) {
		fp.extensionField = w
		w.SetText(fp.extensions)
		w.OnChange(func(e events.Event) {
			fp.SetExtensions(w.Text()).Update()
		})
	})
}

func (fp *FilePicker) configWatcher() error {
	if fp.watcher != nil {
		return nil
	}
	var err error
	fp.watcher, err = fsnotify.NewWatcher()
	return err
}

func (fp *FilePicker) watchWatcher() {
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

// updateFilesEvent updates the list of files and other views for the current path.
func (fp *FilePicker) updateFilesEvent() { //types:add
	fp.readFiles()
	fp.Update()
	// sf := fv.SelectField()
	// sf.SetFocusEvent()
}

func (fp *FilePicker) readFiles() {
	effpath, err := filepath.EvalSymlinks(fp.directory)
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
	filepath.Walk(effpath, func(path string, info fs.FileInfo, err error) error {
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
		if fp.Filterer != nil {
			keep = fp.Filterer(fp, fi)
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

// updateFavorites updates list of files and other views for current path
func (fp *FilePicker) updateFavorites() {
	sv := fp.favoritesTable
	sv.Update()
}

// addPathToFavorites adds the current path to favorites
func (fp *FilePicker) addPathToFavorites() { //types:add
	dp := fp.directory
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
	if _, found := SystemSettings.FavPaths.findPath(dp); found {
		MessageSnackbar(fp, "Error: path is already on the favorites list")
		return
	}
	fi := favoritePathItem{"folder", fnm, dp}
	SystemSettings.FavPaths = append(SystemSettings.FavPaths, fi)
	ErrorSnackbar(fp, SaveSettings(SystemSettings), "Error saving settings")
	// fv.FileSig.Emit(fv.This, int64(FilePickerFavAdded), fi)
	fp.updateFavorites()
}

// directoryUp moves up one directory in the path
func (fp *FilePicker) directoryUp() { //types:add
	pdr := filepath.Dir(fp.directory)
	if pdr == "" {
		return
	}
	fp.directory = pdr
	fp.updateFilesEvent()
}

// newFolder creates a new folder with the given name in the current directory.
func (fp *FilePicker) newFolder(name string) error { //types:add
	dp := fp.directory
	if dp == "" {
		return nil
	}
	np := filepath.Join(dp, name)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		return err
	}
	fp.updateFilesEvent()
	return nil
}

// setSelectedFile sets the currently selected file to the given name, sends
// a selection event, and updates the selection in the table.
func (fp *FilePicker) setSelectedFile(file string) {
	fp.selectedFilename = file
	sv := fp.filesTable
	ef := fp.extensionField
	exts := ef.Text()
	if !sv.selectFieldValue("Name", fp.selectedFilename) { // not found
		extl := strings.Split(exts, ",")
		if len(extl) == 1 {
			if !strings.HasSuffix(fp.selectedFilename, extl[0]) {
				fp.selectedFilename += extl[0]
			}
		}
	}
	fp.selectedIndex = sv.SelectedIndex
	sf := fp.selectField
	sf.SetText(fp.selectedFilename) // make sure
	fp.Send(events.Select)          // receiver needs to get selectedFile
}

// fileSelect updates the selection with the given selected file index and
// sends a select event.
func (fp *FilePicker) fileSelect(idx int) {
	if idx < 0 {
		return
	}
	fp.saveSortSettings()
	fi := fp.files[idx]
	fp.selectedIndex = idx
	fp.selectedFilename = fi.Name
	sf := fp.selectField
	sf.SetText(fp.selectedFilename)
	fp.Send(events.Select)
}

// SetExtensions sets the [FilePicker.Extensions] to the given comma separated
// list of file extensions, which each must start with a dot (".").
func (fp *FilePicker) SetExtensions(ext string) *FilePicker {
	if ext == "" {
		if fp.selectedFilename != "" {
			ext = strings.ToLower(filepath.Ext(fp.selectedFilename))
		}
	}
	fp.extensions = ext
	exts := strings.Split(fp.extensions, ",")
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

// favoritesSelect selects a favorite path and goes there
func (fp *FilePicker) favoritesSelect(idx int) {
	if idx < 0 || idx >= len(SystemSettings.FavPaths) {
		return
	}
	fi := SystemSettings.FavPaths[idx]
	fp.directory, _ = homedir.Expand(fi.Path)
	fp.updateFilesEvent()
}

// saveSortSettings saves current sorting preferences
func (fp *FilePicker) saveSortSettings() {
	sv := fp.filesTable
	if sv == nil {
		return
	}
	SystemSettings.FilePickerSort = sv.sortFieldName()
	// fmt.Printf("sort: %v\n", Settings.FilePickerSort)
	ErrorSnackbar(fp, SaveSettings(SystemSettings), "Error saving settings")
}

// fileComplete finds the possible completions for the file field
func (fp *FilePicker) fileComplete(data any, text string, posLine, posChar int) (md complete.Matches) {
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

// fileCompleteEdit is the editing function called when inserting the completion selection in the file field
func (fp *FilePicker) fileCompleteEdit(data any, text string, cursorPos int, c complete.Completion, seed string) (ed complete.Edit) {
	ed = complete.EditWord(text, cursorPos, c.Text, seed)
	return ed
}

// editRecentPaths displays a dialog allowing the user to
// edit the recent paths list.
func (fp *FilePicker) editRecentPaths() {
	d := NewBody("Recent file paths")
	NewText(d).SetType(TextSupporting).SetText("You can delete paths you no longer use")
	NewList(d).SetSlice(&recentPaths)
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			saveRecentPaths()
			fp.Update()
		})
	})
	d.RunDialog(fp)
}

// Filename is used to specify an file path.
// It results in a [FileButton] [Value].
type Filename = fsx.Filename

// FileButton represents a [Filename] value with a button
// that opens a [FilePicker].
type FileButton struct {
	Button
	Filename string

	// Extensions are the target file extensions for the file picker.
	Extensions string
}

func (fb *FileButton) WidgetValue() any { return &fb.Filename }

func (fb *FileButton) OnBind(value any, tags reflect.StructTag) {
	if ext, ok := tags.Lookup("extension"); ok {
		fb.SetExtensions(ext)
	}
}

func (fb *FileButton) Init() {
	fb.Button.Init()
	fb.SetType(ButtonTonal).SetIcon(icons.File)
	fb.Updater(func() {
		if fb.Filename == "" {
			fb.SetText("Select file")
		} else {
			fb.SetText(elide.Middle(fb.Filename, 38))
		}
	})
	var fp *FilePicker
	InitValueButton(fb, false, func(d *Body) {
		d.Title = "Select file"
		d.DeleteChildByName("body-title") // file picker has its own title
		fp = NewFilePicker(d).SetFilename(fb.Filename).SetExtensions(fb.Extensions)
		fb.setFlag(true, widgetValueNewWindow)
		d.AddTopBar(func(bar *Frame) {
			NewToolbar(bar).Maker(fp.MakeToolbar)
		})
	}, func() {
		fb.Filename = fp.SelectedFile()
	})
}

func (fb *FileButton) WidgetTooltip(pos image.Point) (string, image.Point) {
	if fb.Filename == "" {
		return fb.Tooltip, fb.DefaultTooltipPos()
	}
	fnm := "(" + fb.Filename + ")"
	if fb.Tooltip == "" {
		return fnm, fb.DefaultTooltipPos()
	}
	return fnm + " " + fb.Tooltip, fb.DefaultTooltipPos()
}

// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

//go:generate core generate

import (
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// NodeHighlighting is the default style for syntax highlighting to use for
// file node buffers
var NodeHighlighting = highlighting.StyleDefault

// Node represents a file in the file system, as a [core.Tree] node.
// The name of the node is the name of the file.
// Folders have children containing further nodes.
type Node struct { //core:embedder
	core.Tree

	// Filepath is the full path to this file.
	Filepath core.Filename `edit:"-" set:"-" json:"-" xml:"-" copier:"-"`

	// Info is the full standard file info about this file.
	Info fileinfo.FileInfo `edit:"-" set:"-" json:"-" xml:"-" copier:"-"`

	// FileIsOpen indicates that this file has been opened, indicated by Italics.
	FileIsOpen bool

	// DirRepo is the version control system repository for this directory,
	// only non-nil if this is the highest-level directory in the tree under vcs control.
	DirRepo vcs.Repo `edit:"-" set:"-" json:"-" xml:"-" copier:"-"`
}

func (fn *Node) AsFileNode() *Node {
	return fn
}

// FileRoot returns the Root node as a [Tree].
func (fn *Node) FileRoot() *Tree {
	return AsTree(fn.Root)
}

func (fn *Node) Init() {
	fn.Tree.Init()
	fn.IconOpen = icons.FolderOpen
	fn.IconClosed = icons.Folder
	fn.ContextMenus = nil // do not include tree
	fn.AddContextMenu(fn.contextMenu)
	fn.Styler(func(s *styles.Style) {
		s.IconSize.Set(units.Em(1))
		fn.styleFromStatus()
	})
	fn.On(events.KeyChord, func(e events.Event) {
		if core.DebugSettings.KeyEventTrace {
			fmt.Printf("Tree KeyInput: %v\n", fn.Path())
		}
		kf := keymap.Of(e.KeyChord())
		selMode := events.SelectModeBits(e.Modifiers())

		if selMode == events.SelectOne {
			if fn.SelectMode {
				selMode = events.ExtendContinuous
			}
		}

		// first all the keys that work for ReadOnly and active
		if !fn.IsReadOnly() && !e.IsHandled() {
			switch kf {
			case keymap.Delete:
				fn.This.(Filer).DeleteFiles()
				e.SetHandled()
			case keymap.Backspace:
				fn.This.(Filer).DeleteFiles()
				e.SetHandled()
			case keymap.Duplicate:
				fn.duplicateFiles()
				e.SetHandled()
			case keymap.Insert: // New File
				core.CallFunc(fn, fn.newFile)
				e.SetHandled()
			case keymap.InsertAfter: // New Folder
				core.CallFunc(fn, fn.newFolder)
				e.SetHandled()
			}
		}
	})

	fn.Parts.Styler(func(s *styles.Style) {
		s.Gap.X.Em(0.4)
	})
	fn.Parts.OnClick(func(e events.Event) {
		if !e.HasAnyModifier(key.Control, key.Meta, key.Alt, key.Shift) {
			fn.Open()
		}
	})
	fn.Parts.OnDoubleClick(func(e events.Event) {
		e.SetHandled()
		if fn.IsDir() {
			fn.ToggleClose()
		} else {
			fn.This.(Filer).OpenFile()
		}
	})
	tree.AddChildInit(fn.Parts, "text", func(w *core.Text) {
		w.Styler(func(s *styles.Style) {
			if fn.IsExec() && !fn.IsDir() {
				s.Font.Weight = rich.Bold
			}
			if fn.FileIsOpen {
				s.Font.Slant = rich.Italic
			}
		})
	})

	fn.Updater(func() {
		fn.setFileIcon()
		if fn.IsDir() {
			repo, rnode := fn.Repo()
			if repo != nil && rnode.This == fn.This {
				rnode.updateRepoFiles()
			}
		} else {
			fn.This.(Filer).GetFileInfo()
		}
		fn.Text = fn.Info.Name
		cc := fn.Styles.Color
		fn.styleFromStatus()
		if fn.Styles.Color != cc && fn.Parts != nil {
			fn.Parts.StyleTree()
		}
	})

	fn.Maker(func(p *tree.Plan) {
		if fn.Filepath == "" {
			return
		}
		if fn.Name == externalFilesName {
			files := fn.FileRoot().externalFiles
			for _, fi := range files {
				tree.AddNew(p, fi, func() Filer {
					return tree.NewOfType(fn.FileRoot().FileNodeType).(Filer)
				}, func(wf Filer) {
					w := wf.AsFileNode()
					w.Root = fn.Root
					w.NeedsLayout()
					w.Filepath = core.Filename(fi)
					w.Info.Mode = os.ModeIrregular
					w.Info.VCS = vcs.Stored
				})
			}
			return
		}
		if !fn.IsDir() || fn.IsIrregular() {
			return
		}
		if !((fn.FileRoot().inOpenAll && !fn.Info.IsHidden()) || fn.FileRoot().isDirOpen(fn.Filepath)) {
			return
		}
		repo, _ := fn.Repo()
		files := fn.dirFileList()
		for _, fi := range files {
			fpath := filepath.Join(string(fn.Filepath), fi.Name())
			if fn.FileRoot().FilterFunc != nil && !fn.FileRoot().FilterFunc(fpath, fi) {
				continue
			}
			tree.AddNew(p, fi.Name(), func() Filer {
				return tree.NewOfType(fn.FileRoot().FileNodeType).(Filer)
			}, func(wf Filer) {
				w := wf.AsFileNode()
				w.Root = fn.Root
				w.NeedsLayout()
				w.Filepath = core.Filename(fpath)
				w.This.(Filer).GetFileInfo()
				if w.FileRoot().FS == nil {
					if w.IsDir() && repo == nil {
						w.detectVCSRepo()
					}
				}
			})
		}
	})
}

// styleFromStatus updates font color from
func (fn *Node) styleFromStatus() {
	status := fn.Info.VCS
	hex := ""
	switch {
	case status == vcs.Untracked:
		hex = "#808080"
	case status == vcs.Modified:
		hex = "#4b7fd1"
	case status == vcs.Added:
		hex = "#008800"
	case status == vcs.Deleted:
		hex = "#ff4252"
	case status == vcs.Conflicted:
		hex = "#ce8020"
	case status == vcs.Updated:
		hex = "#008060"
	case status == vcs.Stored:
		fn.Styles.Color = colors.Scheme.OnSurface
	}
	if fn.Info.Generated {
		hex = "#8080C0"
	}
	if hex != "" {
		fn.Styles.Color = colors.Uniform(colors.ToBase(errors.Must1(colors.FromHex(hex))))
	} else {
		fn.Styles.Color = colors.Scheme.OnSurface
	}
	// if fn.Name == "test.go" {
	// 	rep, err := fn.Repo()
	// 	fmt.Println("style updt:", status, hex, rep != nil, err)
	// }
}

// IsDir returns true if file is a directory (folder)
func (fn *Node) IsDir() bool {
	return fn.Info.IsDir()
}

// IsIrregular  returns true if file is a special "Irregular" node
func (fn *Node) IsIrregular() bool {
	return (fn.Info.Mode & os.ModeIrregular) != 0
}

// isExternal returns true if file is external to main file tree
func (fn *Node) isExternal() bool {
	isExt := false
	fn.WalkUp(func(k tree.Node) bool {
		sfn := AsNode(k)
		if sfn == nil {
			return tree.Break
		}
		if sfn.IsIrregular() {
			isExt = true
			return tree.Break
		}
		return tree.Continue
	})
	return isExt
}

// IsExec returns true if file is an executable file
func (fn *Node) IsExec() bool {
	return fn.Info.IsExec()
}

// isOpen returns true if file is flagged as open
func (fn *Node) isOpen() bool {
	return !fn.Closed
}

// isAutoSave returns true if file is an auto-save file (starts and ends with #)
func (fn *Node) isAutoSave() bool {
	return strings.HasPrefix(fn.Info.Name, "#") && strings.HasSuffix(fn.Info.Name, "#")
}

// RelativePath returns the relative path from root for this node
func (fn *Node) RelativePath() string {
	if fn.IsIrregular() || fn.FileRoot() == nil {
		return fn.Name
	}
	return fsx.RelativeFilePath(string(fn.Filepath), string(fn.FileRoot().Filepath))
}

// dirFileList returns the list of files in this directory,
// sorted according to DirsOnTop and SortByModTime options
func (fn *Node) dirFileList() []fs.FileInfo {
	path := string(fn.Filepath)
	var files []fs.FileInfo
	var dirs []fs.FileInfo // for DirsOnTop mode
	var di []fs.DirEntry
	isFS := false
	if fn.FileRoot().FS == nil {
		di = errors.Log1(os.ReadDir(path))
	} else {
		isFS = true
		di = errors.Log1(fs.ReadDir(fn.FileRoot().FS, path))
	}
	for _, d := range di {
		info := errors.Log1(d.Info())
		if fn.FileRoot().DirsOnTop {
			if d.IsDir() {
				dirs = append(dirs, info)
			} else {
				files = append(files, info)
			}
		} else {
			files = append(files, info)
		}
	}
	doModSort := fn.FileRoot().SortByModTime
	if doModSort {
		doModSort = !fn.FileRoot().dirSortByName(core.Filename(path))
	} else {
		doModSort = fn.FileRoot().dirSortByModTime(core.Filename(path))
	}

	if fn.FileRoot().DirsOnTop {
		if doModSort {
			sortByModTime(dirs, isFS) // note: FS = ascending, otherwise descending
			sortByModTime(files, isFS)
		}
		files = append(dirs, files...)
	} else {
		if doModSort {
			sortByModTime(files, isFS)
		}
	}
	return files
}

// sortByModTime sorts by _reverse_ mod time (newest first)
func sortByModTime(files []fs.FileInfo, ascending bool) {
	slices.SortFunc(files, func(a, b fs.FileInfo) int {
		if ascending {
			return a.ModTime().Compare(b.ModTime())
		}
		return b.ModTime().Compare(a.ModTime())
	})
}

func (fn *Node) setFileIcon() {
	if fn.Info.Ic == "" {
		ic, hasic := fn.Info.FindIcon()
		if hasic {
			fn.Info.Ic = ic
		} else {
			fn.Info.Ic = icons.Blank
		}
	}
	fn.IconLeaf = fn.Info.Ic
	if br := fn.Branch; br != nil {
		if br.IconIndeterminate != fn.IconLeaf {
			br.SetIconOn(icons.FolderOpen).SetIconOff(icons.Folder).SetIconIndeterminate(fn.IconLeaf)
			br.UpdateTree()
		}
	}
}

// GetFileInfo is a Filer interface method that can be overwritten
// to do custom file info.
func (fn *Node) GetFileInfo() error {
	return fn.InitFileInfo()
}

// InitFileInfo initializes file info
func (fn *Node) InitFileInfo() error {
	if fn.Filepath == "" {
		return nil
	}
	var err error
	if fn.FileRoot().FS == nil { // deal with symlinks
		ls, err := os.Lstat(string(fn.Filepath))
		if errors.Log(err) != nil {
			return err
		}
		if ls.Mode()&os.ModeSymlink != 0 {
			effpath, err := filepath.EvalSymlinks(string(fn.Filepath))
			if err != nil {
				// this happens too often for links -- skip
				// log.Printf("filetree.Node Path: %v could not be opened -- error: %v\n", fn.Filepath, err)
				return err
			}
			fn.Filepath = core.Filename(effpath)
		}
		err = fn.Info.InitFile(string(fn.Filepath))
	} else {
		err = fn.Info.InitFileFS(fn.FileRoot().FS, string(fn.Filepath))
	}
	if err != nil {
		emsg := fmt.Errorf("filetree.Node InitFileInfo Path %q: Error: %v", fn.Filepath, err)
		log.Println(emsg)
		return emsg
	}
	repo, rnode := fn.Repo()
	if repo != nil {
		if fn.IsDir() {
			fn.Info.VCS = vcs.Stored // always
		} else {
			rstat := rnode.DirRepo.StatusFast(string(fn.Filepath))
			if rstat != fn.Info.VCS {
				fn.Info.VCS = rstat
				fn.NeedsRender()
			}
		}
	} else {
		fn.Info.VCS = vcs.Stored
	}
	return nil
}

// SelectedFunc runs the given function on all selected nodes in reverse order.
func (fn *Node) SelectedFunc(fun func(n *Node)) {
	sels := fn.GetSelectedNodes()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i])
		if sn == nil {
			continue
		}
		fun(sn)
	}
}

func (fn *Node) OnOpen() {
	fn.openDir()
}

func (fn *Node) OnClose() {
	if !fn.IsDir() {
		return
	}
	fn.FileRoot().setDirClosed(fn.Filepath)
}

func (fn *Node) CanOpen() bool {
	return fn.HasChildren() || fn.IsDir()
}

// openDir opens given directory node
func (fn *Node) openDir() {
	if !fn.IsDir() {
		return
	}
	fn.FileRoot().setDirOpen(fn.Filepath)
	fn.Update()
}

// sortBys determines how to sort the selected files in the directory.
// Default is alpha by name, optionally can be sorted by modification time.
func (fn *Node) sortBys(modTime bool) { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.sortBy(modTime)
	})
}

// sortBy determines how to sort the files in the directory -- default is alpha by name,
// optionally can be sorted by modification time.
func (fn *Node) sortBy(modTime bool) {
	fn.FileRoot().setDirSortBy(fn.Filepath, modTime)
	fn.Update()
}

// openAll opens all directories under this one
func (fn *Node) openAll() { //types:add
	fn.FileRoot().inOpenAll = true // causes chaining of opening
	fn.Tree.OpenAll()
	fn.FileRoot().inOpenAll = false
}

// removeFromExterns removes file from list of external files
func (fn *Node) removeFromExterns() { //types:add
	fn.SelectedFunc(func(sn *Node) {
		if !sn.isExternal() {
			return
		}
		sn.FileRoot().removeExternalFile(string(sn.Filepath))
		sn.Delete()
	})
}

// RelativePathFrom returns the relative path from node for given full path
func (fn *Node) RelativePathFrom(fpath core.Filename) string {
	return fsx.RelativeFilePath(string(fpath), string(fn.Filepath))
}

// dirsTo opens all the directories above the given filename, and returns the node
// for element at given path (can be a file or directory itself -- not opened -- just returned)
func (fn *Node) dirsTo(path string) (*Node, error) {
	pth, err := filepath.Abs(path)
	if err != nil {
		log.Printf("filetree.Node DirsTo path %v could not be turned into an absolute path: %v\n", path, err)
		return nil, err
	}
	rpath := fn.RelativePathFrom(core.Filename(pth))
	if rpath == "." {
		return fn, nil
	}
	dirs := strings.Split(rpath, string(filepath.Separator))
	cfn := fn
	sz := len(dirs)
	for i := 0; i < sz; i++ {
		dr := dirs[i]
		sfni := cfn.ChildByName(dr, 0)
		if sfni == nil {
			if i == sz-1 { // ok for terminal -- might not exist yet
				return cfn, nil
			}
			err = fmt.Errorf("filetree.Node could not find node %v in: %v, orig: %v, rel: %v", dr, cfn.Filepath, pth, rpath)
			// slog.Error(err.Error()) // note: this is expected sometimes
			return nil, err
		}
		sfn := AsNode(sfni)
		if sfn.IsDir() || i == sz-1 {
			if i < sz-1 && !sfn.isOpen() {
				sfn.openDir()
			} else {
				cfn = sfn
			}
		} else {
			err := fmt.Errorf("filetree.Node non-terminal node %v is not a directory in: %v", dr, cfn.Filepath)
			slog.Error(err.Error())
			return nil, err
		}
		cfn = sfn
	}
	return cfn, nil
}

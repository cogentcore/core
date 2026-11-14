// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package filetree provides a GUI tree interface to a filesystem.
package filetree

//go:generate core generate

import (
	"cogentcore.org/core/base/dirsort"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
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
		// (an app opens by double-clicking; its arrow still opens it as a directory)
		if !e.HasAnyModifier(key.Control, key.Meta, key.Alt, key.Shift) && !fn.IsApp() {
			fn.Open()
		}
	})
	fn.Parts.OnDoubleClick(func(e events.Event) {
		e.SetHandled()
		switch {
		case fn.IsApp():
			fn.OpenFileDefault() // (not OpenFile, which can open it in an editor)
		case fn.IsDir():
			fn.ToggleClose()
		default:
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
			fpath := fn.joinPath(string(fn.Filepath), fi.Name())
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

// joinPath joins the given directory and name of a file in it: paths in
// [Tree.FS] always use / (see [io/fs]), whereas os paths use the separator
// of the platform (\ on Windows).
func (fn *Node) joinPath(dir, name string) string {
	if fn.FileRoot().FS != nil {
		return path.Join(dir, name)
	}
	return filepath.Join(dir, name)
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

// AppBundles is whether directories named *.app are app bundles, which
// are shown and opened as apps (see [Node.IsApp]): on macOS, as in the Finder.
var AppBundles = runtime.GOOS == "darwin"

// IsAppBundle returns whether the given path, which is a directory if
// isDir, is an app bundle: a directory named *.app, when [AppBundles].
func IsAppBundle(path string, isDir bool) bool {
	return AppBundles && isDir && strings.EqualFold(filepath.Ext(path), ".app")
}

// IsApp returns whether the node is an app bundle (see [IsAppBundle]),
// unless the tree shows them as directories ([Tree.AppsAsDirs]): it has
// the icon of an app, and double-clicking opens it as an app, without
// opening it in the tree, which its arrow does, for its files.
func (fn *Node) IsApp() bool {
	if !IsAppBundle(string(fn.Filepath), fn.IsDir()) {
		return false
	}
	root := fn.FileRoot()
	return root == nil || !root.AppsAsDirs
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

// dirFileList returns the list of files in this directory, in its sort
// order (see [Node.dirSort]), with the directories first if DirsOnTop.
func (fn *Node) dirFileList() []fs.FileInfo {
	path := string(fn.Filepath)
	var files []fs.FileInfo
	var dirs []fs.FileInfo // for DirsOnTop mode
	var di []fs.DirEntry
	sort := fn.dirSort()
	if fn.FileRoot().FS == nil {
		di = errors.Log1(os.ReadDir(path))
	} else {
		di = errors.Log1(fs.ReadDir(fn.FileRoot().FS, path))
	}
	for _, d := range di {
		info, err := d.Info()
		if err != nil {
			continue // (as one that was removed since it was read)
		}
		if fn.FileRoot().DirsOnTop && d.IsDir() {
			dirs = append(dirs, info)
		} else {
			files = append(files, info)
		}
	}
	sort.SortInfos(dirs)
	sort.SortInfos(files)
	return append(dirs, files...)
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
		on, off := icons.FolderOpen, icons.Folder
		if fn.IsApp() {
			on, off = icons.Apps, icons.Apps
		}
		if br.IconIndeterminate != fn.IconLeaf || br.IconOff != off {
			br.SetIconOn(on).SetIconOff(off).SetIconIndeterminate(fn.IconLeaf)
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
		var ls os.FileInfo
		ls, err = os.Lstat(string(fn.Filepath))
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
		if tree.IsNil(sn) {
			continue
		}
		fun(sn)
	}
}

// selectedTop returns the selected nodes that are not within another
// selected directory, for operations that also apply to the contents of
// directories (such as deleting).
func (fn *Node) selectedTop() []*Node {
	var sels []*Node
	fn.SelectedFunc(func(sn *Node) {
		sels = append(sels, sn)
	})
	var top []*Node
	for _, sn := range sels {
		if !slices.ContainsFunc(sels, func(dn *Node) bool { return isWithin(string(sn.Filepath), string(dn.Filepath)) }) {
			top = append(top, sn)
		}
	}
	return top
}

// isWithin returns whether the given path is within the given directory path.
func isWithin(path, dir string) bool {
	return strings.HasPrefix(path, dir+string(filepath.Separator))
}

// topPaths returns the given paths without those within another of them.
func topPaths(paths []string) []string {
	var top []string
	for _, p := range paths {
		if !slices.ContainsFunc(paths, func(d string) bool { return isWithin(p, d) }) {
			top = append(top, p)
		}
	}
	return top
}

func (fn *Node) OnOpen() {
	fn.openDir()
}

func (fn *Node) OnClose() {
	if !fn.IsDir() {
		return
	}
	fn.FileRoot().setDirClosed(fn.Filepath)
	fn.FileRoot().syncWatchDirs()
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
	if !fn.FileRoot().inOpenAll {
		fn.FileRoot().syncWatchDirs()
	}
}

// sortBys sets the sort order of the selected directories, by the given
// field of their files, up (ascending) or down, which is their sort order
// in all apps (see [dirsort.Set]).
func (fn *Node) sortBys(by dirsort.Fields, down bool) { //types:add
	fn.SelectedFunc(func(sn *Node) {
		sn.sortBy(dirsort.Sort{By: by, Down: down})
	})
}

// sortBy sets the sort order of the files in this directory, which is its
// sort order in all apps (see [dirsort.Set]), or in this tree for a
// directory of a [Tree.FS].
func (fn *Node) sortBy(s dirsort.Sort) {
	if root := fn.FileRoot(); root.FS != nil {
		if root.fsSorts == nil {
			root.fsSorts = map[string]dirsort.Sort{}
		}
		root.fsSorts[string(fn.Filepath)] = s
	} else {
		core.ErrorSnackbar(fn, dirsort.Set(string(fn.Filepath), s), "Could not save the sort order")
	}
	fn.Update()
}

// openAll opens all directories under this one
func (fn *Node) openAll() { //types:add
	fn.FileRoot().inOpenAll = true // causes chaining of opening
	fn.Tree.OpenAll()
	fn.FileRoot().inOpenAll = false
	fn.FileRoot().syncWatchDirs()
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
	pth := path
	sep := string(filepath.Separator)
	var err error
	if fn.isFS() { // (paths within the FS, separated by /)
		sep = "/"
	} else if pth, err = filepath.Abs(path); err != nil {
		log.Printf("filetree.Node DirsTo path %v could not be turned into an absolute path: %v\n", path, err)
		return nil, err
	}
	rpath := fn.RelativePathFrom(core.Filename(pth))
	if fn.isFS() {
		rpath = filepath.ToSlash(rpath)
	}
	if rpath == "." {
		return fn, nil
	}
	dirs := strings.Split(rpath, sep)
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

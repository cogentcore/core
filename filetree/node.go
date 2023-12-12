// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

//go:generate goki generate

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"goki.dev/enums"
	"goki.dev/fi"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/gi/v2/texteditor/histyle"
	"goki.dev/glop/dirs"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/vci/v2"
)

// NodeHiStyle is the default style for syntax highlighting to use for
// file node buffers
var NodeHiStyle = histyle.StyleDefault

// Node represents a file in the file system, as a TreeView node.
// The name of the node is the name of the file.
// Folders have children containing further nodes.
type Node struct { //goki:embedder
	giv.TreeView

	// full path to this file
	FPath gi.FileName `edit:"-" set:"-" json:"-" xml:"-" copy:"-"`

	// full standard file info about this file
	Info fi.FileInfo `edit:"-" set:"-" json:"-" xml:"-" copy:"-"`

	// file buffer for editing this file
	Buf *texteditor.Buf `edit:"-" set:"-" json:"-" xml:"-" copy:"-"`

	// root of the tree -- has global state
	FRoot *Tree `edit:"-" set:"-" json:"-" xml:"-" copy:"-"`

	// version control system repository for this directory,
	// only non-nil if this is the highest-level directory in the tree under vcs control
	DirRepo vci.Repo `edit:"-" set:"-" json:"-" xml:"-" copy:"-"`

	// version control system repository file status -- only valid during ReadDir
	RepoFiles vci.Files `edit:"-" set:"-" json:"-" xml:"-" copy:"-"`
}

func (fn *Node) FlagType() enums.BitFlagSetter {
	return (*NodeFlags)(&fn.Flags)
}

//	func (fn *Node) CopyFieldsFrom(frm any) {
//		// note: not copying ki.Node as it doesn't have any copy fields
//		// fr := frm.(*Node)
//		// and indeed nothing here should be copied!
//	}

// NodeFlags define bitflags for Node state -- these extend TreeViewFlags
// and storage is an int64
type NodeFlags giv.TreeViewFlags //enums:bitflag -trim-prefix Node

const (
	// NodeOpen means file is open. For directories, this means that
	// sub-files should be / have been loaded. For files, means that they
	// have been opened e.g., for editing.
	NodeOpen NodeFlags = NodeFlags(giv.TreeViewFlagsN) + iota

	// NodeSymLink indicates that file is a symbolic link.
	// File info is all for the target of the symlink.
	NodeSymLink
)

func (fn *Node) BaseType() *gti.Type {
	return fn.KiType()
}

// IsDir returns true if file is a directory (folder)
func (fn *Node) IsDir() bool {
	return fn.Info.IsDir()
}

// IsIrregular  returns true if file is a special "Irregular" node
func (fn *Node) IsIrregular() bool {
	return (fn.Info.Mode & os.ModeIrregular) != 0
}

// IsExternal returns true if file is external to main file tree
func (fn *Node) IsExternal() bool {
	isExt := false
	fn.WalkUp(func(k ki.Ki) bool {
		sfn := AsNode(k)
		if sfn == nil {
			return ki.Break
		}
		if sfn.IsIrregular() {
			isExt = true
			return ki.Break
		}
		return ki.Continue
	})
	return isExt
}

// HasClosedParent returns true if node has a parent node with !IsOpen flag set
func (fn *Node) HasClosedParent() bool {
	hasClosed := false
	fn.WalkUpParent(func(k ki.Ki) bool {
		sfn := AsNode(k)
		if sfn == nil {
			return ki.Break
		}
		if !sfn.IsOpen() {
			hasClosed = true
			return ki.Break
		}
		return ki.Continue
	})
	return hasClosed
}

// IsSymLink returns true if file is a symlink
func (fn *Node) IsSymLink() bool {
	return fn.Is(NodeSymLink)
}

// IsExec returns true if file is an executable file
func (fn *Node) IsExec() bool {
	return fn.Info.IsExec()
}

// IsOpen returns true if file is flagged as open
func (fn *Node) IsOpen() bool {
	return !fn.IsClosed()
}

// IsChanged returns true if the file is open and has been changed (edited) since last EditDone
func (fn *Node) IsChanged() bool {
	if fn.Buf != nil && fn.Buf.IsChanged() {
		return true
	}
	return false
}

// IsNotSaved returns true if the file is open and has been changed (edited) since last Save
func (fn *Node) IsNotSaved() bool {
	if fn.Buf != nil && fn.Buf.IsNotSaved() {
		return true
	}
	return false
}

// IsAutoSave returns true if file is an auto-save file (starts and ends with #)
func (fn *Node) IsAutoSave() bool {
	if strings.HasPrefix(fn.Info.Name, "#") && strings.HasSuffix(fn.Info.Name, "#") {
		return true
	}
	return false
}

// MyRelPath returns the relative path from root for this node
func (fn *Node) MyRelPath() string {
	if fn.IsIrregular() {
		return fn.Nm
	}
	return dirs.RelFilePath(string(fn.FPath), string(fn.FRoot.FPath))
}

// ReadDir reads all the files at given directory into this directory node --
// uses config children to preserve extra info already stored about files.
// The root node represents the directory at the given path.
// Returns os.Stat error if path cannot be accessed.
func (fn *Node) ReadDir(path string) error {
	_, fnm := filepath.Split(path)
	fn.SetText(fnm)
	pth, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	fn.FPath = gi.FileName(pth)
	err = fn.Info.InitFile(string(fn.FPath))
	if err != nil {
		log.Printf("giv.Tree: could not read directory: %v err: %v\n", fn.FPath, err)
		return err
	}

	fn.UpdateDir()
	return nil
}

// UpdateDir updates the directory and all the nodes under it
func (fn *Node) UpdateDir() {
	fn.DetectVCSRepo(true) // update files
	path := string(fn.FPath)
	// fmt.Printf("path: %v  node: %v\n", path, fn.Path())
	repo, rnode := fn.Repo()
	fn.Open() // ensure
	config := fn.ConfigOfFiles(path)
	hasExtFiles := false
	if fn.This() == fn.FRoot.This() {
		if len(fn.FRoot.ExtFiles) > 0 {
			config = append(ki.Config{{Type: fn.FRoot.NodeType, Name: ExternalFilesName}}, config...)
			hasExtFiles = true
		}
	}
	mods, updt := fn.ConfigChildren(config) // NOT unique names
	if mods {
		// fmt.Printf("got mods: %v\n", path)
	}
	// always go through kids, regardless of mods
	for _, sfk := range fn.Kids {
		sf := AsNode(sfk)
		sf.FRoot = fn.FRoot
		if hasExtFiles && sf.Nm == ExternalFilesName {
			fn.FRoot.UpdateExtFiles(sf)
			continue
		}
		fp := filepath.Join(path, sf.Nm)
		// if sf.Buf != nil {
		// 	fmt.Printf("fp: %v  nm: %v\n", fp, sf.Nm)
		// }
		sf.SetNodePath(fp)
		if sf.IsDir() {
			sf.Info.Vcs = vci.Stored // always
		} else if repo != nil {
			rstat := rnode.RepoFiles.Status(repo, string(sf.FPath))
			sf.Info.Vcs = rstat
		} else {
			sf.Info.Vcs = vci.Stored
		}
	}
	if mods {
		root := fn.FRoot
		fn.Update()
		if root != nil {
			root.TreeViewChanged(nil)
		}
		fn.UpdateEndLayout(updt)
	}
}

// ConfigOfFiles returns a type-and-name list for configuring nodes based on
// files immediately within given path
func (fn *Node) ConfigOfFiles(path string) ki.Config {
	config1 := ki.Config{}
	config2 := ki.Config{}
	typ := fn.FRoot.NodeType
	filepath.Walk(path, func(pth string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("giv.Node ConfigFilesIn Path %q: Error: %v", path, err)
			log.Println(emsg)
			return nil // ignore
		}
		if pth == path { // proceed..
			return nil
		}
		_, fnm := filepath.Split(pth)
		if fn.FRoot.DirsOnTop {
			if info.IsDir() {
				config1.Add(typ, fnm)
			} else {
				config2.Add(typ, fnm)
			}
		} else {
			config1.Add(typ, fnm)
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	modSort := fn.FRoot.DirSortByModTime(gi.FileName(path))
	if fn.FRoot.DirsOnTop {
		if modSort {
			fn.SortConfigByModTime(config2) // just sort files, not dirs
		}
		config1 = append(config1, config2...)
	} else {
		if modSort {
			fn.SortConfigByModTime(config1) // all
		}
	}
	return config1
}

// SortConfigByModTime sorts given config list by mod time
func (fn *Node) SortConfigByModTime(confg ki.Config) {
	sort.Slice(confg, func(i, j int) bool {
		ifn, _ := os.Stat(filepath.Join(string(fn.FPath), confg[i].Name))
		jfn, _ := os.Stat(filepath.Join(string(fn.FPath), confg[j].Name))
		return ifn.ModTime().After(jfn.ModTime()) // descending
	})
}

func (fn *Node) SetFileIcon() {
	ic, hasic := fn.Info.FindIcon()
	if !hasic {
		ic = icons.Blank
	}
	if bp, ok := fn.BranchPart(); ok {
		if bp.IconUnk != ic {
			bp.IconUnk = ic
			bp.Update()
			fn.SetNeedsRender(true)
		}
	}
}

// SetNodePath sets the path for given node and updates it based on associated file
func (fn *Node) SetNodePath(path string) error {
	pth, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	fn.FPath = gi.FileName(pth)
	err = fn.InitFileInfo()
	if err != nil {
		return err
	}
	if fn.IsDir() && !fn.IsIrregular() {
		openAll := fn.FRoot.InOpenAll && !fn.Info.IsHidden()
		if openAll || fn.FRoot.IsDirOpen(fn.FPath) {
			fn.ReadDir(string(fn.FPath)) // keep going down..
		}
	}
	fn.SetFileIcon()
	return nil
}

// InitFileInfo initializes file info
func (fn *Node) InitFileInfo() error {
	effpath, err := filepath.EvalSymlinks(string(fn.FPath))
	if err != nil {
		// this happens too often for links -- skip
		// log.Printf("giv.Node Path: %v could not be opened -- error: %v\n", fn.FPath, err)
		return err
	}
	fn.FPath = gi.FileName(effpath)
	err = fn.Info.InitFile(string(fn.FPath))
	if err != nil {
		emsg := fmt.Errorf("giv.Node InitFileInfo Path %q: Error: %v", fn.FPath, err)
		log.Println(emsg)
		return emsg
	}
	return nil
}

// UpdateNode updates information in node based on its associated file in FPath.
// This is intended to be called ad-hoc for individual nodes that might need
// updating -- use ReadDir for mass updates as it is more efficient.
func (fn *Node) UpdateNode() error {
	err := fn.InitFileInfo()
	if err != nil {
		return err
	}
	if fn.IsIrregular() {
		return nil
	}
	if fn.IsDir() {
		openAll := fn.FRoot.InOpenAll && !fn.Info.IsHidden()
		if openAll || fn.FRoot.IsDirOpen(fn.FPath) {
			// fmt.Printf("set open: %s\n", fn.FPath)
			fn.Open()
			repo, rnode := fn.Repo()
			if repo != nil {
				rnode.UpdateRepoFiles()
			}
			fn.UpdateDir()
		}
	} else {
		repo, _ := fn.Repo()
		if repo != nil {
			fn.Info.Vcs, _ = repo.Status(string(fn.FPath))
		}
		fn.SetFileIcon()
		fn.SetNeedsRender(true)
	}
	return nil
}

func (fn *Node) UpdateBranchIcons() {
	fn.SetFileIcon()
}

// OpenDirs opens directories for selected views
func (fn *Node) OpenDirs() {
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		fn := AsNode(sels[i].This())
		fn.OpenDir()
	}
}

func (fn *Node) OnOpen() {
	fn.OpenDir()
}

// OpenDir opens given directory node
func (fn *Node) OpenDir() {
	// fmt.Printf("fn: %s opened\n", fn.FPath)
	fn.FRoot.SetDirOpen(fn.FPath)
	fn.UpdateNode()
}

func (fn *Node) OnClose() {
	fn.CloseDir()
}

// CloseDir closes given directory node -- updates memory state
func (fn *Node) CloseDir() {
	// fmt.Printf("fn: %s closed\n", fn.FPath)
	fn.FRoot.SetDirClosed(fn.FPath)
	// note: not doing anything with open files within directory..
}

// OpenEmptyDir will attempt to open a directory that has no children
// which presumably was not processed originally
func (fn *Node) OpenEmptyDir() bool {
	if fn.IsDir() && !fn.HasChildren() {
		updt := fn.UpdateStart()
		fn.OpenDir()
		fn.Open()
		fn.Update()
		fn.UpdateNode() // needs a second pass
		fn.UpdateEndLayout(updt)
		return true
	}
	return false
}

// SortBys determines how to sort the selected files in the directory.
// Default is alpha by name, optionally can be sorted by modification time.
func (fn *Node) SortBys(modTime bool) { //gti:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		sn.SortBy(modTime)
	}
}

// SortBy determines how to sort the files in the directory -- default is alpha by name,
// optionally can be sorted by modification time.
func (fn *Node) SortBy(modTime bool) {
	fn.FRoot.SetDirSortBy(fn.FPath, modTime)
	fn.SetNeedsLayout(true)
}

// OpenAll opens all directories under this one
func (fn *Node) OpenAll() { //gti:add
	fn.FRoot.InOpenAll = true // causes chaining of opening
	fn.TreeView.OpenAll()
	fn.FRoot.InOpenAll = false
}

// CloseAll closes all directories under this one, this included
func (fn *Node) CloseAll() { //gti:add
	fn.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		sfn := AsNode(wi)
		if sfn == nil {
			return ki.Continue
		}
		if sfn.IsDir() {
			sfn.Close()
		}
		return ki.Continue
	})
}

// OpenBuf opens the file in its buffer if it is not already open.
// returns true if file is newly opened
func (fn *Node) OpenBuf() (bool, error) {
	if fn.IsDir() {
		err := fmt.Errorf("giv.Node cannot open directory in editor: %v", fn.FPath)
		log.Println(err)
		return false, err
	}
	if fn.Buf != nil {
		if fn.Buf.Filename == fn.FPath { // close resets filename
			return false, nil
		}
	} else {
		fn.Buf = texteditor.NewBuf()
		fn.Buf.OnChange(func(e events.Event) {
			if fn.Info.Vcs == vci.Stored {
				fn.Info.Vcs = vci.Modified
			}
		})
	}
	fn.Buf.Hi.Style = NodeHiStyle
	return true, fn.Buf.Open(fn.FPath)
}

// RemoveFromExterns removes file from list of external files
func (fn *Node) RemoveFromExterns() { //gti:add
	sels := fn.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := AsNode(sels[i].This())
		if sn != nil && sn.IsExternal() {
			sn.FRoot.RemoveExtFile(string(sn.FPath))
			sn.CloseBuf()
			sn.Delete(true)
		}
	}
}

// CloseBuf closes the file in its buffer if it is open.
// returns true if closed.
func (fn *Node) CloseBuf() bool {
	if fn.Buf == nil {
		return false
	}
	fn.Buf.Close(nil)
	fn.Buf = nil
	return true
}

// RelPath returns the relative path from node for given full path
func (fn *Node) RelPath(fpath gi.FileName) string {
	return dirs.RelFilePath(string(fpath), string(fn.FPath))
}

// DirsTo opens all the directories above the given filename, and returns the node
// for element at given path (can be a file or directory itself -- not opened -- just returned)
func (fn *Node) DirsTo(path string) (*Node, error) {
	pth, err := filepath.Abs(path)
	if err != nil {
		log.Printf("giv.Node DirsTo path %v could not be turned into an absolute path: %v\n", path, err)
		return nil, err
	}
	rpath := fn.RelPath(gi.FileName(pth))
	if rpath == "." {
		return fn, nil
	}
	dirs := strings.Split(rpath, string(filepath.Separator))
	cfn := fn
	sz := len(dirs)
	for i := 0; i < sz; i++ {
		dr := dirs[i]
		sfni, err := cfn.ChildByNameTry(dr, 0)
		if err != nil {
			if i == sz-1 { // ok for terminal -- might not exist yet
				return cfn, nil
			} else {
				err = fmt.Errorf("giv.Node could not find node %v in: %v", dr, cfn.FPath)
				// slog.Error(err.Error())
				return nil, err
			}
		}
		sfn := AsNode(sfni)
		if sfn.IsDir() || i == sz-1 {
			if i < sz-1 && !sfn.IsOpen() {
				sfn.OpenDir()
				sfn.UpdateNode()
			} else {
				cfn = sfn
			}
		} else {
			err := fmt.Errorf("giv.Node non-terminal node %v is not a directory in: %v", dr, cfn.FPath)
			slog.Error(err.Error())
			return nil, err
		}
		cfn = sfn
	}
	return cfn, nil
}

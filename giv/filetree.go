// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

// FileTree is the root of a tree representing files in a given directory (and
// subdirectories thereof), and has some overall management state for how to
// view things.  The FileTree can be viewed by a TreeView to provide a GUI
// interface into it.
type FileTree struct {
	FileNode
	OpenDirs  OpenDirMap `desc:"records which directories within the tree (encoded using paths relative to root) are open (i.e., have been opened by the user) -- can persist this to restore prior view of a tree"`
	DirsOnTop bool       `desc:"if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted"`
}

var KiT_FileTree = kit.Types.AddType(&FileTree{}, FileTreeProps)

var FileTreeProps = ki.Props{}

// OpenPath opens a filetree at given directory path -- reads all the files at
// given path into this tree -- uses config children to preserve extra info
// already stored about files.  Only paths listed in OpenDirs will be opened.
func (ft *FileTree) OpenPath(path string) {
	ft.FRoot = ft // we are our own root..
	ft.OpenDirs.ClearFlags()
	ft.ReadDir(path)
}

// UpdateNewFile should be called with path to a new file that has just been
// created -- will update view to show that file.
func (ft *FileTree) UpdateNewFile(filename gi.FileName) {
	fpath, _ := filepath.Split(string(filename))
	fpath = filepath.Clean(fpath)
	if fn, ok := ft.FindFile(string(filename)); ok {
		fn.UpdateNode()
	} else if fn, ok := ft.FindFile(fpath); ok {
		fn.UpdateNode()
	}
}

// RelPath returns the relative path from root for given full path
func (ft *FileTree) RelPath(fpath gi.FileName) string {
	rpath, err := filepath.Rel(string(ft.FPath), string(fpath))
	if err != nil {
		log.Printf("giv.FileTree RelPath error: %v\n", err.Error())
	}
	return rpath
}

// IsDirOpen returns true if given directory path is open (i.e., has been
// opened in the view)
func (ft *FileTree) IsDirOpen(fpath gi.FileName) bool {
	return ft.OpenDirs.IsOpen(ft.RelPath(fpath))
}

// SetDirOpen sets the given directory path to be open
func (ft *FileTree) SetDirOpen(fpath gi.FileName) {
	ft.OpenDirs.SetOpen(ft.RelPath(fpath))
}

// SetDirClosed sets the given directory path to be closed
func (ft *FileTree) SetDirClosed(fpath gi.FileName) {
	ft.OpenDirs.SetClosed(ft.RelPath(fpath))
}

//////////////////////////////////////////////////////////////////////////////
//    FileNode

// FileNode represents a file in the file system -- the name of the node is
// the name of the file.  Folders have children containing further nodes.
type FileNode struct {
	ki.Node
	Ic      gi.IconName `desc:"icon for this file"`
	FPath   gi.FileName `desc:"full path to this file"`
	Size    FileSize    `desc:"size of the file in bytes"`
	Kind    string      `width:"20" max-width:"20" desc:"type of file / directory -- including MIME type"`
	Mode    os.FileMode `desc:"file mode bits"`
	ModTime FileTime    `desc:"time that contents (only) were last modified"`
	Buf     *TextBuf    `json:"-" xml:"-" desc:"file buffer for editing this file"`
	FRoot   *FileTree   `json:"-" xml:"-" desc:"root of the tree -- has global state"`
}

var KiT_FileNode = kit.Types.AddType(&FileNode{}, nil)

func init() {
	kit.Types.SetProps(KiT_FileNode, FileNodeProps)
}

// IsDir returns true if file is a directory (folder)
func (fn *FileNode) IsDir() bool {
	return fn.Kind == "Folder"
}

// IsSymLink returns true if file is a symlink
func (fn *FileNode) IsSymLink() bool {
	return bitflag.Has(fn.Flag, int(FileNodeOpen))
}

// IsOpen returns true if file is flagged as open
func (fn *FileNode) IsOpen() bool {
	return bitflag.Has(fn.Flag, int(FileNodeOpen))
}

// SetOpen sets the open flag
func (fn *FileNode) SetOpen() {
	bitflag.Set(&fn.Flag, int(FileNodeOpen))
}

// ReadDir reads all the files at given directory into this directory node --
// uses config children to preserve extra info already stored about files. The
// root node represents the directory at the given path.
func (fn *FileNode) ReadDir(path string) {
	_, fnm := filepath.Split(path)
	fn.SetName(fnm)
	fn.FPath = gi.FileName(filepath.Clean(path))
	fn.SetOpen()

	typ := fn.NodeType()
	config := fn.ConfigOfFiles(path)
	mods, updt := fn.ConfigChildren(config, true) // unique names
	// always go through kids, regardless of mods
	for _, sfk := range fn.Kids {
		sf := sfk.Embed(KiT_FileNode).(*FileNode)
		sf.FRoot = fn.FRoot
		sf.SetChildType(typ) // propagate
		fp := filepath.Join(path, sf.Nm)
		sf.SetNodePath(fp)
	}
	if mods {
		fn.UpdateEnd(updt)
	}
}

// NodeType returns the type of nodes to create -- set ChildType property on
// NodeTree to seed this -- otherwise always FileNode
func (fn *FileNode) NodeType() reflect.Type {
	if ntp, ok := fn.Prop("ChildType"); ok {
		return ntp.(reflect.Type)
	}
	return KiT_FileNode
}

// ConfigOfFiles returns a type-and-name list for configuring nodes based on
// files immediately within given path
func (fn *FileNode) ConfigOfFiles(path string) kit.TypeAndNameList {
	config1 := kit.TypeAndNameList{}
	config2 := kit.TypeAndNameList{}
	typ := fn.NodeType()
	filepath.Walk(path, func(pth string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("gide.FileNode ConfigFilesIn Path %q: Error: %v", path, err)
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
	if fn.FRoot.DirsOnTop {
		for _, tn := range config2 {
			config1 = append(config1, tn)
		}
	}
	return config1
}

// SetNodePath sets the path for given node and updates it based on associated file
func (fn *FileNode) SetNodePath(path string) error {
	fn.FPath = gi.FileName(filepath.Clean(path))
	return fn.UpdateNode()
}

// UpdateNode updates information in node based on its associated file in FPath
func (fn *FileNode) UpdateNode() error {
	path := string(fn.FPath)
	info, err := os.Lstat(path)
	if err != nil {
		emsg := fmt.Errorf("gide.FileNode UpdateNode Path %q: Error: %v", path, err)
		log.Println(emsg)
		return emsg
	}
	fn.Size = FileSize(info.Size())
	fn.Mode = info.Mode()
	fn.ModTime = FileTime(info.ModTime())

	if info.IsDir() {
		fn.Kind = "Folder"
	} else {
		ext := filepath.Ext(fn.Nm)
		fn.Kind = mime.TypeByExtension(ext)
		fn.Kind = strings.TrimPrefix(fn.Kind, "application/") // long and unnec
	}
	fn.Ic = FileKindToIcon(fn.Kind, fn.Nm)

	if fn.IsDir() {
		if fn.FRoot.IsDirOpen(fn.FPath) {
			fn.ReadDir(path) // keep going down..
		}
	}
	return nil
}

// OpenDir opens given directory node
func (fn *FileNode) OpenDir() {
	fn.FRoot.SetDirOpen(fn.FPath)
	fn.UpdateNode()
}

// CloseDir closes given directory node -- updates memory state
func (fn *FileNode) CloseDir() {
	fn.FRoot.SetDirClosed(fn.FPath)
	// todo: do anything with open files within directory??
}

// OpenBuf opens the file in its buffer
func (fn *FileNode) OpenBuf() error {
	if fn.IsDir() {
		err := fmt.Errorf("gide.FileNode cannot open directory in editor: %v", fn.FPath)
		log.Println(err.Error())
		return err
	}
	fn.Buf = &TextBuf{}
	fn.Buf.InitName(fn.Buf, fn.Nm)
	return fn.Buf.Open(fn.FPath)
}

// FindFile finds first node representing given file (false if not found) --
// looks for full path names that have the given string as their suffix, so
// you can include as much of the path (including whole thing) as is relevant
// to disambiguate.  See FilesMatching for a list of files that match a given
// string.
func (fn *FileNode) FindFile(fnm string) (*FileNode, bool) {
	var ffn *FileNode
	found := false
	fn.FuncDownMeFirst(0, fn, func(k ki.Ki, level int, d interface{}) bool {
		sfn := k.Embed(KiT_FileNode).(*FileNode)
		if strings.HasSuffix(string(sfn.FPath), fnm) {
			ffn = sfn
			found = true
			return false
		}
		return true
	})
	return ffn, found
}

// FilesMatching returns list of all nodes whose file name contains given
// string (no regexp) -- ignoreCase transforms everything into lowercase
func (fn *FileNode) FilesMatching(match string, ignoreCase bool) []*FileNode {
	mls := make([]*FileNode, 0)
	if ignoreCase {
		match = strings.ToLower(match)
	}
	fn.FuncDownMeFirst(0, fn, func(k ki.Ki, level int, d interface{}) bool {
		sfn := k.Embed(KiT_FileNode).(*FileNode)
		if ignoreCase {
			nm := strings.ToLower(sfn.Nm)
			if strings.Contains(nm, match) {
				mls = append(mls, sfn)
			}
		} else {
			if strings.Contains(sfn.Nm, match) {
				mls = append(mls, sfn)
			}
		}
		return true
	})
	return mls
}

// FileNodeNameCount is used to report counts of different string-based things
// in the file tree
type FileNodeNameCount struct {
	Name  string
	Count int
}

// FileExtCounts returns a count of all the different file extensions, sorted
// from highest to lowest
func (fn *FileNode) FileExtCounts() []FileNodeNameCount {
	cmap := make(map[string]int, 20)
	fn.FuncDownMeFirst(0, fn, func(k ki.Ki, level int, d interface{}) bool {
		sfn := k.Embed(KiT_FileNode).(*FileNode)
		ext := strings.ToLower(filepath.Ext(sfn.Nm))
		if ec, has := cmap[ext]; has {
			cmap[ext] = ec + 1
		} else {
			cmap[ext] = 1
		}
		return true
	})
	ecs := make([]FileNodeNameCount, len(cmap))
	idx := 0
	for key, val := range cmap {
		ecs[idx] = FileNodeNameCount{key, val}
		idx++
	}
	sort.Slice(ecs, func(i, j int) bool {
		return ecs[i].Count > ecs[j].Count
	})
	return ecs
}

// DuplicateFile creates a copy of given file -- only works for regular files, not directories
func (fn *FileNode) DuplicateFile() {
	if fn.IsDir() {
		log.Printf("Duplicate: cannot copy directories\n")
		return
	}
	path := string(fn.FPath)
	ext := filepath.Ext(path)
	noext := strings.TrimSuffix(path, ext)
	dst := noext + "_Copy" + ext
	CopyFile(dst, path, fn.Mode)
	if fn.Par != nil {
		fnp := fn.Par.Embed(KiT_FileNode).(*FileNode)
		fnp.UpdateNode()
	}
}

// DeleteFile deletes this file
func (fn *FileNode) DeleteFile() {
	if fn.IsDir() {
		log.Printf("FileNode Delete -- cannot delete directories!\n")
		return
	}
	path := string(fn.FPath)
	os.Remove(path)
	fn.Delete(true) // we're done
}

// RenameFile renames file to new name
func (fn *FileNode) RenameFile(newpath string) {
	if newpath == "" {
		log.Printf("FileNode Rename: new name is empty!\n")
		return
	}
	path := string(fn.FPath)
	if newpath == path {
		return
	}
	ndir, nfn := filepath.Split(newpath)
	if ndir == "" {
		if nfn == fn.Nm {
			return
		}
		dir, _ := filepath.Split(path)
		newpath = filepath.Join(dir, newpath)
	}
	os.Rename(path, newpath)
	fn.FPath = gi.FileName(filepath.Clean(newpath))
	fn.SetName(nfn)
	fn.UpdateSig()
}

// FileNodeFlags define bitflags for FileNode state -- these extend ki.Flags
// and storage is an int64
type FileNodeFlags int64

const (
	// FileNodeOpen means file is open -- for directories, this means that
	// sub-files should be / have been loaded -- for files, means that they
	// have been opened e.g., for editing
	FileNodeOpen FileNodeFlags = FileNodeFlags(ki.FlagsN) + iota

	// FileNodeSymLink indicates that file is a symbolic link -- file info is
	// all for the target of the symlink
	FileNodeSymLink

	FileNodeFlagsN
)

//go:generate stringer -type=FileNodeFlags

var KiT_FileNodeFlags = kit.Enums.AddEnum(FileNodeFlagsN, true, nil) // true = bitflags

var FileNodeProps = ki.Props{
	"CtxtMenu": ki.PropSlice{
		{"DuplicateFile", ki.Props{
			"label": "Duplicate",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(KiT_FileNode).(*FileNode)
				act.SetInactiveStateUpdt(fn.IsDir())
			},
		}},
		{"DeleteFile", ki.Props{
			"label":   "Delete",
			"desc":    "Ok to delete this file?  This is not undoable and is not moving to trash / recycle bin",
			"confirm": true,
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(KiT_FileNode).(*FileNode)
				act.SetInactiveStateUpdt(fn.IsDir())
			},
		}},
		{"RenameFile", ki.Props{
			"label": "Rename",
			"desc":  "Rename file to new file name",
			"Args": ki.PropSlice{
				{"New Name", ki.Props{
					"default-field": "Name",
				}},
			},
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenDir", ki.Props{
			"desc": "open given directory to see files within",
			"updtfunc": func(fni interface{}, act *gi.Action) {
				fn := fni.(ki.Ki).Embed(KiT_FileNode).(*FileNode)
				act.SetActiveStateUpdt(fn.IsDir())
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////////////////
//    OpenDirMap

// OpenDirMap is a map for encoding directories that are open in the file
// tree.  The strings are typically relative paths.  The bool value is used to
// mark active paths and inactive (unmarked) ones can be removed.
type OpenDirMap map[string]bool

// Init initializes the map
func (dm *OpenDirMap) Init() {
	if *dm == nil {
		*dm = make(OpenDirMap, 1000)
	}
}

// IsOpen returns true if path is listed on the open map
func (dm *OpenDirMap) IsOpen(path string) bool {
	dm.Init()
	if _, ok := (*dm)[path]; ok {
		(*dm)[path] = true // mark
		return true
	}
	return false
}

// SetOpen adds the given path to the open map
func (dm *OpenDirMap) SetOpen(path string) {
	dm.Init()
	(*dm)[path] = true
}

// SetClosed removes given path from the open map
func (dm *OpenDirMap) SetClosed(path string) {
	dm.Init()
	delete(*dm, path)
}

// ClearFlags sets all the bool flags to false -- do this prior to traversing
// full set of active paths -- can then call RemoveStale to get rid of unused paths
func (dm *OpenDirMap) ClearFlags() {
	dm.Init()
	for key, _ := range *dm {
		(*dm)[key] = false
	}
}

// RemoveStale removes all entries with a bool = false value indicating that
// they have not been accessed since ClearFlags was called.
func (dm *OpenDirMap) RemoveStale() {
	dm.Init()
	for key, val := range *dm {
		if !val {
			delete(*dm, key)
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
//    FileTreeView

// FileTreeView is a TreeView that knows how to operate on FileNode nodes
type FileTreeView struct {
	TreeView
}

var KiT_FileTreeView = kit.Types.AddType(&FileTreeView{}, TreeViewProps)

var fnFolderProps = ki.Props{
	"icon":     "folder-open",
	"icon-off": "folder",
}

func (tv *FileTreeView) Style2D() {
	fn := tv.SrcNode.Ptr.Embed(KiT_FileNode).(*FileNode)
	if fn.IsDir() {
		if fn.IsOpen() {
			tv.Icon = gi.IconName("")
		} else {
			tv.Icon = gi.IconName("folder")
		}
		tv.SetProp("#branch", fnFolderProps)
	} else {
		tv.Icon = fn.Ic
	}
	tv.StyleTreeView()
	tv.LayData.SetFromStyle(&tv.Sty.Layout) // also does reset
}

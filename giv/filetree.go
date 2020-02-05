// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"errors"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/Masterminds/vcs"
	"github.com/goki/gi/gi"
	"github.com/goki/gi/histyle"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/gi/vci"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/filecat"
)

// FileTree is the root of a tree representing files in a given directory (and
// subdirectories thereof), and has some overall management state for how to
// view things.  The FileTree can be viewed by a TreeView to provide a GUI
// interface into it.
type FileTree struct {
	FileNode
	OpenDirs  OpenDirMap   `desc:"records which directories within the tree (encoded using paths relative to root) are open (i.e., have been opened by the user) -- can persist this to restore prior view of a tree"`
	DirsOnTop bool         `desc:"if true, then all directories are placed at the top of the tree view -- otherwise everything is alpha sorted"`
	NodeType  reflect.Type `view:"-" json:"-" xml:"-" desc:"type of node to create -- defaults to giv.FileNode but can use custom node types"`
}

var KiT_FileTree = kit.Types.AddType(&FileTree{}, FileTreeProps)

var FileTreeProps = ki.Props{
	"EnumType:Flag": KiT_FileNodeFlags,
}

func (ft *FileTree) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*FileTree)
	ft.FileNode.CopyFieldsFrom(&fr.FileNode)
	ft.DirsOnTop = fr.DirsOnTop
	ft.NodeType = fr.NodeType
}

// OpenPath opens a filetree at given directory path -- reads all the files at
// given path into this tree -- uses config children to preserve extra info
// already stored about files.  Only paths listed in OpenDirs will be opened.
func (ft *FileTree) OpenPath(path string) {
	ft.FRoot = ft // we are our own root..
	if ft.NodeType == nil {
		ft.NodeType = KiT_FileNode
	}
	ft.OpenDirs.ClearFlags()
	ft.ReadDir(path)
}

// UpdateNewFile should be called with path to a new file that has just been
// created -- will update view to show that file, and if that file doesn't
// exist, it updates the directory containing that file
func (ft *FileTree) UpdateNewFile(filename string) {
	ft.OpenDirsTo(filename)
	fpath, _ := filepath.Split(filename)
	fpath = filepath.Clean(fpath)
	if fn, ok := ft.FindFile(filename); ok {
		// fmt.Printf("updating node for file: %v\n", filename)
		fn.UpdateNode()
	} else if fn, ok := ft.FindFile(fpath); ok {
		// fmt.Printf("updating node for path: %v\n", fpath)
		fn.UpdateNode()
	} else {
		log.Printf("giv.FileTree UpdateNewFile: no node found for path to update: %v\n", filename)
	}
}

// IsDirOpen returns true if given directory path is open (i.e., has been
// opened in the view)
func (ft *FileTree) IsDirOpen(fpath gi.FileName) bool {
	if fpath == ft.FPath { // we are always open
		return true
	}
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

// FileNodeHiStyle is the default style for syntax highlighting to use for
// file node buffers
var FileNodeHiStyle = histyle.StyleDefault

// FileNode represents a file in the file system -- the name of the node is
// the name of the file.  Folders have children containing further nodes.
type FileNode struct {
	ki.Node
	FPath     gi.FileName `json:"-" xml:"-" copy:"-" desc:"full path to this file"`
	Info      FileInfo    `json:"-" xml:"-" copy:"-" desc:"full standard file info about this file"`
	Buf       *TextBuf    `json:"-" xml:"-" copy:"-" desc:"file buffer for editing this file"`
	FRoot     *FileTree   `json:"-" xml:"-" copy:"-" desc:"root of the tree -- has global state"`
	DirRepo   vci.Repo    `json:"-" xml:"-" copy:"-" desc:"version control system repository for this directory, only non-nil if this is the highest-level directory in the tree under vcs control"`
	RepoFiles vci.Files   `json:"-" xml:"-" copy:"-" desc:"version control system repository file status -- only valid during ReadDir"`
}

var KiT_FileNode = kit.Types.AddType(&FileNode{}, FileNodeProps)

func (fn *FileNode) CopyFieldsFrom(frm interface{}) {
	// note: not copying ki.Node as it doesn't have any copy fields
	// fr := frm.(*FileNode)
	// and indeed nothing here should be copied!
}

// IsDir returns true if file is a directory (folder)
func (fn *FileNode) IsDir() bool {
	return fn.Info.IsDir()
}

// IsSymLink returns true if file is a symlink
func (fn *FileNode) IsSymLink() bool {
	return fn.HasFlag(int(FileNodeSymLink))
}

// IsExec returns true if file is an executable file
func (fn *FileNode) IsExec() bool {
	return fn.Info.IsExec()
}

// IsOpen returns true if file is flagged as open
func (fn *FileNode) IsOpen() bool {
	return fn.HasFlag(int(FileNodeOpen))
}

// SetOpen sets the open flag
func (fn *FileNode) SetOpen() {
	fn.SetFlag(int(FileNodeOpen))
}

// SetClosed clears the open flag
func (fn *FileNode) SetClosed() {
	fn.ClearFlag(int(FileNodeOpen))
}

// IsChanged returns true if the file is open and has been changed (edited) since last save
func (fn *FileNode) IsChanged() bool {
	if fn.Buf != nil && fn.Buf.IsChanged() {
		return true
	}
	return false
}

// IsAutoSave returns true if file is an auto-save file (starts and ends with #)
func (fn *FileNode) IsAutoSave() bool {
	if strings.HasPrefix(fn.Info.Name, "#") && strings.HasSuffix(fn.Info.Name, "#") {
		return true
	}
	return false
}

// MyRelPath returns the relative path from root for this node
func (fn *FileNode) MyRelPath() string {
	rpath, err := filepath.Rel(string(fn.FRoot.FPath), string(fn.FPath))
	if err != nil {
		log.Printf("giv.FileNode RelPath error: %v\n", err.Error())
	}
	return rpath
}

// ReadDir reads all the files at given directory into this directory node --
// uses config children to preserve extra info already stored about files. The
// root node represents the directory at the given path.  Returns os.Stat
// error if path cannot be accessed.
func (fn *FileNode) ReadDir(path string) error {
	_, fnm := filepath.Split(path)
	fn.SetName(fnm)
	pth, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	fn.FPath = gi.FileName(pth)
	err = fn.Info.InitFile(string(fn.FPath))
	if err != nil {
		log.Printf("giv.FileTree: could not read directory: %v err: %v\n", fn.FPath, err)
		return err
	}

	fn.UpdateDir()
	return nil
}

// UpdateDir updates the directory and all the nodes under it
func (fn *FileNode) UpdateDir() {
	path := string(fn.FPath)
	// fmt.Printf("path: %v  node: %v\n", path, fn.PathUnique())
	var err error
	repo, rnode := fn.Repo()
	if repo == nil {
		rtyp := vci.DetectRepo(path)
		if rtyp != vcs.NoVCS {
			repo, err = vci.NewRepo("origin", path)
			if err == nil {
				fn.DirRepo = repo
				fn.UpdateRepoFiles()
				rnode = fn
			}
		}
	}

	fn.SetOpen()
	config := fn.ConfigOfFiles(path)
	mods, updt := fn.ConfigChildren(config, ki.NonUniqueNames) // NOT unique names
	if mods {
		// fmt.Printf("got mods: %v\n", path)
	}
	// always go through kids, regardless of mods
	for _, sfk := range fn.Kids {
		sf := sfk.Embed(KiT_FileNode).(*FileNode)
		sf.FRoot = fn.FRoot
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
		fn.UpdateEnd(updt)
	}
}

// ConfigOfFiles returns a type-and-name list for configuring nodes based on
// files immediately within given path
func (fn *FileNode) ConfigOfFiles(path string) kit.TypeAndNameList {
	config1 := kit.TypeAndNameList{}
	config2 := kit.TypeAndNameList{}
	typ := fn.FRoot.NodeType
	filepath.Walk(path, func(pth string, info os.FileInfo, err error) error {
		if err != nil {
			emsg := fmt.Sprintf("giv.FileNode ConfigFilesIn Path %q: Error: %v", path, err)
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
	pth, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	fn.FPath = gi.FileName(pth)
	err = fn.InitFileInfo()
	if err != nil {
		return err
	}
	if fn.IsDir() {
		if fn.FRoot.IsDirOpen(fn.FPath) {
			fn.ReadDir(string(fn.FPath)) // keep going down..
		}
	}
	return nil
}

// InitFileInfo initializes file info
func (fn *FileNode) InitFileInfo() error {
	err := fn.Info.InitFile(string(fn.FPath))
	if err != nil {
		emsg := fmt.Errorf("giv.FileNode InitFileInfo Path %q: Error: %v", fn.FPath, err)
		log.Println(emsg)
		return emsg
	}
	return nil
}

// UpdateNode updates information in node based on its associated file in FPath.
// This is intended to be called ad-hoc for individual nodes that might need
// updating -- use ReadDir for mass updates as it is more efficient.
func (fn *FileNode) UpdateNode() error {
	err := fn.InitFileInfo()
	if err != nil {
		return err
	}
	if fn.IsDir() {
		if fn.FRoot.IsDirOpen(fn.FPath) {
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
		fn.UpdateSig()
		fn.FRoot.UpdateSig()
	}
	return nil
}

// OpenDir opens given directory node
func (fn *FileNode) OpenDir() {
	fn.SetOpen()
	fn.FRoot.SetDirOpen(fn.FPath)
	fn.UpdateNode()
}

// CloseDir closes given directory node -- updates memory state
func (fn *FileNode) CloseDir() {
	fn.SetClosed()
	fn.FRoot.SetDirClosed(fn.FPath)
	// todo: do anything with open files within directory??
}

// OpenBuf opens the file in its buffer if it is not already open.
// returns true if file is newly opened
func (fn *FileNode) OpenBuf() (bool, error) {
	if fn.IsDir() {
		err := fmt.Errorf("giv.FileNode cannot open directory in editor: %v", fn.FPath)
		log.Println(err.Error())
		return false, err
	}
	if fn.Buf != nil {
		if fn.Buf.Filename == fn.FPath { // close resets filename
			return false, nil
		}
	} else {
		fn.Buf = &TextBuf{}
		fn.Buf.InitName(fn.Buf, fn.Nm)
		fn.Buf.AddFileNode(fn)
	}
	fn.Buf.Hi.Style = FileNodeHiStyle
	return true, fn.Buf.Open(fn.FPath)
}

// CloseBuf closes the file in its buffer if it is open -- returns true if closed
func (fn *FileNode) CloseBuf() bool {
	if fn.Buf == nil {
		return false
	}
	fn.Buf.Close(nil)
	fn.Buf = nil
	return true
}

// RelPath returns the relative path from node for given full path
func (fn *FileNode) RelPath(fpath gi.FileName) string {
	rpath, err := filepath.Rel(string(fn.FPath), string(fpath))
	if err != nil {
		log.Printf("giv.FileNode RelPath error: %v\n", err.Error())
		return ""
	}
	return rpath
}

// OpenDirsTo opens all the directories above the given filename, and returns the node
// for element at given path (can be a file or directory itself -- not opened -- just returned)
func (fn *FileNode) OpenDirsTo(path string) (*FileNode, error) {
	pth, err := filepath.Abs(path)
	if err != nil {
		log.Printf("giv.FileNode OpenDirsTo path %v could not be turned into an absolute path: %v\n", path, err)
		return nil, err
	}
	rpath := fn.RelPath(gi.FileName(pth))
	if rpath == "." {
		return fn, nil
	}
	if rpath == "" {
		err := fmt.Errorf("giv.FileNode OpenDirsTo path %v is not within file tree path: %v", pth, fn.FPath)
		log.Println(err)
		return nil, err
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
				err = fmt.Errorf("giv.FileNode could not find node %v in: %v", dr, cfn.FPath)
				log.Println(err)
				return nil, err
			}
		}
		sfn := sfni.Embed(KiT_FileNode).(*FileNode)
		if sfn.IsDir() || i == sz-1 {
			if i < sz-1 && !sfn.IsOpen() {
				sfn.OpenDir()
			} else {
				cfn = sfn
			}
		} else {
			err := fmt.Errorf("giv.FileNode non-terminal node %v is not a directory in: %v", dr, cfn.FPath)
			log.Println(err)
			return nil, err
		}
		cfn = sfn
	}
	return cfn, nil
}

// FindFile finds first node representing given file (false if not found) --
// looks for full path names that have the given string as their suffix, so
// you can include as much of the path (including whole thing) as is relevant
// to disambiguate.  See FilesMatching for a list of files that match a given
// string.
func (fn *FileNode) FindFile(fnm string) (*FileNode, bool) {
	if fnm == "" {
		return nil, false
	}
	fneff := fnm
	if fneff[:2] == ".." { // relative path -- get rid of it and just look for relative part
		dirs := strings.Split(fneff, string(filepath.Separator))
		for i, dr := range dirs {
			if dr != ".." {
				fneff = filepath.Join(dirs[i:]...)
				break
			}
		}
	}

	if strings.HasPrefix(fneff, string(fn.FPath)) { // full path
		ffn, err := fn.OpenDirsTo(fneff)
		if err == nil {
			return ffn, true
		}
		return nil, false
	}

	var ffn *FileNode
	found := false
	fn.FuncDownMeFirst(0, fn, func(k ki.Ki, level int, d interface{}) bool {
		sfn := k.Embed(KiT_FileNode).(*FileNode)
		if strings.HasSuffix(string(sfn.FPath), fneff) {
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
		ecs[idx] = FileNodeNameCount{Name: key, Count: val}
		idx++
	}
	sort.Slice(ecs, func(i, j int) bool {
		return ecs[i].Count > ecs[j].Count
	})
	return ecs
}

//////////////////////////////////////////////////////////////////////////////
//    File ops

// Duplicate creates a copy of given file -- only works for regular files, not
// directories
func (fn *FileNode) DuplicateFile() error {
	_, err := fn.Info.Duplicate()
	if err == nil && fn.Par != nil {
		fnp := fn.Par.Embed(KiT_FileNode).(*FileNode)
		fnp.UpdateNode()
	}
	return err
}

// DeleteFile deletes this file
func (fn *FileNode) DeleteFile() (err error) {
	repo, _ := fn.Repo()
	if !fn.Info.IsDir() && repo != nil && fn.Info.Vcs >= vci.Stored {
		// fmt.Printf("del repo: %v\n", fn.FPath)
		err = repo.Delete(string(fn.FPath))
	} else {
		// fmt.Printf("del raw: %v\n", fn.FPath)
		err = fn.Info.Delete()
	}
	if err == nil {
		fn.Delete(true)
	}
	return err
}

// RenameFile renames file to new name
func (fn *FileNode) RenameFile(newpath string) (err error) {
	orgpath := fn.FPath
	newpath, err = fn.Info.Rename(newpath)
	if len(newpath) == 0 || err != nil {
		return err
	}
	if fn.IsDir() {
		if fn.FRoot.IsDirOpen(orgpath) {
			fn.FRoot.SetDirOpen(gi.FileName(newpath))
		}
	}
	repo, _ := fn.Repo()
	if repo != nil && fn.Info.Vcs >= vci.Stored {
		err = repo.Move(string(orgpath), newpath)
	} else {
		err = os.Rename(string(orgpath), newpath)
	}
	if err == nil {
		err = fn.Info.InitFile(newpath)
	}
	if err == nil {
		fn.FPath = gi.FileName(fn.Info.Path)
		fn.SetName(fn.Info.Name)
	}
	fn.UpdateSig()
	fn.FRoot.UpdateDir() // need full update
	return err
}

// NewFile makes a new file in given selected directory node
func (fn *FileNode) NewFile(filename string, addToVcs bool) {
	ppath := string(fn.FPath)
	if !fn.IsDir() {
		ppath, _ = filepath.Split(ppath)
	}
	np := filepath.Join(ppath, filename)
	_, err := os.Create(np)
	if err != nil {
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Couldn't Make File", Prompt: fmt.Sprintf("Could not make new file at: %v, err: %v", np, err)}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	fn.FRoot.UpdateNewFile(np)
	if addToVcs {
		nfn, ok := fn.FRoot.FindFile(np)
		if ok && nfn.This() != fn.FRoot.This() {
			nfn.AddToVcs()
		}
	}
}

// NewFolder makes a new folder (directory) in given selected directory node
func (fn *FileNode) NewFolder(foldername string) {
	ppath := string(fn.FPath)
	if !fn.IsDir() {
		ppath, _ = filepath.Split(ppath)
	}
	np := filepath.Join(ppath, foldername)
	err := os.MkdirAll(np, 0775)
	if err != nil {
		emsg := fmt.Sprintf("giv.FileNode at: %q: Error: %v", ppath, err)
		gi.PromptDialog(nil, gi.DlgOpts{Title: "Couldn't Make Folder", Prompt: emsg}, gi.AddOk, gi.NoCancel, nil, nil)
		return
	}
	fn.FRoot.UpdateNewFile(ppath)
}

// CopyFileToDir copies given file path into node that is a directory.
// This does NOT check for overwriting -- that must be done at higher level!
func (fn *FileNode) CopyFileToDir(filename string, perm os.FileMode) {
	ppath := string(fn.FPath)
	_, sfn := filepath.Split(filename)
	tpath := filepath.Join(ppath, sfn)
	CopyFile(tpath, filename, perm)
	fn.FRoot.UpdateNewFile(ppath)
	ofn, ok := fn.FRoot.FindFile(filename)
	if ok && ofn.Info.Vcs >= vci.Stored {
		nfn, ok := fn.FRoot.FindFile(tpath)
		if ok && nfn.This() != fn.FRoot.This() {
			nfn.AddToVcs()
			nfn.UpdateNode()
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
//    File VCS ops

// Repo returns the version control repository associated with this file,
// and the node for the directory where the repo is based.
// Goes up the tree until a repository is found.
func (fn *FileNode) Repo() (vci.Repo, *FileNode) {
	var repo vci.Repo
	var rnode *FileNode
	fn.FuncUpParent(0, fn, func(k ki.Ki, level int, d interface{}) bool {
		sfni := k.Embed(KiT_FileNode)
		if sfni == nil {
			return false
		}
		sfn := sfni.(*FileNode)
		if sfn.DirRepo != nil {
			repo = sfn.DirRepo
			rnode = sfn
			return false
		}
		return true
	})
	return repo, rnode
}

func (fn *FileNode) UpdateRepoFiles() {
	if fn.DirRepo == nil {
		return
	}
	fn.RepoFiles, _ = fn.DirRepo.Files()
}

// AddToVcs adds file to version control
func (fn *FileNode) AddToVcs() {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	// fmt.Printf("adding to vcs: %v\n", fn.FPath)
	err := repo.Add(string(fn.FPath))
	if err == nil {
		fn.Info.Vcs = vci.Added
		fn.UpdateSig()
		fn.FRoot.UpdateSig()
		return
	}
	fmt.Println(err)
}

// DeleteFromVcs removes file from version control
func (fn *FileNode) DeleteFromVcs() {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	// fmt.Printf("deleting remote from vcs: %v\n", fn.FPath)
	err := repo.DeleteRemote(string(fn.FPath))
	if fn != nil && err == nil {
		fn.Info.Vcs = vci.Deleted
		fn.UpdateSig()
		fn.FRoot.UpdateSig()
		return
	}
	fmt.Println(err)
}

// CommitToVcs commits file changes to version control system
func (fn *FileNode) CommitToVcs(message string) (err error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	if fn.Info.Vcs == vci.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	err = repo.CommitFile(string(fn.FPath), message)
	if err == nil {
		fn.Info.Vcs = vci.Stored
		fn.UpdateSig()
		fn.FRoot.UpdateSig()
	}
	return err
}

// RevertVcs reverts file changes since last commit
func (fn *FileNode) RevertVcs() (err error) {
	repo, _ := fn.Repo()
	if repo == nil {
		return
	}
	if fn.Info.Vcs == vci.Untracked {
		return errors.New("file not in vcs repo: " + string(fn.FPath))
	}
	err = repo.RevertFile(string(fn.FPath))
	if err == nil {
		if fn.Info.Vcs == vci.Modified {
			fn.Info.Vcs = vci.Stored
		} else if fn.Info.Vcs == vci.Added {
			// do nothing - leave in "added" state
		}
		if fn.Buf != nil {
			fn.Buf.Revert()
		}
		fn.UpdateSig()
		fn.FRoot.UpdateSig()
	}
	return err
}

// FileNodeFlags define bitflags for FileNode state -- these extend ki.Flags
// and storage is an int64
type FileNodeFlags int64

//go:generate stringer -type=FileNodeFlags

var KiT_FileNodeFlags = kit.Enums.AddEnumExt(ki.KiT_Flags, FileNodeFlagsN, kit.BitFlag, nil)

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

var FileNodeProps = ki.Props{
	"EnumType:Flag": KiT_FileNodeFlags,
	"CallMethods": ki.PropSlice{
		{"RenameFile", ki.Props{
			"label": "Rename...",
			"desc":  "Rename file to new file name",
			"Args": ki.PropSlice{
				{"New Name", ki.Props{
					"width":         60,
					"default-field": "Nm",
				}},
			},
		}},
		{"NewFile", ki.Props{
			"label": "New File...",
			"desc":  "Create a new file in this folder",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"width": 60,
				}},
				{"Add To Version Control", ki.Props{}},
			},
		}},
		{"NewFolder", ki.Props{
			"label": "New Folder...",
			"desc":  "Create a new folder within this folder",
			"Args": ki.PropSlice{
				{"Folder Name", ki.Props{
					"width": 60,
				}},
			},
		}},
		{"CommitToVcs", ki.Props{
			"label": "Commit to Vcs...",
			"desc":  "Commit this file to version control",
			"Args": ki.PropSlice{
				{"Message", ki.Props{
					"width": 60,
				}},
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
// full set of active paths -- can then call DeleteStale to get rid of unused paths
func (dm *OpenDirMap) ClearFlags() {
	dm.Init()
	for key, _ := range *dm {
		(*dm)[key] = false
	}
}

// DeleteStale removes all entries with a bool = false value indicating that
// they have not been accessed since ClearFlags was called.
func (dm *OpenDirMap) DeleteStale() {
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

var KiT_FileTreeView = kit.Types.AddType(&FileTreeView{}, nil)

// AddNewFileTreeView adds a new filetreeview to given parent node, with given name.
func AddNewFileTreeView(parent ki.Ki, name string) *FileTreeView {
	tv := parent.AddNewChild(KiT_FileTreeView, name).(*FileTreeView)
	return tv
}

func init() {
	kit.Types.SetProps(KiT_FileTreeView, FileTreeViewProps)
}

// FileNode returns the SrcNode as a FileNode
func (ftv *FileTreeView) FileNode() *FileNode {
	if ftv.This() == nil {
		return nil
	}
	fni := ftv.SrcNode.Embed(KiT_FileNode)
	if fni == nil {
		return nil
	}
	return fni.(*FileNode)
}

func (ftv *FileTreeView) UpdateAllFiles() {
	fn := ftv.FileNode()
	if fn != nil {
		fn.FRoot.UpdateDir()
	}
}

func (ftv *FileTreeView) ConnectEvents2D() {
	ftv.FileTreeViewEvents()
}

func (ftv *FileTreeView) FileTreeViewEvents() {
	ftv.ConnectEvent(oswin.KeyChordEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		tvv := recv.Embed(KiT_FileTreeView).(*FileTreeView)
		kt := d.(*key.ChordEvent)
		tvv.KeyInput(kt)
	})
	ftv.ConnectEvent(oswin.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		de := d.(*dnd.Event)
		tvvi := recv.Embed(KiT_FileTreeView)
		if tvvi == nil {
			return
		}
		tvv := tvvi.(*FileTreeView)
		switch de.Action {
		case dnd.Start:
			tvv.DragNDropStart()
		case dnd.DropOnTarget:
			tvv.DragNDropTarget(de)
		case dnd.DropFmSource:
			tvv.This().(gi.DragNDropper).Dragged(de)
		case dnd.External:
			tvv.DragNDropExternal(de)
		}
	})
	ftv.ConnectEvent(oswin.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		de := d.(*dnd.FocusEvent)
		tvvi := recv.Embed(KiT_FileTreeView)
		if tvvi == nil {
			return
		}
		tvv := tvvi.(*FileTreeView)
		switch de.Action {
		case dnd.Enter:
			tvv.Viewport.Win.DNDSetCursor(de.Mod)
		case dnd.Exit:
			tvv.Viewport.Win.DNDNotCursor()
		case dnd.Hover:
			tvv.Open()
		}
	})
	if ftv.HasChildren() {
		if wb, ok := ftv.BranchPart(); ok {
			wb.ButtonSig.ConnectOnly(ftv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.ButtonToggled) {
					ftvv, _ := recv.Embed(KiT_FileTreeView).(*FileTreeView)
					ftvv.ToggleClose()
				}
			})
		}
	}
	if lbl, ok := ftv.LabelPart(); ok {
		// HiPri is needed to override label's native processing
		lbl.ConnectEvent(oswin.MouseEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
			lb, _ := recv.(*gi.Label)
			ftvvi := lb.Parent().Parent()
			if ftvvi == nil || ftvvi.This() == nil { // deleted
				return
			}
			ftvv := ftvvi.Embed(KiT_FileTreeView).(*FileTreeView)
			me := d.(*mouse.Event)
			switch me.Button {
			case mouse.Left:
				switch me.Action {
				case mouse.DoubleClick:
					ftvv.ToggleClose()
					me.SetProcessed()
				case mouse.Release:
					ftvv.SelectAction(me.SelectMode())
					me.SetProcessed()
				}
			case mouse.Right:
				if me.Action == mouse.Release {
					me.SetProcessed()
					ftvv.This().(gi.Node2D).ContextMenu()
				}
			}
		})
	}
}

func (ftv *FileTreeView) KeyInput(kt *key.ChordEvent) {
	if gi.KeyEventTrace {
		fmt.Printf("TreeView KeyInput: %v\n", ftv.PathUnique())
	}
	kf := gi.KeyFun(kt.Chord())
	selMode := mouse.SelectModeBits(kt.Modifiers)

	if selMode == mouse.SelectOne {
		if ftv.SelectMode() {
			selMode = mouse.ExtendContinuous
		}
	}

	// first all the keys that work for inactive and active
	if !ftv.IsInactive() && !kt.IsProcessed() {
		switch kf {
		case gi.KeyFunDelete:
			ftv.DeleteFiles()
			kt.SetProcessed()
			// todo: remove when gi issue 237 is resolved
		case gi.KeyFunBackspace:
			ftv.DeleteFiles()
			kt.SetProcessed()
		case gi.KeyFunDuplicate:
			ftv.DuplicateFiles()
			kt.SetProcessed()
		case gi.KeyFunInsert: // New File
			CallMethod(ftv, "NewFile", ftv.Viewport)
			kt.SetProcessed()
		case gi.KeyFunInsertAfter: // New Folder
			CallMethod(ftv, "NewFolder", ftv.Viewport)
			kt.SetProcessed()
		}
	}
	if !kt.IsProcessed() {
		ftv.TreeView.KeyInput(kt)
	}
}

// ShowFileInfo calls ViewFile on selected files
func (ftv *FileTreeView) ShowFileInfo() {
	sels := ftv.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i]
		fftv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
		fn := fftv.FileNode()
		if fn != nil {
			StructViewDialog(ftv.Viewport, &fn.Info, DlgOpts{Title: "File Info", Inactive: true}, nil, nil)
		}
	}
}

// DuplicateFiles calls DuplicateFile on any selected nodes
func (ftv *FileTreeView) DuplicateFiles() {
	sels := ftv.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i]
		ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
		fn := ftvv.FileNode()
		if fn != nil {
			fn.DuplicateFile()
		}
	}
}

// DeleteFilesImpl does the actual deletion, no prompts
func (ftv *FileTreeView) DeleteFilesImpl() {
	sels := ftv.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i]
		ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
		fn := ftvv.FileNode()
		if fn == nil {
			return
		}
		if fn.Info.IsDir() {
			openList := []string{}
			var fns []string
			fn.Info.FileNames(&fns)
			ft := fn.FRoot
			for _, filename := range fns {
				fn, ok := ft.FindFile(filename)
				if !ok {
					return
				}
				if fn.Buf != nil {
					openList = append(openList, filename)
				}
			}
			if len(openList) > 0 {
				for _, filename := range openList {
					fn, _ := ft.FindFile(filename)
					fn.CloseBuf()
				}
			}
			fn.DeleteFile()
		} else {
			if fn.Buf != nil {
				fn.CloseBuf()
			}
			fn.DeleteFile()
		}
	}
}

// DeleteFiles calls DeleteFile on any selected nodes. If any directory is selected
// all files and subdirectories are also deleted.
func (ftv *FileTreeView) DeleteFiles() {
	gi.ChoiceDialog(ftv.Viewport, gi.DlgOpts{Title: "Delete Files?",
		Prompt: "Ok to delete file(s)?  This is not undoable and files are not moving to trash / recycle bin. If any selections are directories all files and subdirectories will also be deleted."},
		[]string{"Delete Files", "Cancel"},
		ftv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			switch sig {
			case 0:
				ftv.DeleteFilesImpl()
			case 1:
				// do nothing
			}
		})
}

// RenameFiles calls RenameFile on any selected nodes
func (ftv *FileTreeView) RenameFiles() {
	sels := ftv.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i]
		ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
		fn := ftvv.FileNode()
		if fn != nil {
			CallMethod(fn, "RenameFile", ftv.Viewport)
		}
	}
}

// OpenDirs
func (ftv *FileTreeView) OpenDirs() {
	sels := ftv.SelectedViews()
	for i := len(sels) - 1; i >= 0; i-- {
		sn := sels[i]
		ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
		fn := ftvv.FileNode()
		if fn != nil {
			fn.OpenDir()
		}
	}
}

// NewFile makes a new file in given selected directory node
func (ftv *FileTreeView) NewFile(filename string, addToVcs bool) {
	sels := ftv.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := sels[sz-1]
	ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftvv.FileNode()
	if fn != nil {
		fn.NewFile(filename, addToVcs)
	}
}

// NewFolder makes a new file in given selected directory node
func (ftv *FileTreeView) NewFolder(foldername string) {
	sels := ftv.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := sels[sz-1]
	ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftvv.FileNode()
	if fn != nil {
		fn.NewFolder(foldername)
	}
}

// AddToVcs adds the file to version control system
func (ftv *FileTreeView) AddToVcs() {
	sels := ftv.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := sels[sz-1]
	ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftvv.FileNode()
	if fn != nil {
		fn.AddToVcs()
	}
}

// DeleteFromVcs removes the file from version control system
func (ftv *FileTreeView) DeleteFromVcs() {
	sels := ftv.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := sels[sz-1]
	ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftvv.FileNode()
	if fn != nil {
		fn.DeleteFromVcs()
	}
}

// CommitToVcs removes the file from version control system
func (ftv *FileTreeView) CommitToVcs() {
	sels := ftv.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := sels[sz-1]
	ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftvv.FileNode()
	if fn != nil {
		CallMethod(fn, "CommitToVcs", ftv.Viewport)
	}
}

// RevertVcs removes the file from version control system
func (ftv *FileTreeView) RevertVcs() {
	sels := ftv.SelectedViews()
	sz := len(sels)
	if sz == 0 { // shouldn't happen
		return
	}
	sn := sels[sz-1]
	ftvv := sn.Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftvv.FileNode()
	if fn != nil {
		fn.RevertVcs()
	}
}

///////////////////////////////////////////////////////////////////////////////
//   Clipboard

// MimeData adds mimedata for this node: a text/plain of the PathUnique,
// text/plain of filename, and text/
func (ftv *FileTreeView) MimeData(md *mimedata.Mimes) {
	sroot := ftv.RootView.SrcNode
	fn := ftv.SrcNode.Embed(KiT_FileNode).(*FileNode)
	path := string(fn.FPath)
	*md = append(*md, mimedata.NewTextData(fn.PathFromUnique(sroot)))
	*md = append(*md, mimedata.NewTextData(path))
	if int(fn.Info.Size) < gi.Prefs.Params.BigFileSize {
		in, err := os.Open(path)
		if err != nil {
			log.Println(err)
			return
		}
		b, err := ioutil.ReadAll(in)
		if err != nil {
			log.Println(err)
			return
		}
		fd := &mimedata.Data{fn.Info.Mime, b}
		*md = append(*md, fd)
	} else {
		*md = append(*md, mimedata.NewTextData("File exceeds BigFileSize"))
	}
}

// Cut copies to clip.Board and deletes selected items
// satisfies gi.Clipper interface and can be overridden by subtypes
func (ftv *FileTreeView) Cut() {
	if ftv.IsRootOrField("Cut") {
		return
	}
	ftv.Copy(false)
	// todo: in the future, move files somewhere temporary, then use those temps for paste..
	gi.PromptDialog(ftv.Viewport, gi.DlgOpts{Title: "Cut Not Supported", Prompt: "File names were copied to clipboard and can be pasted to copy elsewhere, but files are not deleted because contents of files are not placed on the clipboard and thus cannot be pasted as such.  Use Delete to delete files."}, gi.AddOk, gi.NoCancel, nil, nil)
}

// Paste pastes clipboard at given node
// satisfies gi.Clipper interface and can be overridden by subtypes
func (ftv *FileTreeView) Paste() {
	md := oswin.TheApp.ClipBoard(ftv.Viewport.Win.OSWin).Read([]string{filecat.TextPlain})
	if md != nil {
		ftv.PasteMime(md)
	}
}

// Drop pops up a menu to determine what specifically to do with dropped items
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (ftv *FileTreeView) Drop(md mimedata.Mimes, mod dnd.DropMods) {
	ftv.PasteMime(md)
}

// DropExternal is not handled by base case but could be in derived
func (ftv *FileTreeView) DropExternal(md mimedata.Mimes, mod dnd.DropMods) {
	ftv.PasteMime(md)
}

// PasteCheckExisting checks for existing files in target node directory if
// that is non-nil (otherwise just uses absolute path), and returns list of existing
// and node for last one if exists.
func (ftv *FileTreeView) PasteCheckExisting(tfn *FileNode, md mimedata.Mimes) ([]string, *FileNode) {
	sroot := ftv.RootView.SrcNode
	tpath := ""
	if tfn != nil {
		tpath = string(tfn.FPath)
	}
	intl := ftv.Viewport.Win.EventMgr.DNDIsInternalSrc()
	nf := len(md)
	if intl {
		nf /= 3
	}
	var sfn *FileNode
	var existing []string
	for i := 0; i < nf; i++ {
		var d *mimedata.Data
		if intl {
			d = md[i*3+1]
			npath := string(md[i*3].Data)
			sfni, err := sroot.FindPathUniqueTry(npath)
			if err == nil {
				sfn = sfni.Embed(KiT_FileNode).(*FileNode)
			}
		} else {
			d = md[i] // just a list
		}
		if d.Type != filecat.TextPlain {
			continue
		}
		path := string(d.Data)
		if strings.HasPrefix(path, "file://") {
			path = path[7:]
		}
		if tfn != nil {
			_, fnm := filepath.Split(path)
			path = filepath.Join(tpath, fnm)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			existing = append(existing, path)
		}
	}
	return existing, sfn
}

// PasteCopyFiles copies files in given data into given target directory
func (ftv *FileTreeView) PasteCopyFiles(tdir *FileNode, md mimedata.Mimes) {
	sroot := ftv.RootView.SrcNode
	intl := ftv.Viewport.Win.EventMgr.DNDIsInternalSrc()
	nf := len(md)
	if intl {
		nf /= 3
	}
	for i := 0; i < nf; i++ {
		var d *mimedata.Data
		mode := os.FileMode(0664)
		if intl {
			d = md[i*3+1]
			npath := string(md[i*3].Data)
			sfni, err := sroot.FindPathUniqueTry(npath)
			if err != nil {
				fmt.Println(err)
				continue
			}
			sfn := sfni.Embed(KiT_FileNode).(*FileNode)
			mode = sfn.Info.Mode
		} else {
			d = md[i] // just a list
		}
		if d.Type != filecat.TextPlain {
			continue
		}
		path := string(d.Data)
		if strings.HasPrefix(path, "file://") {
			path = path[7:]
		}
		tdir.CopyFileToDir(path, mode)
	}
}

// PasteMime applies a paste / drop of mime data onto this node
// always does a copy of files into / onto target
func (ftv *FileTreeView) PasteMime(md mimedata.Mimes) {
	tfn := ftv.FileNode()
	if tfn == nil {
		return
	}
	tupdt := ftv.RootView.UpdateStart()
	defer ftv.RootView.UpdateEnd(tupdt)
	tpath := string(tfn.FPath)
	if tfn.IsDir() {
		existing, _ := ftv.PasteCheckExisting(tfn, md)
		if len(existing) > 0 {
			gi.ChoiceDialog(nil, gi.DlgOpts{Title: "File(s) Exist in Target Dir, Overwrite?",
				Prompt: fmt.Sprintf("File(s): %v exist, do you want to overwrite?", existing)},
				[]string{"No, Cancel", "Yes, Overwrite"},
				ftv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					switch sig {
					case 0:
						ftv.DropCancel()
					case 1:
						ftv.PasteCopyFiles(tfn, md)
						ftv.DragNDropFinalizeDefMod()
					}
				})
		} else {
			ftv.PasteCopyFiles(tfn, md)
			ftv.DragNDropFinalizeDefMod()
		}
	} else { // dropping on top of existing file
		if len(md) > 3 {
			gi.PromptDialog(ftv.Viewport, gi.DlgOpts{Title: "Can Only Copy 1 File", Prompt: fmt.Sprintf("Only one file can be copied target file: %v -- currently have: %v", tfn.Name(), len(md)/3)}, gi.AddOk, gi.NoCancel, nil, nil)
			ftv.DropCancel()
			return
		}
		existing, sfn := ftv.PasteCheckExisting(nil, md)
		if len(existing) != 1 {
			return
		}
		path := existing[0]
		mode := os.FileMode(0664)
		if sfn != nil {
			mode = sfn.Info.Mode
		}
		gi.ChoiceDialog(nil, gi.DlgOpts{Title: "Overwrite?",
			Prompt: fmt.Sprintf("Are you sure you want to overwrite file: %v with: %v?", tpath, path)},
			[]string{"No, Cancel", "Yes, Overwrite"},
			ftv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
					ftv.DropCancel()
				case 1:
					CopyFile(tpath, path, mode)
					ftv.DragNDropFinalizeDefMod()
				}
			})
	}
}

// Dragged is called after target accepts the drop -- we just remove
// elements that were moved
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (ftv *FileTreeView) Dragged(de *dnd.Event) {
	// fmt.Printf("ftv dragged: %v\n", ftv.PathUnique())
	if de.Mod != dnd.DropMove {
		return
	}
	sroot := ftv.RootView.SrcNode
	tfn := ftv.FileNode()
	if tfn == nil {
		return
	}
	md := de.Data
	nf := len(md) / 3 // always internal
	for i := 0; i < nf; i++ {
		npath := string(md[i*3].Data)
		sfni, err := sroot.FindPathUniqueTry(npath)
		if err != nil {
			fmt.Println(err)
			continue
		}
		sfn := sfni.Embed(KiT_FileNode).(*FileNode)
		if sfn == nil {
			continue
		}
		// fmt.Printf("dnd deleting: %v  path: %v\n", sfn.PathUnique(), sfn.FPath)
		sfn.DeleteFile()
	}
}

// FileTreeInactiveDirFunc is an ActionUpdateFunc that inactivates action if node is a dir
var FileTreeInactiveDirFunc = ActionUpdateFunc(func(fni interface{}, act *gi.Action) {
	ftv := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftv.FileNode()
	if fn != nil {
		act.SetInactiveState(fn.IsDir())
	}
})

// FileTreeActiveDirFunc is an ActionUpdateFunc that activates action if node is a dir
var FileTreeActiveDirFunc = ActionUpdateFunc(func(fni interface{}, act *gi.Action) {
	ftv := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftv.FileNode()
	if fn != nil {
		act.SetActiveState(fn.IsDir())
	}
})

// FileTreeActiveNotInVcsFunc is an ActionUpdateFunc that inactivates action if node is not under version control
var FileTreeActiveNotInVcsFunc = ActionUpdateFunc(func(fni interface{}, act *gi.Action) {
	ftv := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftv.FileNode()
	if fn != nil {
		repo, _ := fn.Repo()
		if repo == nil || fn.IsDir() {
			act.SetActiveState((false))
			return
		}
		act.SetActiveState((fn.Info.Vcs == vci.Untracked))
	}
})

// FileTreeActiveInVcsFunc is an ActionUpdateFunc that activates action if node is under version control
var FileTreeActiveInVcsFunc = ActionUpdateFunc(func(fni interface{}, act *gi.Action) {
	ftv := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftv.FileNode()
	if fn != nil {
		repo, _ := fn.Repo()
		if repo == nil || fn.IsDir() {
			act.SetActiveState((false))
			return
		}
		act.SetActiveState((fn.Info.Vcs >= vci.Stored))
	}
})

// FileTreeActiveInVcsModifiedFunc is an ActionUpdateFunc that activates action if node is under version control
// and the file has been modified
var FileTreeActiveInVcsModifiedFunc = ActionUpdateFunc(func(fni interface{}, act *gi.Action) {
	ftv := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftv.FileNode()
	if fn != nil {
		repo, _ := fn.Repo()
		if repo == nil || fn.IsDir() {
			act.SetActiveState((false))
			return
		}
		act.SetActiveState((fn.Info.Vcs == vci.Modified || fn.Info.Vcs == vci.Added))
	}
})

// VcsGetRemoveLabelFunc gets the appropriate label for removing from version control
var VcsLabelFunc = LabelFunc(func(fni interface{}, act *gi.Action) string {
	ftv := fni.(ki.Ki).Embed(KiT_FileTreeView).(*FileTreeView)
	fn := ftv.FileNode()
	label := act.Text
	if fn != nil {
		repo, _ := fn.Repo()
		if repo != nil {
			label = strings.Replace(label, "Vcs", string(repo.Vcs()), 1)
		}
	}
	return label
})

var FileTreeViewProps = ki.Props{
	"EnumType:Flag":    KiT_TreeViewFlags,
	"indent":           units.NewCh(2),
	"spacing":          units.NewCh(.5),
	"border-width":     units.NewPx(0),
	"border-radius":    units.NewPx(0),
	"padding":          units.NewPx(0),
	"margin":           units.NewPx(1),
	"text-align":       gi.AlignLeft,
	"vertical-align":   gi.AlignTop,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": "inherit",
	"no-templates":     true,
	".exec": ki.Props{
		"font-weight": gi.WeightBold,
	},
	".open": ki.Props{
		"font-style": gi.FontItalic,
	},
	".untracked": ki.Props{
		"color": "#808080",
	},
	".modified": ki.Props{
		"color": "#4b7fd1",
	},
	".added": ki.Props{
		"color": "#008800",
	},
	".deleted": ki.Props{
		"color": "#ff4252",
	},
	".conflicted": ki.Props{
		"color": "#ce8020",
	},
	".updated": ki.Props{
		"color": "#008060",
	},
	"#icon": ki.Props{
		"width":   units.NewEm(1),
		"height":  units.NewEm(1),
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
		"fill":    &gi.Prefs.Colors.Icon,
		"stroke":  &gi.Prefs.Colors.Font,
	},
	"#branch": ki.Props{
		"icon":             "wedge-down",
		"icon-off":         "wedge-right",
		"margin":           units.NewPx(0),
		"padding":          units.NewPx(0),
		"background-color": color.Transparent,
		"max-width":        units.NewEm(.8),
		"max-height":       units.NewEm(.8),
	},
	"#space": ki.Props{
		"width": units.NewEm(.5),
	},
	"#label": ki.Props{
		"margin":    units.NewPx(0),
		"padding":   units.NewPx(0),
		"min-width": units.NewCh(16),
	},
	"#menu": ki.Props{
		"indicator": "none",
	},
	TreeViewSelectors[TreeViewActive]: ki.Props{},
	TreeViewSelectors[TreeViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
	TreeViewSelectors[TreeViewFocus]: ki.Props{
		"background-color": &gi.Prefs.Colors.Control,
	},
	"CtxtMenuActive": ki.PropSlice{
		{"ShowFileInfo", ki.Props{
			"label": "File Info",
		}},
		{"DuplicateFiles", ki.Props{
			"label":    "Duplicate",
			"updtfunc": FileTreeInactiveDirFunc,
			"shortcut": gi.KeyFunDuplicate,
		}},
		{"DeleteFiles", ki.Props{
			"label":    "Delete",
			"desc":     "Ok to delete file(s)?  This is not undoable and is not moving to trash / recycle bin",
			"shortcut": gi.KeyFunDelete,
		}},
		{"RenameFiles", ki.Props{
			"label": "Rename",
			"desc":  "Rename file to new file name",
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenDirs", ki.Props{
			"label":    "Open Dir",
			"desc":     "open given folder to see files within",
			"updtfunc": FileTreeActiveDirFunc,
		}},
		{"NewFile", ki.Props{
			"label":    "New File...",
			"desc":     "make a new file in this folder",
			"shortcut": gi.KeyFunInsert,
			"updtfunc": FileTreeActiveDirFunc,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"width": 60,
				}},
				{"Add To Version Control", ki.Props{}},
			},
		}},
		{"NewFolder", ki.Props{
			"label":    "New Folder...",
			"desc":     "make a new folder within this folder",
			"shortcut": gi.KeyFunInsertAfter,
			"updtfunc": FileTreeActiveDirFunc,
			"Args": ki.PropSlice{
				{"Folder Name", ki.Props{
					"width": 60,
				}},
			},
		}},
		{"sep-vcs", ki.BlankProp{}},
		{"AddToVcs", ki.Props{
			"desc":       "Add file to version control",
			"updtfunc":   FileTreeActiveNotInVcsFunc,
			"label-func": VcsLabelFunc,
		}},
		{"DeleteFromVcs", ki.Props{
			"desc":       "Delete file from version control",
			"updtfunc":   FileTreeActiveInVcsFunc,
			"label-func": VcsLabelFunc,
		}},
		{"CommitToVcs", ki.Props{
			"desc":       "Commit file to version control",
			"updtfunc":   FileTreeActiveInVcsModifiedFunc,
			"label-func": VcsLabelFunc,
		}},
		{"RevertVcs", ki.Props{
			"desc":       "Revert file to last commit",
			"updtfunc":   FileTreeActiveInVcsModifiedFunc,
			"label-func": VcsLabelFunc,
		}},
	},
}

var fnFolderProps = ki.Props{
	"icon":     "folder-open",
	"icon-off": "folder",
}

func (ft *FileTreeView) Style2D() {
	fn := ft.FileNode()
	ft.Class = ""
	if fn != nil {
		if fn.IsDir() {
			if fn.HasChildren() {
				ft.Icon = gi.IconName("")
			} else {
				ft.Icon = gi.IconName("folder")
			}
			ft.SetProp("#branch", fnFolderProps)
			ft.AddClass("folder")
		} else {
			ft.Icon = fn.Info.Ic
			if ft.Icon == "" || ft.Icon == "none" {
				ft.Icon = "blank"
			}
			if fn.IsExec() {
				ft.AddClass("exec")
			}
			if fn.IsOpen() {
				ft.AddClass("open")
			}
			switch fn.Info.Vcs {
			case vci.Untracked:
				ft.AddClass("untracked")
			case vci.Stored:
				ft.AddClass("stored")
			case vci.Modified:
				ft.AddClass("modified")
			case vci.Added:
				ft.AddClass("added")
			case vci.Deleted:
				ft.AddClass("deleted")
			case vci.Conflicted:
				ft.AddClass("conflicted")
			case vci.Updated:
				ft.AddClass("updated")
			}
		}
		ft.StyleTreeView()
		ft.LayData.SetFromStyle(&ft.Sty.Layout) // also does reset
	}
}

// FileNodeBufSigRecv receives a signal from the buffer and updates view accordingly
func FileNodeBufSigRecv(rvwki, sbufki ki.Ki, sig int64, data interface{}) {
	fn := rvwki.Embed(KiT_FileNode).(*FileNode)
	switch TextBufSignals(sig) {
	case TextBufDone, TextBufInsert, TextBufDelete:
		if fn.Info.Vcs == vci.Stored {
			fn.Info.Vcs = vci.Modified
		}
	}
}

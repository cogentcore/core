// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

/*
// MimeData adds mimedata for this node: a text/plain of the Path,
// text/plain of filename, and text/
func (fn *Node) MimeData(md *mimedata.Mimes) {
	sroot := fn.RootView.SrcNode
	fn := AsNode(fn.SrcNode)
	path := string(fn.FPath)
	punq := fn.PathFrom(sroot)
	*md = append(*md, mimedata.NewTextData(punq))
	*md = append(*md, mimedata.NewTextData(path))
	if int(fn.Info.Size) < gi.Prefs.Params.BigFileSize {
		in, err := os.Open(path)
		if err != nil {
			slog.Error(err.Error())
			return
		}
		b, err := ioutil.ReadAll(in)
		if err != nil {
			slog.Error(err.Error())
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
func (fn *Node) Cut() {
	if fn.IsRoot("Cut") {
		return
	}
	fn.Copy(false)
	// todo: in the future, move files somewhere temporary, then use those temps for paste..
	gi.PromptDialog(fn, gi.DlgOpts{Title: "Cut Not Known", Prompt: "File names were copied to clipboard and can be pasted to copy elsewhere, but files are not deleted because contents of files are not placed on the clipboard and thus cannot be pasted as such.  Use Delete to delete files.", Ok: true, Cancel: false}, nil)
}

// Paste pastes clipboard at given node
// satisfies gi.Clipper interface and can be overridden by subtypes
func (fn *Node) Paste() {
	md := fn.EventMgr().ClipBoard().Read([]string{fi.TextPlain})
	if md != nil {
		fn.PasteMime(md)
	}
}

// Drop pops up a menu to determine what specifically to do with dropped items
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (fn *Node) Drop(md mimedata.Mimes, mod events.DropMods) {
	fn.PasteMime(md)
}

// DropExternal is not handled by base case but could be in derived
func (fn *Node) DropExternal(md mimedata.Mimes, mod events.DropMods) {
	fn.PasteMime(md)
}

// PasteCheckExisting checks for existing files in target node directory if
// that is non-nil (otherwise just uses absolute path), and returns list of existing
// and node for last one if exists.
func (fn *Node) PasteCheckExisting(tfn *Node, md mimedata.Mimes) ([]string, *Node) {
	sroot := fn.RootView.SrcNode
	tpath := ""
	if tfn != nil {
		tpath = string(tfn.FPath)
	}
	// intl := ftv.EventMgr.DNDIsInternalSrc() // todo
	intl := false
	nf := len(md)
	if intl {
		nf /= 3
	}
	var sfn *Node
	var existing []string
	for i := 0; i < nf; i++ {
		var d *mimedata.Data
		if intl {
			d = md[i*3+1]
			npath := string(md[i*3].Data)
			sfni, err := sroot.FindPathTry(npath)
			if err == nil {
				sfn = AsNode(sfni)
			}
		} else {
			d = md[i] // just a list
		}
		if d.Type != fi.TextPlain {
			continue
		}
		path := string(d.Data)
		path = strings.TrimPrefix(path, "file://")
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
func (fn *Node) PasteCopyFiles(tdir *Node, md mimedata.Mimes) {
	sroot := fn.RootView.SrcNode
	// intl := ftv.EventMgr.DNDIsInternalSrc()
	intl := true
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
			sfni, err := sroot.FindPathTry(npath)
			if err != nil {
				fmt.Println(err)
				continue
			}
			sfn := AsNode(sfni)
			mode = sfn.Info.Mode
		} else {
			d = md[i] // just a list
		}
		if d.Type != fi.TextPlain {
			continue
		}
		path := string(d.Data)
		if strings.HasPrefix(path, "file://") {
			path = path[7:]
		}
		tdir.CopyFileToDir(path, mode)
	}
}

// PasteMimeCopyFilesCheck copies files into given directory node,
// first checking if any already exist -- if they exist, prompts.
func (fn *Node) PasteMimeCopyFilesCheck(tdir *Node, md mimedata.Mimes) {
	// existing, _ := ftv.PasteCheckExisting(tdir, md)
	// if len(existing) > 0 {
	// 	gi.ChoiceDialog(ftv, gi.DlgOpts{Title: "File(s) Exist in Target Dir, Overwrite?",
	// 		Prompt: fmt.Sprintf("File(s): %v exist, do you want to overwrite?", existing)},
	// 		[]string{"No, Cancel", "Yes, Overwrite"}, func(dlg *gi.Dialog) {
	// 			switch dlg.Data.(int) {
	// 			case 0:
	// 				ftv.DropCancel()
	// 			case 1:
	// 				ftv.PasteCopyFiles(tdir, md)
	// 				ftv.DragNDropFinalizeDefMod()
	// 			}
	// 		})
	// } else {
	// 	ftv.PasteCopyFiles(tdir, md)
	// 	ftv.DragNDropFinalizeDefMod()
	// }
}

// PasteMime applies a paste / drop of mime data onto this node
// always does a copy of files into / onto target
func (fn *Node) PasteMime(md mimedata.Mimes) {
	if len(md) == 0 {
		fn.DropCancel()
		return
	}
	tfn := fn.Node()
	if tfn == nil || tfn.IsExternal() {
		fn.DropCancel()
		return
	}
	tupdt := fn.RootView.UpdateStart()
	defer fn.RootView.UpdateEnd(tupdt)
	tpath := string(tfn.FPath)
	isdir := tfn.IsDir()
	if isdir {
		fn.PasteMimeCopyFilesCheck(tfn, md)
		return
	}
	if len(md) > 3 { // multiple files -- automatically goes into parent dir
		tdir := AsNode(tfn.Parent())
		fn.PasteMimeCopyFilesCheck(tdir, md)
		return
	}
	// single file dropped onto a single target file
	srcpath := ""
	// intl := ftv.EventMgr.DNDIsInternalSrc() // todo
	intl := true
	if intl {
		srcpath = string(md[1].Data) // 1 has file path, 0 = ki path, 2 = file data
	} else {
		srcpath = string(md[0].Data) // just file path
	}
	fname := filepath.Base(srcpath)
	tdir := AsNode(tfn.Parent())
	existing, sfn := fn.PasteCheckExisting(tdir, md)
	mode := os.FileMode(0664)
	if sfn != nil {
		mode = sfn.Info.Mode
	}
	if len(existing) == 1 && fname == tfn.Nm {
		gi.ChoiceDialog(nil, gi.DlgOpts{Title: "Overwrite?",
			Prompt: fmt.Sprintf("Overwrite target file: %s with source file of same name?, or diff (compare) two files, or cancel?", tfn.Nm)},
			[]string{"Overwrite Target", "Diff Files", "Cancel"}, func(dlg *gi.Dialog) {
				switch dlg.Data.(int) {
				case 0:
					CopyFile(tpath, srcpath, mode)
					fn.DragNDropFinalizeDefMod()
				case 1:
					// DiffFiles(tpath, srcpath)
					fn.DropCancel()
				case 2:
					fn.DropCancel()
				}
			})
	} else if len(existing) > 0 {
		gi.ChoiceDialog(nil, gi.DlgOpts{Title: "Overwrite?",
			Prompt: fmt.Sprintf("Overwrite target file: %s with source file: %s, or overwrite existing file with same name as source file (%s), or diff (compare) files, or cancel?", tfn.Nm, fname, fname)},
			[]string{"Overwrite Target", "Overwrite Existing", "Diff to Target", "Diff to Existing", "Cancel"},
			func(dlg *gi.Dialog) {
				switch dlg.Data.(int) {
				case 0:
					CopyFile(tpath, srcpath, mode)
					fn.DragNDropFinalizeDefMod()
				case 1:
					npath := filepath.Join(string(tdir.FPath), fname)
					CopyFile(npath, srcpath, mode)
					fn.DragNDropFinalizeDefMod()
				case 2:
					// DiffFiles(tpath, srcpath)
					fn.DropCancel()
				case 3:
					npath := filepath.Join(string(tdir.FPath), fname)
					_ = npath
					// DiffFiles(npath, srcpath)
					fn.DropCancel()
				case 4:
					fn.DropCancel()
				}
			})
	} else {
		gi.ChoiceDialog(nil, gi.DlgOpts{Title: "Overwrite?",
			Prompt: fmt.Sprintf("Overwrite target file: %s with source file: %s, or copy to: %s in current folder (which doesn't yet exist), or diff (compare) the two files, or cancel?", tfn.Nm, fname, fname)},
			[]string{"Overwrite Target", "Copy New File", "Diff Files", "Cancel"}, func(dlg *gi.Dialog) {
				switch dlg.Data.(int) {
				case 0:
					CopyFile(tpath, srcpath, mode)
					fn.DragNDropFinalizeDefMod()
				case 1:
					tdir.CopyFileToDir(srcpath, mode) // does updating, vcs stuff
					fn.DragNDropFinalizeDefMod()
				case 2:
					// DiffFiles(tpath, srcpath)
					fn.DropCancel()
				case 3:
					fn.DropCancel()
				}
			})
	}

}

// Dragged is called after target accepts the drop -- we just remove
// elements that were moved
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (fn *Node) Dragged(de events.Event) {
	// fmt.Printf("ftv dragged: %v\n", ftv.Path())
	// if de.Mod != events.DropMove {
	// 	return
	// }
	sroot := AsNode(fn.RootView.SrcNode)
	tfn := sroot.Node()
	if tfn == nil || tfn.IsExternal() {
		return
	}
	// todo
	md := de.Data
	nf := len(md) / 3 // always internal
	for i := 0; i < nf; i++ {
		npath := string(md[i*3].Data)
		sfni, err := sroot.FindPathTry(npath)
		if err != nil {
			fmt.Println(err)
			continue
		}
		sfn := AsNode(sfni)
		if sfn == nil {
			continue
		}
		// fmt.Printf("dnd deleting: %v  path: %v\n", sfn.Path(), sfn.FPath)
		sfn.DeleteFile()
	}
}

*/

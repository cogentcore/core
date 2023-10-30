// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

/*
// TreeInactiveExternFunc is an ActionUpdateFunc that inactivates action if node is external
var TreeInactiveExternFunc = ActionUpdateFunc(func(fni any, act *gi.Button) {
	ftv := AsNode(fni.(ki.Ki))
	fn := ftv.Node()
	if fn != nil {
		act.SetState(fn.IsExternal(), states.Disabled)
	}
})

// TreeActiveExternFunc is an ActionUpdateFunc that activates action if node is external
var TreeActiveExternFunc = ActionUpdateFunc(func(fni any, act *gi.Button) {
	ftv := AsNode(fni.(ki.Ki))
	fn := ftv.Node()
	if fn != nil {
		act.SetEnabledState(fn.IsExternal() && !fn.IsIrregular())
	}
})

// TreeInactiveDirFunc is an ActionUpdateFunc that inactivates action if node is a dir
var TreeInactiveDirFunc = ActionUpdateFunc(func(fni any, act *gi.Button) {
	ftv := AsNode(fni.(ki.Ki))
	fn := ftv.Node()
	if fn != nil {
		act.SetState(fn.IsDir() || fn.IsExternal(), states.Disabled)
	}
})

// TreeActiveDirFunc is an ActionUpdateFunc that activates action if node is a dir
var TreeActiveDirFunc = ActionUpdateFunc(func(fni any, act *gi.Button) {
	ftv := AsNode(fni.(ki.Ki))
	fn := ftv.Node()
	if fn != nil {
		act.SetEnabledState(fn.IsDir() && !fn.IsExternal())
	}
})

// TreeActiveNotInVcsFunc is an ActionUpdateFunc that inactivates action if node is not under version control
var TreeActiveNotInVcsFunc = ActionUpdateFunc(func(fni any, act *gi.Button) {
	ftv := AsNode(fni.(ki.Ki))
	fn := ftv.Node()
	if fn != nil {
		repo, _ := fn.Repo()
		if repo == nil || fn.IsDir() {
			act.SetEnabledState((false))
			return
		}
		act.SetEnabledState((fn.Info.Vcs == vci.Untracked))
	}
})

// TreeActiveInVcsFunc is an ActionUpdateFunc that activates action if node is under version control
var TreeActiveInVcsFunc = ActionUpdateFunc(func(fni any, act *gi.Button) {
	ftv := AsNode(fni.(ki.Ki))
	fn := ftv.Node()
	if fn != nil {
		repo, _ := fn.Repo()
		if repo == nil || fn.IsDir() {
			act.SetEnabledState((false))
			return
		}
		act.SetEnabledState((fn.Info.Vcs >= vci.Stored))
	}
})

// TreeActiveInVcsModifiedFunc is an ActionUpdateFunc that activates action if node is under version control
// and the file has been modified
var TreeActiveInVcsModifiedFunc = ActionUpdateFunc(func(fni any, act *gi.Button) {
	ftv := AsNode(fni.(ki.Ki))
	fn := ftv.Node()
	if fn != nil {
		repo, _ := fn.Repo()
		if repo == nil || fn.IsDir() {
			act.SetEnabledState((false))
			return
		}
		act.SetEnabledState((fn.Info.Vcs == vci.Modified || fn.Info.Vcs == vci.Added))
	}
})

// VcsGetRemoveLabelFunc gets the appropriate label for removing from version control
var VcsLabelFunc = LabelFunc(func(fni any, act *gi.Button) string {
	ftv := AsNode(fni.(ki.Ki))
	fn := ftv.Node()
	label := act.Text
	if fn != nil {
		repo, _ := fn.Repo()
		if repo != nil {
			label = strings.Replace(label, "Vcs", string(repo.Vcs()), 1)
		}
	}
	return label
})

var NodeProps = ki.Props{
	"CtxtMenuActive": ki.PropSlice{
		{"ShowFileInfo", ki.Props{
			"label": "File Info",
		}},
		{"OpenFileDefault", ki.Props{
			"label": "Open (w/default app)",
		}},
		{"sep-act", ki.BlankProp{}},
		{"DuplicateFiles", ki.Props{
			"label":    "Duplicate",
			"updtfunc": TreeInactiveDirFunc,
			"shortcut": keyfun.Duplicate,
		}},
		{"DeleteFiles", ki.Props{
			"label":    "Delete",
			"desc":     "Ok to delete file(s)?  This is not undoable and is not moving to trash / recycle bin",
			"updtfunc": TreeInactiveExternFunc,
			"shortcut": keyfun.Delete,
		}},
		{"RenameFiles", ki.Props{
			"label":    "Rename",
			"desc":     "Rename file to new file name",
			"updtfunc": TreeInactiveExternFunc,
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenAll", ki.Props{
			"updtfunc": TreeActiveDirFunc,
		}},
		{"CloseAll", ki.Props{
			"updtfunc": TreeActiveDirFunc,
		}},
		{"SortBy", ki.Props{
			"desc":     "Choose how to sort files in the directory -- default by Name, optionally can use Modification Time",
			"updtfunc": TreeActiveDirFunc,
			"Args": ki.PropSlice{
				{"Modification Time", ki.Props{}},
			},
		}},
		{"sep-new", ki.BlankProp{}},
		{"NewFile", ki.Props{
			"label":    "New File...",
			"desc":     "make a new file in this folder",
			"shortcut": keyfun.Insert,
			"updtfunc": TreeActiveDirFunc,
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
			"shortcut": keyfun.InsertAfter,
			"updtfunc": TreeActiveDirFunc,
			"Args": ki.PropSlice{
				{"Folder Name", ki.Props{
					"width": 60,
				}},
			},
		}},
		{"sep-vcs", ki.BlankProp{}},
		{"AddToVcs", ki.Props{
			"desc":       "Add file to version control",
			"updtfunc":   TreeActiveNotInVcsFunc,
			"label-func": VcsLabelFunc,
		}},
		{"DeleteFromVcs", ki.Props{
			"desc":       "Delete file from version control",
			"updtfunc":   TreeActiveInVcsFunc,
			"label-func": VcsLabelFunc,
		}},
		{"CommitToVcs", ki.Props{
			"desc":       "Commit file to version control",
			"updtfunc":   TreeActiveInVcsModifiedFunc,
			"label-func": VcsLabelFunc,
		}},
		{"RevertVcs", ki.Props{
			"desc":       "Revert file to last commit",
			"updtfunc":   TreeActiveInVcsModifiedFunc,
			"label-func": VcsLabelFunc,
		}},
		{"sep-vcs-log", ki.BlankProp{}},
		{"DiffVcs", ki.Props{
			"desc":       "shows the diffs between two versions of this file, given by the revision specifiers -- if empty, defaults to A = current HEAD, B = current WC file.   -1, -2 etc also work as universal ways of specifying prior revisions.",
			"updtfunc":   TreeActiveInVcsFunc,
			"label-func": VcsLabelFunc,
			"Args": ki.PropSlice{
				{"Revision A", ki.Props{}},
				{"Revision B", ki.Props{}},
			},
		}},
		{"LogVcs", ki.Props{
			"desc":       "shows the VCS log of commits for this file, optionally with a since date qualifier: If since is non-empty, it should be a date-like expression that the VCS will understand, such as 1/1/2020, yesterday, last year, etc (SVN only supports a max number of entries).  If allFiles is true, then the log will show revisions for all files, not just this one.",
			"updtfunc":   TreeActiveInVcsFunc,
			"label-func": VcsLabelFunc,
			"Args": ki.PropSlice{
				{"All Files", ki.Props{}},
				{"Since Date", ki.Props{}},
			},
		}},
		{"BlameVcs", ki.Props{
			"desc":       "shows the VCS blame report for this file, reporting for each line the revision and author of the last change.",
			"updtfunc":   TreeActiveInVcsFunc,
			"label-func": VcsLabelFunc,
		}},
		{"sep-extrn", ki.BlankProp{}},
		{"RemoveFromExterns", ki.Props{
			"desc":       "Remove file from external files listt",
			"updtfunc":   TreeActiveExternFunc,
			"label-func": VcsLabelFunc,
		}},
	},
}

var NodeProps = ki.Props{
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
		{"OpenFileWith", ki.Props{
			"label": "Open With...",
			"desc":  "Open the file with given command...",
			"Args": ki.PropSlice{
				{"Command", ki.Props{
					"width": 60,
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

*/

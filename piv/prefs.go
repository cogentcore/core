// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package piv

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/parse"
)

// ProjPrefs are the preferences for saving for a project -- this IS the project file
type ProjPrefs struct {
	ProjFile   gi.FileName     `desc:"filename for project (i.e, these preference)"`
	ParserFile gi.FileName     `desc:"filename for parser"`
	TestFile   gi.FileName     `desc:"the file for testing"`
	TraceOpts  parse.TraceOpts `desc:"the options for tracing parsing"`
}

var KiT_ProjPrefs = kit.Types.AddType(&ProjPrefs{}, nil)

// OpenJSON open from JSON file
func (pf *ProjPrefs) OpenJSON(filename gi.FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, pf)
	if err == nil {
		pf.ProjFile = filename
	}
	return err
}

// SaveJSON save to JSON file
func (pf *ProjPrefs) SaveJSON(filename gi.FileName) error {
	pf.ProjFile = filename
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		log.Println(err)
	}
	return err
}

// InitPrefs is the overall init at startup for PiView project
func InitPrefs() {
	OpenPaths()
}

//////////////////////////////////////////////////////////////////////////////////////
//   Saved Projects / Paths

// SavedPaths is a slice of strings that are file paths
var SavedPaths gi.FilePaths

// SavedPathsFileName is the name of the saved file paths file in GoPi prefs directory
var SavedPathsFileName = "gopi_saved_paths.json"

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.SaveJSON(pnm)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	pdir := oswin.TheApp.AppPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.OpenJSON(pnm)
}

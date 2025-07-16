// Copyright 2015 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mobile

import (
	"bytes"
	"path/filepath"
	"text/template"
)

var gopath string

func diffOutput(got string, wantTmpl *template.Template) (string, error) {
	got = filepath.ToSlash(got)

	wantBuf := new(bytes.Buffer)
	data, err := defaultOutputData("")
	if err != nil {
		return "", err
	}
	if err := wantTmpl.Execute(wantBuf, data); err != nil {
		return "", err
	}
	want := wantBuf.String()
	if got != want {
		return diff(got, want)
	}
	return "", nil
}

type outputData struct {
	GOOS      string
	GOARCH    string
	GOPATH    string
	NDKARCH   string
	EXE       string // .extension for executables. (ex. ".exe" for windows)
	Xproj     string
	Xcontents string
	Xinfo     infoPlistTmplData
}

func defaultOutputData(teamID string) (outputData, error) {
	projPbxproj := new(bytes.Buffer)
	if err := projPbxprojTmpl.Execute(projPbxproj, projPbxprojTmplData{
		TeamID: teamID,
	}); err != nil {
		return outputData{}, err
	}

	data := outputData{
		GOOS:      goos,
		GOARCH:    goarch,
		GOPATH:    gopath,
		NDKARCH:   archNDK(),
		Xproj:     projPbxproj.String(),
		Xcontents: contentsJSON,
		Xinfo:     infoPlistTmplData{BundleID: "org.golang.todo.basic", Name: "Basic"},
	}
	if goos == "windows" {
		data.EXE = ".exe"
	}
	return data, nil
}

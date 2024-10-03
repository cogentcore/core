// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Var represents one variable
type Var struct {
	Name string

	Type string
}

// Group represents one variable group.
type Group struct {
	Vars []*Var
}

// System represents a ComputeSystem, and its variables.
type System struct {
	Name string

	Groups []*Group
}

// Kernel represents a kernel function, which is the basis for
// each wgsl generated code file.
type Kernel struct {
	Name string

	// accumulating lines of code for the wgsl file.
	FileLines [][]byte
}

// File has info for a file being processed
type File struct {
	Name string

	Lines [][]byte
}

// State holds the current processing state
type State struct {
	// Config options
	Config *Config

	// files with gosl content in current directory
	Files map[string]*File

	// has all the imports
	Imports map[string]map[string]*File

	// short package names
	ImportPackages map[string]bool

	Kernels map[string]*Kernel

	Systems map[string]*System

	// ExcludeMap is the compiled map of functions to exclude.
	ExcludeMap map[string]bool
}

func (st *State) Init(cfg *Config) {
	st.Config = cfg
	st.Imports = make(map[string]map[string]*File)
	st.Kernels = make(map[string]*Kernel)
	st.Systems = make(map[string]*System)
	st.ExcludeMap = make(map[string]bool)
	ex := strings.Split(cfg.Exclude, ",")
	for _, fn := range ex {
		st.ExcludeMap[fn] = true
	}

	sy := &System{Name: "Default"}
	sy.Groups = append(sy.Groups, &Group{})
	st.Systems["Default"] = sy
}

func (st *State) Run() error {
	if gomod := os.Getenv("GO111MODULE"); gomod == "off" {
		err := errors.New("gosl only works in go modules mode, but GO111MODULE=off")
		return err
	}
	if st.Config.Output == "" {
		st.Config.Output = "shaders"
	}
	imps := filepath.Join(st.Config.Output, "imports")
	os.MkdirAll(st.Config.Output, 0755)
	os.MkdirAll(imps, 0755)
	RemoveGenFiles(st.Config.Output)
	RemoveGenFiles(imps)

	st.ProjectFiles()   // recursively gets imports etc.
	st.ExtractImports() // save all the import files

	return nil
}

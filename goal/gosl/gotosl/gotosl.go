// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// System represents a ComputeSystem, and its kernels and variables.
type System struct {
	Name string

	// Kernels are the kernels using this compute system.
	Kernels map[string]*Kernel

	// Groups are the variables for this compute system.
	Groups []*Group
}

func NewSystem(name string) *System {
	sy := &System{Name: name}
	sy.Kernels = make(map[string]*Kernel)
	return sy
}

// Kernel represents a kernel function, which is the basis for
// each wgsl generated code file.
type Kernel struct {
	Name string

	Args string

	// function code
	FuncCode string

	// Lines is full shader code
	Lines [][]byte
}

// Var represents one global system buffer variable.
type Var struct {
	Name string

	// comment docs about this var.
	Doc string

	// Type of variable: either []Type or tensor.Float32, tensor.Int32
	Type string
}

// Group represents one variable group.
type Group struct {
	Name string

	// comment docs about this group
	Doc string

	// Uniform indicates a uniform group; else default is Storage
	Uniform bool

	Vars []*Var
}

// File has contents of a file as lines of bytes.
type File struct {
	Name  string
	Lines [][]byte
}

// State holds the current Go -> WGSL processing state.
type State struct {
	// Config options.
	Config *Config

	// path to shaders/imports directory.
	ImportsDir string

	// GoFiles are all the files with gosl content in current directory.
	GoFiles map[string]*File

	// GoImports has all the imported files.
	GoImports map[string]map[string]*File

	// ImportPackages has short package names, to remove from go code
	// so everything lives in same main package.
	ImportPackages map[string]bool

	// Systems has the kernels and variables for each system.
	// There is an initial "Default" system when system is not specified.
	Systems map[string]*System

	// SLImportFiles are all the extracted and translated WGSL files in shaders/imports,
	// which are copied into the generated shader kernel files.
	SLImportFiles []*File

	// ExcludeMap is the compiled map of functions to exclude in Go -> WGSL translation.
	ExcludeMap map[string]bool
}

func (st *State) Init(cfg *Config) {
	st.Config = cfg
	st.GoImports = make(map[string]map[string]*File)
	st.Systems = make(map[string]*System)
	st.ExcludeMap = make(map[string]bool)
	ex := strings.Split(cfg.Exclude, ",")
	for _, fn := range ex {
		st.ExcludeMap[fn] = true
	}
	st.Systems["Default"] = NewSystem("Default")
}

func (st *State) Run() error {
	if gomod := os.Getenv("GO111MODULE"); gomod == "off" {
		err := errors.New("gosl only works in go modules mode, but GO111MODULE=off")
		return err
	}
	if st.Config.Output == "" {
		st.Config.Output = "shaders"
	}
	st.ImportsDir = filepath.Join(st.Config.Output, "imports")
	os.MkdirAll(st.Config.Output, 0755)
	os.MkdirAll(st.ImportsDir, 0755)
	RemoveGenFiles(st.Config.Output)
	RemoveGenFiles(st.ImportsDir)

	st.ProjectFiles()   // get list of all files, recursively gets imports etc.
	st.ExtractFiles()   // get .go from project files
	st.ExtractImports() // get .go from imports
	st.TranslateDir("./" + st.ImportsDir)

	for _, sy := range st.Systems {
		for _, kn := range sy.Kernels {
			st.GenKernel(sy, kn)
		}
	}

	return nil
}

// System returns the given system by name, making if not made.
// if name is empty, "Default" is used.
func (st *State) System(sysname string) *System {
	if sysname == "" {
		sysname = "Default"
	}
	sy, ok := st.Systems[sysname]
	if ok {
		return sy
	}
	sy = NewSystem(sysname)
	st.Systems[sysname] = sy
	return sy
}

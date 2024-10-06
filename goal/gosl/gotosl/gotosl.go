// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"cogentcore.org/core/base/errors"
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

	// Filename is the name of the kernel shader file, e.g., shaders/Compute.wgsl
	Filename string

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

	// Type of variable: either []Type or F32, U32 for tensors
	Type string

	// ReadOnly indicates that this variable is never read back from GPU,
	// specified by the gosl:read-only property in the variable comments.
	// It is important to optimize GPU memory usage to indicate this.
	ReadOnly bool

	// True if a tensor type
	Tensor bool

	// Number of dimensions
	TensorDims int

	// data kind of the tensor
	TensorKind reflect.Kind
}

func (vr *Var) SetTensorKind() {
	kindStr := strings.TrimPrefix(vr.Type, "tensor.")
	kind := reflect.Float32
	switch kindStr {
	case "Float32":
		kind = reflect.Float32
	case "Uint32":
		kind = reflect.Uint32
	case "Int32":
		kind = reflect.Int32
	default:
		errors.Log(fmt.Errorf("gosl: variable %q type is not supported: %q", vr.Name, kindStr))
	}
	vr.TensorKind = kind
}

// SLType returns the WGSL type string
func (vr *Var) SLType() string {
	if vr.Tensor {
		switch vr.TensorKind {
		case reflect.Float32:
			return "f32"
		case reflect.Int32:
			return "i32"
		case reflect.Uint32:
			return "u32"
		}
	} else {
		return vr.Type[2:]
	}
	return ""
}

// IndexFunc returns the index function name
func (vr *Var) IndexFunc() string {
	typ := strings.ToUpper(vr.SLType())
	return fmt.Sprintf("Index%s%dD", typ, vr.TensorDims)
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

	// name of the package
	Package string

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

	// generated Go GPU gosl.go file contents
	GPUFile File

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

	st.ProjectFiles() // get list of all files, recursively gets imports etc.
	if len(st.GoFiles) == 0 {
		if st.Config.Debug {
			fmt.Println("gosl: no gosl files in current directory")
		}
		return nil
	}
	st.ExtractFiles()   // get .go from project files
	st.ExtractImports() // get .go from imports
	st.TranslateDir("./" + st.ImportsDir)

	for _, sy := range st.Systems {
		for _, kn := range sy.Kernels {
			st.GenKernel(sy, kn)
		}
	}

	st.GenGPU()

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

// GlobalVar returns global variable of given name, if found.
func (st *State) GlobalVar(vrnm string) *Var {
	if st == nil {
		return nil
	}
	if st.Systems == nil {
		return nil
	}
	for _, sy := range st.Systems {
		for _, gp := range sy.Groups {
			for _, vr := range gp.Vars {
				if vr.Name == vrnm {
					return vr
				}
			}
		}
	}
	return nil
}

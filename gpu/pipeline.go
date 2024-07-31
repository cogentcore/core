// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"log"
	"log/slog"

	"cogentcore.org/core/base/errors"
	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Pipeline is the shared Base for Graphics and Compute Pipelines.
// It manages Shader program(s) that accomplish a specific
// type of rendering or compute function, using Vars / Values
// defined by the overall System.
// In the graphics context, each pipeline could handle a different
// class of materials (textures, Phong lighting, etc).
type Pipeline struct {
	// unique name of this pipeline
	Name string

	// system that we belong to and manages all shared resources:
	// Vars, Values, etc
	Sys *System

	// Shaders contains actual shader code loaded for this pipeline.
	// A single shader can have multiple entry points: see Entries.
	Shaders map[string]*Shader

	// Entries contains the entry points into shader code,
	// which are what is actually called.
	Entries map[string]*ShaderEntry

	layout *wgpu.PipelineLayout
}

// Vars returns a pointer to the vars for this pipeline,
// which has Values within it.
func (pl *Pipeline) Vars() *Vars {
	return &pl.Sys.Vars
}

// AddShader adds Shader with given name to the pipeline
func (pl *Pipeline) AddShader(name string) *Shader {
	if pl.Shaders == nil {
		pl.Shaders = make(map[string]*Shader)
	}
	if sh, has := pl.Shaders[name]; has {
		log.Printf("gpu.Pipeline AddShader: Shader named: %s already exists in pipline: %s\n", name, pl.Name)
		return sh
	}
	sh := NewShader(name, &pl.Sys.Device)
	pl.Shaders[name] = sh
	return sh
}

// ShaderByName returns Shader by name.
// Returns nil if not found (error auto logged).
func (pl *Pipeline) ShaderByName(name string) *Shader {
	sh, ok := pl.Shaders[name]
	if !ok {
		slog.Error("gpu.Pipeline ShaderByName", "Shader", name, "not found in pipeline", pl.Name)
		return nil
	}
	return sh
}

// EntryByName returns ShaderEntry by name, which is Shader:Entry.
// Returns nil if not found (error auto logged).
func (pl *Pipeline) EntryByName(name string) *ShaderEntry {
	sh, ok := pl.Entries[name]
	if !ok {
		slog.Error("gpu.Pipeline EntryByName", "Entry", name, "not found in pipeline", pl.Name)
		return nil
	}
	return sh
}

// EntryByType returns ShaderEntry by ShaderType.
// Returns nil if not found.
func (pl *Pipeline) EntryByType(typ ShaderTypes) *ShaderEntry {
	for _, sh := range pl.Entries {
		if sh.Type == typ {
			return sh
		}
	}
	return nil
}

// AddEntry adds ShaderEntry for given shader, [ShaderTypes], and entry function name.
func (pl *Pipeline) AddEntry(sh *Shader, typ ShaderTypes, entry string) *ShaderEntry {
	if pl.Entries == nil {
		pl.Entries = make(map[string]*ShaderEntry)
	}
	name := sh.Name + ":" + entry
	if se, has := pl.Entries[name]; has {
		slog.Error("gpu.Pipeline AddEntry", "ShaderEntry named", name, "already exists in pipline", pl.Name)
		return se
	}
	se := NewShaderEntry(sh, typ, entry)
	pl.Entries[name] = se
	return se
}

// ReleaseShaders releases the shaders
func (pl *Pipeline) ReleaseShaders() {
	for _, sh := range pl.Shaders {
		sh.Release()
	}
	pl.Shaders = nil
	pl.Entries = nil
}

// BindLayout configures the PipeLineLayout based on Vars
func (pl *Pipeline) BindLayout() error {
	lays := pl.Vars().bindLayout(&pl.Sys.Device)

	rpl, err := pl.Sys.Device.Device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:            pl.Name,
		BindGroupLayouts: lays,
	})
	if errors.Log(err) != nil {
		return err
	}
	pl.layout = rpl
	return nil
}

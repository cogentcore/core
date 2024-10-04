// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"io/fs"
	"path"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"github.com/cogentcore/webgpu/wgpu"
)

// ComputePipeline is a compute pipeline, which runs shader code on vars data.
type ComputePipeline struct {
	Pipeline

	// computePipeline is the configured, instantiated wgpu pipeline
	computePipeline *wgpu.ComputePipeline
}

// NewComputePipeline returns a new ComputePipeline.
func NewComputePipeline(name string, sy System) *ComputePipeline {
	pl := &ComputePipeline{}
	pl.Name = name
	pl.System = sy
	return pl
}

// NewComputePipelineShaderFS returns a new ComputePipeline,
// opening the given shader code file from given filesystem,
// and setting the name of the pipeline to the filename
// (without paths or extensions).  The shader entry point is "main".
// This is a convenience method for standard case where there is
// one shader program per pipeline.
func NewComputePipelineShaderFS(fsys fs.FS, fname string, sy *ComputeSystem) *ComputePipeline {
	name, _ := fsx.ExtSplit(path.Base(fname))
	pl := &ComputePipeline{}
	pl.Name = name
	pl.System = sy
	sh := pl.AddShader(name)
	errors.Log(sh.OpenFileFS(fsys, fname))
	pl.AddEntry(sh, ComputeShader, "main")
	sy.ComputePipelines[pl.Name] = pl
	return pl
}

// Dispatch adds commands to given compute encoder to run this
// pipeline for given number of *warps* (work groups of compute threads)
// along 3 dimensions, which then generate indexes passed into the shader.
// Calls BindPipeline and then DispatchWorkgroups.
// In WGSL, the @workgroup_size(x, y, z) directive specifies the number
// of threads allocated per warp -- the actual number of elements
// processed is threads * warps per each dimension. See Warps function.
// The hardware typically has 32 (NVIDIA, M1, M2) or 64 (AMD) hardware
// threads per warp, and so 64 is typically used as a default sum of
// threads per warp across all of the dimensions.
// Can use subsets of dimensions by using 1 for the other dimensions,
// and see [Dispatch1D] for a convenience method that automatically
// computes the number of warps for a 1D compute shader (everthing in x).
func (pl *ComputePipeline) Dispatch(ce *wgpu.ComputePassEncoder, nx, ny, nz int) error {
	err := pl.BindPipeline(ce)
	if err != nil {
		return err
	}
	ce.DispatchWorkgroups(uint32(nx), uint32(ny), uint32(nz))
	return nil
}

// Dispatch1D adds commands to given compute encoder to run this
// pipeline for given number of computational elements along the first
// (X) dimension, for given number *elements* (threads) per warp (typically 64).
// See [Dispatch] for full info.
// This is just a convenience method for common 1D case that calls
// the Warps method for you.
func (pl *ComputePipeline) Dispatch1D(ce *wgpu.ComputePassEncoder, n, threads int) error {
	return pl.Dispatch(ce, Warps(n, threads), 1, 1)
}

// BindAllGroups binds the Current Value for all variables across all
// variable groups, as the Value to use by shader.
// Automatically called in BindPipeline at start of render for pipeline.
// Be sure to set Current index to correct value before calling!
func (pl *ComputePipeline) BindAllGroups(ce *wgpu.ComputePassEncoder) {
	vs := pl.Vars()
	ngp := vs.NGroups()
	for gi := range ngp {
		pl.BindGroup(ce, gi)
	}
}

// BindGroup binds the Current Value for all variables in given
// variable group, as the Value to use by shader.
// Be sure to set Current index to correct value before calling!
func (pl *ComputePipeline) BindGroup(ce *wgpu.ComputePassEncoder, group int) {
	vs := pl.Vars()
	vg := vs.Groups[group]
	bg, dynOffs, err := vg.bindGroup(vs)
	if err == nil {
		ce.SetBindGroup(uint32(vg.Group), bg, dynOffs)
	}
}

// BindPipeline binds this pipeline as the one to use for next commands in
// the given compute pass.
// This also calls BindAllGroups, to bind the Current Value for all variables.
// Be sure to set the desired Current value prior to calling.
func (pl *ComputePipeline) BindPipeline(ce *wgpu.ComputePassEncoder) error {
	if pl.computePipeline == nil {
		err := pl.Config(false)
		if errors.Log(err) != nil {
			return err
		}
	}
	ce.SetPipeline(pl.computePipeline)
	pl.BindAllGroups(ce)
	return nil
}

// Config is called once all the Config options have been set
// using Set* methods, and the shaders have been loaded.
// The parent ComputeSystem has already done what it can for its config.
// The rebuild flag indicates whether pipelines should rebuild
func (pl *ComputePipeline) Config(rebuild bool) error {
	if pl.computePipeline != nil {
		if !rebuild {
			return nil
		}
		pl.releasePipeline() // starting over: note: requires keeping shaders around
	}
	play, err := pl.bindLayout()
	if errors.Log(err) != nil {
		return err
	}
	defer play.Release()

	sh := pl.EntryByType(ComputeShader)
	cp, err := pl.System.Device().Device.CreateComputePipeline(&wgpu.ComputePipelineDescriptor{
		Layout: play,
		Compute: wgpu.ProgrammableStageDescriptor{
			Module:     sh.Shader.module,
			EntryPoint: sh.Entry,
		},
	})
	if errors.Log(err) != nil {
		return err
	}
	pl.computePipeline = cp
	return nil
}

func (pl *ComputePipeline) Release() {
	pl.releaseShaders()
	pl.releasePipeline()
}

func (pl *ComputePipeline) releasePipeline() {
	if pl.computePipeline != nil {
		pl.computePipeline.Release()
		pl.computePipeline = nil
	}
}

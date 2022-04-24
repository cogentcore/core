// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import vk "github.com/vulkan-go/vulkan"

// System manages a system of Pipelines that all share
// a common collection of Vars, Vals, and a Memory manager.
// For example, this could be a collection of different
// pipelines for different material types, or different
// compute operations performed on a common set of data.
type System struct {
	Name        string               `desc:"optional name of this System"`
	Pipelines   []*Pipeline          `desc:"all pipelines"`
	PipelineMap map[string]*Pipeline `desc:"map of all pipelines -- names must be unique"`
	Vars        Vars                 `desc:"the common set of variables used by all Piplines"`
	Vals        Vals                 `desc:"values of Vars, each with a unique name -- can be any number of different values per same Var (e.g., different meshes with vertex data) -- up to user code to bind each Var prior to pipeline execution.  Each of these Vals is mapped into GPU memory via a Memory manager object."`
	NSets       int                  `desc:"number of replicated sets of Vals of any subset of variables, with corresponding descriptor set -- each a copy of same overall descriptor layout generated from vars -- e.g., if using different vals for each frame in a swapchain"`
	SetVals     []*Vals              `desc:"values of variables allocated into replicated sets, NSets in size -- any vals not found in here will be looked up in the main Vals collection, so you can mix and match dynamic and static vals within sets."`
	Mem         Memory               `desc:"manages all the memory for all the Vals"`

	VkDescPool vk.DescriptorPool `desc:"vulkan descriptor pool"`
}

// AddPipeline adds given pipeline
func (sy *System) AddPipeline(pl *Pipeline) {
	if sy.PipelineMap == nil {
		sy.PipelineMap = make(map[string]*Pipeline)
	}
	sy.Pipelines = append(sy.Pipelines, pl)
	sy.PipelineMap[pl.Name] = pl

}

// AddNewPipeline adds a new pipeline
func (sy *System) AddNewPipeline(name string) *Pipeline {
	pl := &Pipeline{Name: name}
	sy.AddPipeline(pl)
	return pl
}

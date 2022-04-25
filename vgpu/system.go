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
// It maintains its own logical device and associated queue.
type System struct {
	Name        string                `desc:"optional name of this System"`
	GPU         *GPU                  `desc:"gpu device"`
	Device      Device                `desc:"logical device for this System -- has its own queues"`
	Compute     bool                  `desc:"if true, this is a compute system -- otherwise is graphics"`
	Pipelines   []*Pipeline           `desc:"all pipelines"`
	PipelineMap map[string]*Pipeline  `desc:"map of all pipelines -- names must be unique"`
	Vars        Vars                  `desc:"the common set of variables used by all Piplines"`
	Mem         Memory                `desc:"manages all the memory for all the Vals"`
	Views       map[string]*ImageView `desc:"uniquely-named image views"`
	Samplers    map[string]*Sampler   `desc:"uniquely-named image samplers -- referred to by name in Vars of type Sampler or CombinedImage"`

	VkDescPool vk.DescriptorPool `desc:"vulkan descriptor pool"`
}

// Init initializes the System: creates logical device, which is
// either a Compute device or Graphics one.
func (sy *System) Init(gp *GPU, name string, compute bool) error {
	sy.GPU = gp
	sy.Name = name
	sy.Compute = compute
	if compute {
		return sy.Device.Init(gp, vk.QueueComputeBit)
	}
	return sy.Device.Init(gp, vk.QueueGraphicsBit)
}

func (sy *System) Destroy() {
	sy.Mem.Destroy()
	for _, pl := range sy.Pipelines {
		pl.Destroy()
	}
	if sy.Views != nil {
		for _, iv := range sy.Views {
			iv.Destroy(sy.Device.Device)
		}
	}
	if sy.Samplers != nil {
		for _, sm := range sy.Samplers {
			sm.Destroy(sy.Device.Device)
		}
	}
	sy.Device.Destroy()
	sy.GPU = nil
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

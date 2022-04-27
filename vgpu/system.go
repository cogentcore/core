// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"log"

	"github.com/goki/ki/ints"
	vk "github.com/vulkan-go/vulkan"
)

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
}

// Init initializes the System: creates logical device, which is
// either a Compute device or Graphics one.
func (sy *System) Init(gp *GPU, name string, compute bool) error {
	sy.GPU = gp
	sy.Name = name
	sy.Compute = compute
	if compute {
		sy.Device.Init(gp, vk.QueueComputeBit)
	} else {
		sy.Device.Init(gp, vk.QueueGraphicsBit)
	}
	sy.Mem.Init(gp, &sy.Device)

	return nil
}

func (sy *System) Destroy() {
	sy.Mem.Destroy(sy.Device.Device)
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
	sy.Vars.Destroy(sy.Device.Device)
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
	pl.Init(sy)
	sy.AddPipeline(pl)
	return pl
}

// Config configures the entire system, after everything has been
// setup (Pipelines, Vars, etc).  Memory / Vals do not yet need to
// be configured and are not Config'd by this call.
func (sy *System) Config() {
	sy.Vars.Config()
	fmt.Printf("%s\n", sy.Vars.StringDoc())
	sy.Vars.DescLayout(sy.Device.Device)
	for _, pl := range sy.Pipelines {
		pl.Config()
	}
}

// SetVals sets the Vals for given Set of Vars, by name, in order
// that they appear in the Set list of roles and vars
func (sy *System) SetVals(set int, vals ...string) {
	nv := len(vals)
	ws := make([]vk.WriteDescriptorSet, nv)
	sd := sy.Vars.SetDesc[set]
	nv = ints.MinInt(nv, len(sd.Vars))
	for i := 0; i < nv; i++ {
		vnm := vals[i]
		vl, err := sy.Mem.Vals.ValByNameTry(vnm)
		if err != nil {
			log.Println(err)
			continue
		}
		wd := vk.WriteDescriptorSet{
			SType:           vk.StructureTypeWriteDescriptorSet,
			DstSet:          sd.DescSet,
			DstBinding:      uint32(vl.Var.BindLoc),
			DescriptorCount: 1,
			DescriptorType:  RoleDescriptors[vl.Var.Role],
		}
		if vl.Var.Role < StorageImage {
			off := vk.DeviceSize(vl.Offset)
			if vl.Var.Role.IsDynamic() {
				off = 0 // off must be 0 for dynamic
			}
			buff := sy.Mem.Buffs[vl.BuffType()]
			wd.PBufferInfo = []vk.DescriptorBufferInfo{{
				Offset: off,
				Range:  vk.DeviceSize(vl.MemSize),
				Buffer: buff.Dev,
			}}
			if vl.Var.Role.IsDynamic() {
				sy.Vars.DynOffs[vl.Var.DynOffIdx] = uint32(vl.Offset)
			}
		} else {
			// wd.DescriptorCount = uint32(len(texEnabled))
			// wd.PImageInfo =      texInfos
		}
		ws[i] = wd
	}
	vk.UpdateDescriptorSets(sy.Device.Device, uint32(nv), ws, 0, nil)
}

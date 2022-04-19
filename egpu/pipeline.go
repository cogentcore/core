// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"log"

	vk "github.com/vulkan-go/vulkan"
)

// Pipeline manages a sequence of Programs that can be activated in an
// appropriate order to achieve some overall step of rendering.
// A new Pipeline can be created in TheGPU.NewPipeline().
// It corresponds to a vulkan pipeline and can be associated with
// a graphics device on the GPU or a compute device
type Pipeline struct {
	GPU        *GPU
	Device     vk.Device `desc:"device for this pipeline -- could be GPU or Compute"`
	QueueIndex uint32    `desc:"queue index for device"`
	name       string
	progs      map[string]*Program

	CmdPool vk.CommandPool
	CmdBuff vk.CommandBuffer
}

// Name returns name of this pipeline
func (pl *Pipeline) Name() string {
	return pl.name
}

// SetName sets name of this pipeline
func (pl *Pipeline) SetName(name string) {
	pl.name = name
}

// AddProgram adds program with given name to the pipeline
func (pl *Pipeline) AddProgram(name string) *Program {
	if pl.progs == nil {
		pl.progs = make(map[string]*Program)
	}
	pr := &Program{name: name}
	pl.progs[name] = pr
	return pr
}

// ProgramByName returns Program by name.
// Returns nil if not found (error auto logged).
func (pl *Pipeline) ProgramByName(name string) *Program {
	pr, ok := pl.progs[name]
	if !ok {
		log.Printf("glgpu Pipeline ProgramByName: Program: %s not found in pipeline: %s\n", name, pl.name)
		return nil
	}
	return pr
}

// Programs returns list (slice) of Programs in pipeline
func (pl *Pipeline) Programs() []*Program {
	return nil
}

func (pl *Pipeline) Delete() {
	for _, pr := range pl.progs {
		pr.Delete()
	}
	vk.DestroyCommandPool(pl.Device, pl.CmdPool, nil)
}

func (pl *Pipeline) Init(gp *GPU) {
	pl.GPU = gp

	var CmdPool vk.CommandPool
	ret := vk.CreateCommandPool(pl.Device, &vk.CommandPoolCreateInfo{
		SType:            vk.StructureTypeCommandPoolCreateInfo,
		QueueFamilyIndex: pl.QueueIndex,
	}, nil, &CmdPool)
	IfPanic(NewError(ret))
	pl.CmdPool = CmdPool

	var CmdBuff = make([]vk.CommandBuffer, 1)
	ret = vk.AllocateCommandBuffers(pl.Device, &vk.CommandBufferAllocateInfo{
		SType:              vk.StructureTypeCommandBufferAllocateInfo,
		CommandPool:        pl.CmdPool,
		Level:              vk.CommandBufferLevelPrimary,
		CommandBufferCount: 1,
	}, CmdBuff)
	IfPanic(NewError(ret))
	pl.CmdBuff = CmdBuff[0]

	ret = vk.BeginCommandBuffer(pl.CmdBuff, &vk.CommandBufferBeginInfo{
		SType: vk.StructureTypeCommandBufferBeginInfo,
	})
	IfPanic(NewError(ret))

}

/*
func (pl *Pipeline) Run() {
	graphicsQueue := sf.GPU.GraphicsQueue
	var nullFence vk.Fence
	ret = vk.QueueSubmit(graphicsQueue, 1, []vk.SubmitInfo{{
		SType: vk.StructureTypeSubmitInfo,
		PWaitDstStageMask: []vk.SurfaceStageFlags{
			vk.SurfaceStageFlags(vk.SurfaceStageColorAttachmentOutputBit),
		},
		WaitSemaphoreCount: 1,
		PWaitSemaphores: []vk.Semaphore{
			sf.ImageAcquiredSemaphores[sf.FrameIndex],
		},
		CommandBufferCount: 1,
		PCommandBuffers: []vk.CommandBuffer{
			sf.ImageResources[idx].CmdBuff,
		},
		SignalSemaphoreCount: 1,
		PSignalSemaphores: []vk.Semaphore{
			sf.DrawCompleteSemaphores[sf.FrameIndex],
		},
	}}, nullFence)
	IfPanic(NewError(ret))
}
*/

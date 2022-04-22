// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"log"
)

// Pipeline manages a sequence of compute steps, which are fixed once configured.
// Each has an associated set of Vars, which could be maintained collectively for
// multiple different such piplines.
type Pipeline struct {
	Name    string
	GPU     *GPU
	Device  Device `desc:"device for this pipeline -- could be GPU or Compute"`
	CmdPool CmdPool
	Vars    *Vars              `desc:"variables associated with this pipeline"`
	Shaders map[string]*Shader `desc:"shaders loaded for this pipeline"`
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
	pl.CmdPool.Destroy(&pl.Device)
}

// InitCompute initializes for compute pipeline
func (pl *Pipeline) InitCompute(cp *Compute) {
	pl.GPU = cp.GPU
	pl.Device = cp.Device
	pl.InitPipeline()
}

func (pl *Pipeline) InitPipeline() {
	pl.CmdPool.Init(&pl.Device, 0)
	pl.CmdPool.Buff = pl.CmdPool.MakeBuff(&pl.Device)
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

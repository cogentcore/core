// Copyright (c) 2022, The GoKi Authors. All rights reserved.
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
	Name      string
	GPU       *GPU
	Device    Device `desc:"device for this pipeline -- could be GPU or Compute"`
	CmdPool   CmdPool
	Vars      *Vars              `desc:"variables associated with this pipeline"`
	Shaders   []*Shader          `desc:"shaders in order added -- should be execution order"`
	ShaderMap map[string]*Shader `desc:"shaders loaded for this pipeline"`
}

// AddShader adds program with given name to the pipeline
func (pl *Pipeline) AddShader(name string) *Shader {
	if pl.ShaderMap == nil {
		pl.ShaderMap = make(map[string]*Shader)
	}
	sh := &Shader{name: name}
	pl.Shaders = append(pl.Shaders, sh)
	pl.ShaderMap[name] = sh
	return sh
}

// ShaderByName returns Shader by name.
// Returns nil if not found (error auto logged).
func (pl *Pipeline) ShaderByName(name string) *Shader {
	sh, ok := pl.ShaderMap[name]
	if !ok {
		log.Printf("vgpu.Pipeline ShaderByName: Shader: %s not found in pipeline: %s\n", name, pl.Name)
		return nil
	}
	return sh
}

func (pl *Pipeline) Delete() {
	for _, sh := range pl.ShaderMap {
		sh.Delete()
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

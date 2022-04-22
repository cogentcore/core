// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/demos
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License
// and https://bakedbits.dev/posts/vulkan-compute-example/

package main

import (
	"fmt"
	"math/rand"
	"runtime"

	"github.com/emer/egpu/egpu"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/gi/oswin/gpu"

	vk "github.com/vulkan-go/vulkan"
)

func init() {
	// must lock main thread for gpu!  this also means that vulkan must be used
	// for gogi/oswin eventually if we want gui and compute
	runtime.LockOSThread()
}

var TheGPU *egpu.GPU

func main() {
	glfw.Init()
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	vk.Init()

	gp := &egpu.GPU{}
	gp.Init("compute1", true)
	TheGPU = gp

	cp := &egpu.Compute{}
	cp.Init(gp)

	mem := &egpu.Memory{}
	mem.Init(gp, &cp.Device)

	bm := &egpu.VecIdxs{}
	vb := bm.VectorsBuffer()

	pl := &egpu.Pipeline{}
	pl.SetName("compute1")
	pl.InitCompute(cp)

	pr := pl.AddProgram("sqvecel")
	src, err := egpu.OpenFile("sqvecel.spv")
	if err != nil {
		fmt.Println(err)
		exit(0)
	}

	shi, _ := pr.AddShader(gpu.ComputeShader, "sqvecel", string(src))
	sh := shi.(*egpu.Shader)
	sh.Compile(src)

	inv := pr.AddInput("In", gpu.Vec2fVecType, gpu.VertexPosition)
	outv := pr.AddInput("Out", gpu.Vec2fVecType, gpu.VertexPosition)

	n := 20

	vb.AddVectors(inv, false)
	vb.AddVectors(outv, false)
	vb.SetLen(n)
	data := vb.allData()
	for i := 0; i < n; i++ {
		data[i*2+0] = rand.Float32()
		data[i*2+1] = rand.Float32()
	}

	mem.AddBuff(bm)
	mem.Config() // sets everything up

	var pipelineCache vk.PipelineCache
	ret := vk.CreatePipelineCache(dev, &vk.PipelineCacheCreateInfo{
		SType: vk.StructureTypePipelineCacheCreateInfo,
	}, nil, &pipelineCache)
	egpu.IFPanic(as.NewError(ret))

	pipelineCreateInfos := []vk.GraphicsPipelineCreateInfo{{
		SType:      vk.StructureTypeGraphicsPipelineCreateInfo,
		Layout:     s.pipelineLayout,
		RenderPass: s.renderPass,

		PVertexInputState: &vk.PipelineVertexInputStateCreateInfo{
			SType: vk.StructureTypePipelineVertexInputStateCreateInfo,
		},
		PInputAssemblyState: &vk.PipelineInputAssemblyStateCreateInfo{
			SType:    vk.StructureTypePipelineInputAssemblyStateCreateInfo,
			Topology: vk.PrimitiveTopologyTriangleList,
		},
		StageCount: 2,
		PStages: []vk.PipelineShaderStageCreateInfo{{
			SType:  vk.StructureTypePipelineShaderStageCreateInfo,
			Stage:  vk.ShaderStageVertexBit,
			Module: vs,
			PName:  "main\x00",
		}},
	}}

	pipeline := make([]vk.Pipeline, 1)
	ret = vk.CreateGraphicsPipelines(dev, s.pipelineCache, 1, pipelineCreateInfos, nil, pipeline)
	orPanic(as.NewError(ret))
	s.pipeline = pipeline[0]

	vk.DestroyShaderModule(dev, vs, nil)
	vk.DestroyShaderModule(dev, fs, nil)

	var descPool vk.DescriptorPool
	ret := vk.CreateDescriptorPool(dev, &vk.DescriptorPoolCreateInfo{
		SType:         vk.StructureTypeDescriptorPoolCreateInfo,
		MaxSets:       uint32(len(swapchainImageResources)),
		PoolSizeCount: 2,
		PPoolSizes: []vk.DescriptorPoolSize{{
			Type:            vk.DescriptorTypeUniformBuffer,
			DescriptorCount: uint32(len(swapchainImageResources)),
		}, {
			Type:            vk.DescriptorTypeCombinedImageSampler,
			DescriptorCount: uint32(len(swapchainImageResources) * len(texEnabled)),
		}},
	}, nil, &descPool)
	IFPanic(as.NewError(ret))

	var set vk.DescriptorSet
	ret := vk.AllocateDescriptorSets(dev, &vk.DescriptorSetAllocateInfo{
		SType:              vk.StructureTypeDescriptorSetAllocateInfo,
		DescriptorPool:     s.descPool,
		DescriptorSetCount: 1,
		PSetLayouts:        []vk.DescriptorSetLayout{s.descLayout},
	}, &set)
	orPanic(as.NewError(ret))

	res.SetDescriptorSet(set)

	vk.UpdateDescriptorSets(dev, 2, []vk.WriteDescriptorSet{{
		SType:           vk.StructureTypeWriteDescriptorSet,
		DstSet:          set,
		DescriptorCount: 1,
		DescriptorType:  vk.DescriptorTypeUniformBuffer,
		PBufferInfo: []vk.DescriptorBufferInfo{{
			Offset: 0,
			Range:  vk.DeviceSize(vkTexCubeUniformSize),
			Buffer: res.UniformBuffer(),
		}},
	}, {
		SType:           vk.StructureTypeWriteDescriptorSet,
		DstBinding:      1,
		DstSet:          set,
		DescriptorCount: uint32(len(texEnabled)),
		DescriptorType:  vk.DescriptorTypeCombinedImageSampler,
		PImageInfo:      texInfos,
	}}, 0, nil)

}

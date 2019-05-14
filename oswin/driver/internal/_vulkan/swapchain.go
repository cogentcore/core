// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based extensively on vulkan-go/asche
// The MIT License (MIT)
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>

package gpu

import "github.com/vulkan-go/vulkan"

// SwapchainDimensions describes the size and format of the swapchain.
type SwapchainDimensions struct {
	// Width of the swapchain.
	Width uint32
	// Height of the swapchain.
	Height uint32
	// Format is the pixel format of the swapchain.
	Format vulkan.Format
}

// SwapchainImageResources holds onto image resources
type SwapchainImageResources struct {
	image                vulkan.Image
	cmd                  vulkan.CommandBuffer
	graphicsToPresentCmd vulkan.CommandBuffer

	view          vulkan.ImageView
	framebuffer   vulkan.Framebuffer
	descriptorSet vulkan.DescriptorSet

	uniformBuffer vulkan.Buffer
	uniformMemory vulkan.DeviceMemory
}

func (s *SwapchainImageResources) Destroy(dev vulkan.Device, cmdPool ...vulkan.CommandPool) {
	vulkan.DestroyFramebuffer(dev, s.framebuffer, nil)
	vulkan.DestroyImageView(dev, s.view, nil)
	if len(cmdPool) > 0 {
		vulkan.FreeCommandBuffers(dev, cmdPool[0], 1, []vulkan.CommandBuffer{
			s.cmd,
		})
	}
	vulkan.DestroyBuffer(dev, s.uniformBuffer, nil)
	vulkan.FreeMemory(dev, s.uniformMemory, nil)
}

func (s *SwapchainImageResources) SetImageOwnership(graphicsQueueFamilyIndex, presentQueueFamilyIndex uint32) {
	ret := vulkan.BeginCommandBuffer(s.graphicsToPresentCmd, &vulkan.CommandBufferBeginInfo{
		SType: vulkan.StructureTypeCommandBufferBeginInfo,
		Flags: vulkan.CommandBufferUsageFlags(vulkan.CommandBufferUsageSimultaneousUseBit),
	})
	orPanic(NewError(ret))

	vulkan.CmdPipelineBarrier(s.graphicsToPresentCmd,
		vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit),
		vulkan.PipelineStageFlags(vulkan.PipelineStageColorAttachmentOutputBit),
		0, 0, nil, 0, nil, 1, []vulkan.ImageMemoryBarrier{{
			SType:               vulkan.StructureTypeImageMemoryBarrier,
			DstAccessMask:       vulkan.AccessFlags(vulkan.AccessColorAttachmentWriteBit),
			OldLayout:           vulkan.ImageLayoutPresentSrc,
			NewLayout:           vulkan.ImageLayoutPresentSrc,
			SrcQueueFamilyIndex: graphicsQueueFamilyIndex,
			DstQueueFamilyIndex: presentQueueFamilyIndex,
			Image:               s.image,

			SubresourceRange: vulkan.ImageSubresourceRange{
				AspectMask: vulkan.ImageAspectFlags(vulkan.ImageAspectColorBit),
				LevelCount: 1,
				LayerCount: 1,
			},
		}})

	ret = vulkan.EndCommandBuffer(s.graphicsToPresentCmd)
	orPanic(NewError(ret))
}

func (s *SwapchainImageResources) SetUniformBuffer(buffer vulkan.Buffer, mem vulkan.DeviceMemory) {
	s.uniformBuffer = buffer
	s.uniformMemory = mem
}

func (s *SwapchainImageResources) Framebuffer() vulkan.Framebuffer {
	return s.framebuffer
}

func (s *SwapchainImageResources) SetFramebuffer(fb vulkan.Framebuffer) {
	s.framebuffer = fb
}

func (s *SwapchainImageResources) UniformBuffer() vulkan.Buffer {
	return s.uniformBuffer
}

func (s *SwapchainImageResources) UniformMemory() vulkan.DeviceMemory {
	return s.uniformMemory
}

func (s *SwapchainImageResources) CommandBuffer() vulkan.CommandBuffer {
	return s.cmd
}

func (s *SwapchainImageResources) Image() vulkan.Image {
	return s.image
}

func (s *SwapchainImageResources) View() vulkan.ImageView {
	return s.view
}

func (s *SwapchainImageResources) DescriptorSet() vulkan.DescriptorSet {
	return s.descriptorSet
}

func (s *SwapchainImageResources) SetDescriptorSet(set vulkan.DescriptorSet) {
	s.descriptorSet = set
}

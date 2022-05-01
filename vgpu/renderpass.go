// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"image"

	vk "github.com/vulkan-go/vulkan"
)

// RenderPass contains a vulkan RenderPass object,
// which specifies parameters for rendering to a Framebuffer.
// It can hold the Depth buffer if one is used.
// In general, there should be one RenderPass object for
// each Pipeline, and any associated Framebuffers
// include the RenderPass info. and its Depth buffer.
type RenderPass struct {
	Dev        vk.Device     `desc:"the device we're associated with -- this must be the same device that owns the Framebuffer -- e.g., the Surface"`
	Format     ImageFormat   `desc:"image format information for the framebuffer we render to"`
	RenderPass vk.RenderPass `desc:"the vulkan renderpass handle"`
	Depth      Image         `desc:"the associated depth buffer, if set"`
}

func (rp *RenderPass) Destroy() {
	if rp.RenderPass == nil {
		return
	}
	vk.DestroyRenderPass(rp.Dev, rp.RenderPass, nil)
	rp.RenderPass = nil
	rp.Depth.Destroy()
}

// Config configures the render pass for given device,
// Using standard parameters for graphics rendering,
// based on the given image format and depth image format
// (pass FormatUnknown for no depth buffer).
func (rp *RenderPass) Config(dev vk.Device, imgFmt *ImageFormat, depthFmt vk.Format) {
	// The initial layout for the color and depth attachments will be vk.LayoutUndefined
	// because at the start of the renderpass, we don't care about their contents.
	// At the start of the subpass, the color attachment's layout will be transitioned
	// to vk.LayoutColorAttachmentOptimal and the depth stencil attachment's layout
	// will be transitioned to vk.LayoutDepthStencilAttachmentOptimal.  At the end of
	// the renderpass, the color attachment's layout will be transitioned to
	// vk.LayoutPresentSrc to be ready to present.  This is all done as part of
	// the renderpass, no barriers are necessary.
	rp.Dev = dev
	rp.Format = *imgFmt
	rp.Depth.Format.Format = depthFmt
	var renderPass vk.RenderPass
	if rp.Depth.Format.Format != vk.FormatUndefined {
		ret := vk.CreateRenderPass(dev, &vk.RenderPassCreateInfo{
			SType:           vk.StructureTypeRenderPassCreateInfo,
			AttachmentCount: 2,
			PAttachments: []vk.AttachmentDescription{{
				Format:         rp.Format.Format,
				Samples:        rp.Format.Samples,
				LoadOp:         vk.AttachmentLoadOpClear,
				StoreOp:        vk.AttachmentStoreOpStore,
				StencilLoadOp:  vk.AttachmentLoadOpDontCare,
				StencilStoreOp: vk.AttachmentStoreOpDontCare,
				InitialLayout:  vk.ImageLayoutUndefined,
				FinalLayout:    vk.ImageLayoutPresentSrc,
			}, {
				Format:         rp.Depth.Format.Format,
				Samples:        rp.Format.Samples,
				LoadOp:         vk.AttachmentLoadOpClear,
				StoreOp:        vk.AttachmentStoreOpDontCare,
				StencilLoadOp:  vk.AttachmentLoadOpDontCare,
				StencilStoreOp: vk.AttachmentStoreOpDontCare,
				InitialLayout:  vk.ImageLayoutUndefined,
				FinalLayout:    vk.ImageLayoutDepthStencilAttachmentOptimal,
			}},
			SubpassCount: 1,
			PSubpasses: []vk.SubpassDescription{{
				PipelineBindPoint:    vk.PipelineBindPointGraphics,
				ColorAttachmentCount: 1,
				PColorAttachments: []vk.AttachmentReference{{
					Attachment: 0,
					Layout:     vk.ImageLayoutColorAttachmentOptimal,
				}},
				PDepthStencilAttachment: &vk.AttachmentReference{
					Attachment: 1,
					Layout:     vk.ImageLayoutDepthStencilAttachmentOptimal,
				},
			}},
		}, nil, &renderPass)
		IfPanic(NewError(ret))
	} else {
		ret := vk.CreateRenderPass(dev, &vk.RenderPassCreateInfo{
			SType:           vk.StructureTypeRenderPassCreateInfo,
			AttachmentCount: 1,
			PAttachments: []vk.AttachmentDescription{{
				Format:         rp.Format.Format,
				Samples:        rp.Format.Samples,
				LoadOp:         vk.AttachmentLoadOpClear,
				StoreOp:        vk.AttachmentStoreOpStore,
				StencilLoadOp:  vk.AttachmentLoadOpDontCare,
				StencilStoreOp: vk.AttachmentStoreOpDontCare,
				InitialLayout:  vk.ImageLayoutUndefined,
				FinalLayout:    vk.ImageLayoutPresentSrc,
			}},
			SubpassCount: 1,
			PSubpasses: []vk.SubpassDescription{{
				PipelineBindPoint:    vk.PipelineBindPointGraphics,
				ColorAttachmentCount: 1,
				PColorAttachments: []vk.AttachmentReference{{
					Attachment: 0,
					Layout:     vk.ImageLayoutColorAttachmentOptimal,
				}},
			}},
		}, nil, &renderPass)
		IfPanic(NewError(ret))
	}
	rp.RenderPass = renderPass
}

// SetDepthSize sets size of the Depth buffer, allocating a new one as needed
func (rp *RenderPass) SetDepthSize(size image.Point) {
	rp.Depth.SetSize(size)
}

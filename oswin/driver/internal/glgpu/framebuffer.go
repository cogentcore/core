// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"fmt"
	"image"
	"log"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/goki/gi/oswin/gpu"
)

// Framebuffer is an offscreen render target.
// gi3d renders exclusively to a Framebuffer, which is then copied to
// the overall oswin.Window.WinTex texture that backs the window for
// final display to the user.
type Framebuffer struct {
	init     bool
	handle   uint32
	name     string
	size     image.Point
	nsamp    int
	tex      gpu.Texture2D // externally-provided texture
	drbo     uint32        // depth render buffer object
	cTex     gpu.Texture2D // internal color-buffer texture returned from Texture()
	msampTex uint32        // multi-sampled color texture when not using external tex
	dsampFbo uint32        // down-sampling fbo
	depthTex uint32        // depth float32 texture, either for downsample or nsamp = 0 case
	depthDat []float32     // depth float32 data transferred from GPU
	texOld   bool          // texture is out-of-date since last render
	depthOld bool          // depth data is out-of-date since last render
}

// Name returns the name of the framebuffer
func (fb *Framebuffer) Name() string {
	return fb.name
}

// SetName sets the name of the framebuffer
func (fb *Framebuffer) SetName(name string) {
	fb.name = name
}

// Size returns the size of the framebuffer
func (fb *Framebuffer) Size() image.Point {
	return fb.size
}

// SetSize sets the size of the framebuffer.
// If framebuffer has been Activate'd, then this resizes the GPU side as well,
// and if a texture has been linked to this Framebuffer with SetTexture then
// it is also resized with SetSize.
func (fb *Framebuffer) SetSize(size image.Point) {
	if fb.size == size {
		return
	}
	wasInit := fb.init
	if fb.init {
		fb.Delete()
	}
	fb.size = size
	if fb.tex != nil {
		fb.tex.SetSize(size)
	}
	if wasInit {
		fb.Activate()
	}
}

// Bounds returns the bounds of the framebuffer's image. It is equal to
// image.Rectangle{Max: t.Size()}.
func (fb *Framebuffer) Bounds() image.Rectangle {
	return image.Rectangle{Max: fb.size}
}

// SetSamples sets the number of samples to use for multi-sample
// anti-aliasing.  When using as a primary 3D render target,
// the recommended number is 4 to produce much better-looking results.
// If just using as an intermediate render buffer, then you may
// want to turn off by setting to 0.
// Setting to a number > 0 automatically disables use of external
// Texture2D that might have previously been set by SetTexture --
// must call Texture() to get downsampled texture instead.
func (fb *Framebuffer) SetSamples(samples int) {
	if fb.nsamp == samples {
		return
	}
	wasInit := fb.init
	fb.nsamp = samples
	if samples > 0 && fb.tex != nil {
		fb.tex = nil
	}
	if wasInit {
		fb.Activate()
	}
}

// Samples returns the number of multi-sample samples
func (fb *Framebuffer) Samples() int {
	return fb.nsamp
}

// SetTexture sets an existing Texture2D to serve as the color buffer target
// for this framebuffer.  This also implies SetSamples(0), and that
// the Texture() method just directly returns the texture set here.
// If we have a non-zero size, then the texture is automatically resized
// to the size of the framebuffer, otherwise the fb inherits size of texture.
func (fb *Framebuffer) SetTexture(tex gpu.Texture2D) {
	fb.tex = tex
	if fb.tex == nil {
		return
	}
	fb.nsamp = 0
	if fb.size == image.ZP {
		tsz := tex.Size()
		if tsz != image.ZP {
			fb.SetSize(tsz)
		}
	} else {
		tex.SetSize(fb.size)
	}
}

// Texture returns the current contents of the framebuffer as a Texture2D.
// Returns nil if not activated.
// For Samples() > 0 this reduces the optimized internal render buffer to a
// standard 2D texture -- the return texture is owned and managed by the
// framebuffer, and re-used every time Texture() is called.
// If SetTexture was called, then it just returns that texture
// which was directly rendered to.
func (fb *Framebuffer) Texture() gpu.Texture2D {
	if fb.tex != nil {
		return fb.tex
	}
	if !fb.init {
		return nil
	}
	if fb.nsamp > 0 && fb.texOld {
		szx := int32(fb.size.X)
		szy := int32(fb.size.Y)
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, fb.handle)
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, fb.dsampFbo)
		gl.BlitFramebuffer(0, 0, szx, szy, 0, 0, szx, szy, gl.COLOR_BUFFER_BIT|gl.DEPTH_BUFFER_BIT, gl.NEAREST)
		// copies into cTex and depthTex
	}
	fb.texOld = false
	return fb.cTex
}

// grabDepth gets the depth data from gpu
func (fb *Framebuffer) grabDepth() {
	if fb.nsamp > 0 {
		fb.Texture() // update if needed
	}
	if fb.depthOld {
		totn := fb.size.X * fb.size.Y
		if len(fb.depthDat) != totn {
			fb.depthDat = make([]float32, totn)
		}
		gl.BindTexture(gl.TEXTURE_2D, fb.depthTex)
		gl.GetTexImage(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT, gl.FLOAT, gl.Ptr(fb.depthDat))
		gpu.TheGPU.ErrCheck("grabDepth")
		fb.depthOld = false
	}
}

// DepthAt returns the depth-buffer value at given x,y coordinate in
// framebuffer coordinates (i.e., Y = 0 is at bottom).
// Must be called with a valid gpu context and on proper thread for that context,
// with framebuffer active.
func (fb *Framebuffer) DepthAt(x, y int) float32 {
	if !fb.init {
		return 0
	}
	fb.grabDepth()
	i := y*fb.size.X + x
	d := fb.depthDat[i]
	return d
}

// DepthAll returns the entire depth buffer as a slice of float32 values
// of same size as framebuffer.  This slice is pointer to internal reused
// value -- copy to retain values or modify.
// Must be called with a valid gpu context and on proper thread for that context,
// with framebuffer active.
func (fb *Framebuffer) DepthAll() []float32 {
	if !fb.init {
		return nil
	}
	fb.grabDepth()
	return fb.depthDat
}

// Activate establishes the GPU resources and handle for the
// framebuffer and all other associated buffers etc (if not already done).
// It then sets this as the current rendering target for subsequent
// gpu drawing commands.
func (fb *Framebuffer) Activate() error {
	if !fb.init {
		fb.texOld = true
		fb.depthOld = true
		szx := int32(fb.size.X)
		szy := int32(fb.size.Y)
		gl.GenFramebuffers(1, &fb.handle)
		gl.BindFramebuffer(gl.FRAMEBUFFER, fb.handle)

		gl.GenRenderbuffers(1, &fb.drbo)
		gl.BindRenderbuffer(gl.RENDERBUFFER, fb.drbo)
		if fb.nsamp > 0 {
			gl.RenderbufferStorageMultisample(gl.RENDERBUFFER, int32(fb.nsamp), gl.DEPTH_COMPONENT32F, szx, szy)
			// gpu.TheGPU.ErrCheck("framebuffer storage multisamp")
		} else {
			gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT32F, szx, szy)
		}
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, fb.drbo)
		if fb.tex != nil {
			gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.tex.Handle(), 0)
		} else {
			fb.cTex = gpu.TheGPU.NewTexture2D(fmt.Sprintf("fb-%s-ctex", fb.name))
			fb.cTex.SetSize(fb.size)
			fb.cTex.Activate(0)
			if fb.nsamp > 0 {
				gl.GenTextures(1, &fb.msampTex)
				gl.BindTexture(gl.TEXTURE_2D_MULTISAMPLE, fb.msampTex)
				gl.TexImage2DMultisample(gl.TEXTURE_2D_MULTISAMPLE, int32(fb.nsamp), gl.RGBA, szx, szy, true)
				// gpu.TheGPU.ErrCheck("framebuffer teximage 2d multisamp")
				gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D_MULTISAMPLE, fb.msampTex, 0)
				// gpu.TheGPU.ErrCheck("framebuffer texture2d")

				// downsampling fbo
				gl.GenFramebuffers(1, &fb.dsampFbo)
				gl.BindFramebuffer(gl.FRAMEBUFFER, fb.dsampFbo)
				gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.cTex.Handle(), 0)

				gl.GenTextures(1, &fb.depthTex)
				gl.BindTexture(gl.TEXTURE_2D, fb.depthTex)
				gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT32F, szx, szy, 0, gl.DEPTH_COMPONENT, gl.FLOAT, gl.Ptr(nil))
				gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, fb.depthTex, 0)

				gl.BindFramebuffer(gl.FRAMEBUFFER, fb.handle)
			} else {
				gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fb.cTex.Handle(), 0)
			}
		}
		fb.init = true
	} else {
		gl.BindFramebuffer(gl.FRAMEBUFFER, fb.handle)
	}
	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		err := fmt.Errorf("glgpu gpu.Framebuffer: %s not complete -- this usually means you need to set your GoGi prefs to Smooth3D = off, and restart", fb.name)
		log.Println(err)
		return err
	}
	gl.Viewport(0, 0, int32(fb.size.X), int32(fb.size.Y))
	return nil
}

// Handle returns the GPU handle for the framebuffer -- only
// valid after Activate.
func (fb *Framebuffer) Handle() uint32 {
	return fb.handle
}

// Rendered should be called after rendering to the framebuffer,
// to ensure the update of data transferred from the framebuffer
// (texture, depth buffer)
func (fb *Framebuffer) Rendered() {
	fb.texOld = true
	fb.depthOld = true
}

// Delete deletes the GPU resources associated with this framebuffer
// (requires Activate to re-establish a new one).
// Should be called prior to Go object being deleted
// (ref counting can be done externally).
// Does NOT delete any Texture set by SetTexture -- must be done externally.
func (fb *Framebuffer) Delete() {
	if fb.init {
		if fb.cTex != nil {
			fb.cTex.Delete()
			fb.cTex = nil
		}
		if fb.nsamp > 0 {
			gl.DeleteFramebuffers(1, &fb.dsampFbo)
		}
		gl.DeleteRenderbuffers(1, &fb.drbo)
		gl.DeleteFramebuffers(1, &fb.handle)
		fb.init = false
	}
}

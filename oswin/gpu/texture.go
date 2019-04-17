// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/goki/gi/oswin"

// Texture2D manages a 2D texture, including loading from an image file
// and activating on GPU.  Because a Texture2D is used for rendering to an
// oswin.Window, the oswin.Texture interface defines everything at that
// level, and gpu.Texture2D is just effectively an alias to that same
// interface.
//
// For greater clarity, please use the gpu.Texture2D interface for all
// GPU-specific code, and oswin.Texture for oswin-specific code.
//
type Texture2D interface {
	oswin.Texture

	// Framebuffer returns a framebuffer for rendering onto this
	// texture -- calls ActivateFramebuffer() if one is not
	// already activated.
	Framebuffer() Framebuffer
}

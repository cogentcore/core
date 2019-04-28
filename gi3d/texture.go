// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin/gpu"
)

// TexName provides a GUI interface for choosing textures
type TexName string

// Texture is a texture material -- any objects using the same texture can be rendered
// at the same time.  This is a static texture.
type Texture struct {
	Name string        `desc:"name of the texture -- textures are connected to material / objects by name"`
	File gi.FileName   `desc:"filename for the texture"`
	Tex  gpu.Texture2D `view:"-" desc:"gpu texture object"`
}

// TextureGi2D is a dynamic texture material driven by a gi.Viewport2D viewport
// anything rendered to the viewport will be projected onto the surface of any
// object using this texture.
type TextureGi2D struct {
	Texture
	Viewport *gi.Viewport2D
}

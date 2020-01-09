// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/gi/gi"
	"github.com/goki/ki/kit"
)

// StyleMatFuncs are functions for styling the Material
var StyleMatFuncs = map[string]gi.StyleFunc{
	"color": func(obj interface{}, key string, val interface{}, par interface{}, vp *gi.Viewport2D) {
		mt := obj.(*Material)
		if inh, init := gi.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Color = par.(*Material).Color
			} else if init {
				mt.Color.SetUInt8(128, 128, 128, 255)
			}
			return
		}
		mt.Color.SetIFace(val, vp, key)
	},
	"emissive": func(obj interface{}, key string, val interface{}, par interface{}, vp *gi.Viewport2D) {
		mt := obj.(*Material)
		if inh, init := gi.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Emissive = par.(*Material).Emissive
			} else if init {
				mt.Emissive.SetUInt8(0, 0, 0, 0)
			}
			return
		}
		mt.Emissive.SetIFace(val, vp, key)
	},
	"specular": func(obj interface{}, key string, val interface{}, par interface{}, vp *gi.Viewport2D) {
		mt := obj.(*Material)
		if inh, init := gi.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Specular = par.(*Material).Specular
			} else if init {
				mt.Specular.SetUInt8(255, 255, 255, 255)
			}
			return
		}
		mt.Specular.SetIFace(val, vp, key)
	},
	"shiny": func(obj interface{}, key string, val interface{}, par interface{}, vp *gi.Viewport2D) {
		mt := obj.(*Material)
		if inh, init := gi.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Shiny = par.(*Material).Shiny
			} else if init {
				mt.Shiny = 30
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			mt.Shiny = iv
		}
	},
	"bright": func(obj interface{}, key string, val interface{}, par interface{}, vp *gi.Viewport2D) {
		mt := obj.(*Material)
		if inh, init := gi.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Bright = par.(*Material).Bright
			} else if init {
				mt.Bright = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			mt.Bright = iv
		}
	},
	"texture": func(obj interface{}, key string, val interface{}, par interface{}, vp *gi.Viewport2D) {
		mt := obj.(*Material)
		if inh, init := gi.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Texture = par.(*Material).Texture
			} else if init {
				mt.Texture = ""
			}
			return
		}
		mt.Texture = TexName(kit.ToString(val))
	},
	"cull-back": func(obj interface{}, key string, val interface{}, par interface{}, vp *gi.Viewport2D) {
		mt := obj.(*Material)
		if inh, init := gi.StyleInhInit(val, par); inh || init {
			if inh {
				mt.CullBack = par.(*Material).CullBack
			} else if init {
				mt.CullBack = true
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			mt.CullBack = bv
		}
	},
	"cull-front": func(obj interface{}, key string, val interface{}, par interface{}, vp *gi.Viewport2D) {
		mt := obj.(*Material)
		if inh, init := gi.StyleInhInit(val, par); inh || init {
			if inh {
				mt.CullFront = par.(*Material).CullFront
			} else if init {
				mt.CullFront = false
			}
			return
		}
		if bv, ok := kit.ToBool(val); ok {
			mt.CullFront = bv
		}
	},
}

// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"image/color"
	"strings"

	"github.com/goki/colors"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"goki.dev/gi/gi"
	"goki.dev/gi/gist"
)

// SetMatProps sets Material values based on ki.Props properties
func (mt *Material) SetMatProps(par *Material, props ki.Props, vp *gi.Viewport2D) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		if sfunc, ok := StyleMatFuncs[key]; ok {
			sfunc(mt, key, val, par, vp)
			continue
		}
	}
}

// todo: could generalize this logic and pass in functions.

// ApplyCSS applies css styles for given node, using key to select sub-props
// from overall properties list, and optional selector to select a further
// :name selector within that key
func (mt *Material) ApplyCSS(node Node3D, css ki.Props, key, selector string, vp *gi.Viewport2D) bool {
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}
	if selector != "" {
		pmap, ok = gist.SubProps(pmap, selector)
		if !ok {
			return false
		}
	}
	mt.SetMatProps(nil, pmap, vp)
	return true
}

// StyleCSS applies css style properties to given node, parsing out
// type, .class, and #name selectors, along with optional sub-selector
// (:hover, :active etc)
func (mt *Material) StyleCSS(node Node3D, css ki.Props, selector string, vp *gi.Viewport2D) {
	tyn := strings.ToLower(ki.Type(node).Name()) // type is most general, first
	mt.ApplyCSS(node, css, tyn, selector, vp)
	classes := strings.Split(strings.ToLower(node.AsNode3D().Class), " ")
	for _, cl := range classes {
		cln := "." + strings.TrimSpace(cl)
		mt.ApplyCSS(node, css, cln, selector, vp)
	}
	idnm := "#" + strings.ToLower(node.Name()) // then name
	mt.ApplyCSS(node, css, idnm, selector, vp)
}

// StyleMatFuncs are functions for styling the Material
var StyleMatFuncs = map[string]gist.StyleFunc{
	"color": func(obj any, key string, val any, par any, ctxt gist.Context) {
		mt := obj.(*Material)
		if inh, init := gist.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Color = par.(*Material).Color
			} else if init {
				mt.Color = colors.FromRGB(128, 128, 128)
			}
			return
		}
		mt.Color = colors.LogFromAny(val, ctxt.ContextColor())
	},
	"emissive": func(obj any, key string, val any, par any, ctxt gist.Context) {
		mt := obj.(*Material)
		if inh, init := gist.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Emissive = par.(*Material).Emissive
			} else if init {
				mt.Emissive = color.RGBA{}
			}
			return
		}
		mt.Emissive = colors.LogFromAny(val, ctxt.ContextColor())
	},
	"shiny": func(obj any, key string, val any, par any, ctxt gist.Context) {
		mt := obj.(*Material)
		if inh, init := gist.StyleInhInit(val, par); inh || init {
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
	"reflective": func(obj any, key string, val any, par any, ctxt gist.Context) {
		mt := obj.(*Material)
		if inh, init := gist.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Reflective = par.(*Material).Reflective
			} else if init {
				mt.Reflective = 1
			}
			return
		}
		if iv, ok := kit.ToFloat32(val); ok {
			mt.Reflective = iv
		}
	},
	"bright": func(obj any, key string, val any, par any, ctxt gist.Context) {
		mt := obj.(*Material)
		if inh, init := gist.StyleInhInit(val, par); inh || init {
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
	"texture": func(obj any, key string, val any, par any, ctxt gist.Context) {
		mt := obj.(*Material)
		if inh, init := gist.StyleInhInit(val, par); inh || init {
			if inh {
				mt.Texture = par.(*Material).Texture
			} else if init {
				mt.Texture = ""
			}
			return
		}
		mt.Texture = TexName(kit.ToString(val))
	},
	"cull-back": func(obj any, key string, val any, par any, ctxt gist.Context) {
		mt := obj.(*Material)
		if inh, init := gist.StyleInhInit(val, par); inh || init {
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
	"cull-front": func(obj any, key string, val any, par any, ctxt gist.Context) {
		mt := obj.(*Material)
		if inh, init := gist.StyleInhInit(val, par); inh || init {
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

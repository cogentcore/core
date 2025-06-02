// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/styleprops"
)

// FromProperties sets style field values based on the given property list.
func (s *Style) FromProperties(parent *Style, properties map[string]any, ctxt colors.Context) {
	for key, val := range properties {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		s.FromProperty(parent, key, val, ctxt)
	}
}

// FromProperty sets style field values based on the given property key and value.
func (s *Style) FromProperty(parent *Style, key string, val any, cc colors.Context) {
	if sfunc, ok := styleFuncs[key]; ok {
		if parent != nil {
			sfunc(s, key, val, parent, cc)
		} else {
			sfunc(s, key, val, nil, cc)
		}
		return
	}
}

// FontSizePoints maps standard font names to standard point sizes -- we use
// dpi zoom scaling instead of rescaling "medium" font size, so generally use
// these values as-is.  smaller and larger relative scaling can move in 2pt increments
var FontSizes = map[string]float32{
	"xx-small": 6.0 / 12.0,
	"x-small":  8.0 / 12.0,
	"small":    10.0 / 12.0, // small is also "smaller"
	"smallf":   10.0 / 12.0, // smallf = small font size..
	"medium":   1,
	"large":    14.0 / 12.0,
	"x-large":  18.0 / 12.0,
	"xx-large": 24.0 / 12.0,
}

// styleFuncs are functions for styling the rich.Style object.
var styleFuncs = map[string]styleprops.Func{
	// note: text.Style handles the standard units-based font-size settings
	"font-size": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Size = parent.(*Style).Size
			} else if init {
				fs.Size = 1.0
			}
			return
		}
		switch vt := val.(type) {
		case string:
			if psz, ok := FontSizes[vt]; ok {
				fs.Size = psz
			}
		}
	},
	"font-family": styleprops.Enum(SansSerif,
		func(obj *Style) enums.EnumSetter { return &obj.Family }),
	"font-style": styleprops.Enum(SlantNormal,
		func(obj *Style) enums.EnumSetter { return &obj.Slant }),
	"font-weight": styleprops.Enum(Normal,
		func(obj *Style) enums.EnumSetter { return &obj.Weight }),
	"font-stretch": styleprops.Enum(StretchNormal,
		func(obj *Style) enums.EnumSetter { return &obj.Stretch }),
	"text-decoration": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		cflags := fs.Decoration & colorFlagsMask
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Decoration = parent.(*Style).Decoration
			} else if init {
				fs.Decoration = cflags
			}
			return
		}
		switch vt := val.(type) {
		case string:
			if vt == "none" {
				fs.Decoration = cflags
			} else {
				fs.Decoration.SetString(vt)
				fs.Decoration |= cflags
			}
		case Decorations:
			fs.Decoration = cflags | vt
		default:
			iv, err := reflectx.ToInt(val)
			if err == nil {
				fs.Decoration = cflags | Decorations(iv)
			} else {
				styleprops.SetError(key, val, err)
			}
		}
	},
	"direction": styleprops.Enum(LTR,
		func(obj *Style) enums.EnumSetter { return &obj.Direction }),
	"color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.SetFillColor(parent.(*Style).FillColor())
			} else if init {
				fs.SetFillColor(nil)
			}
			return
		}
		fs.SetFillColor(colors.ToUniform(errors.Log1(gradient.FromAny(val, cc))))
	},
	"stroke-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.SetStrokeColor(parent.(*Style).StrokeColor())
			} else if init {
				fs.SetStrokeColor(nil)
			}
			return
		}
		fs.SetStrokeColor(colors.ToUniform(errors.Log1(gradient.FromAny(val, cc))))
	},
	"background-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.SetBackground(parent.(*Style).Background())
			} else if init {
				fs.SetBackground(nil)
			}
			return
		}
		fs.SetBackground(colors.ToUniform(errors.Log1(gradient.FromAny(val, cc))))
	},
	"opacity": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			return
		}
		fv, _ := reflectx.ToFloat(val)
		if fv < 1 {
			if fs.background != nil {
				fs.background = colors.ApplyOpacity(fs.background, float32(fv))
			}
			if fs.strokeColor != nil {
				fs.strokeColor = colors.ApplyOpacity(fs.strokeColor, float32(fv))
			}
			if fs.fillColor != nil {
				fs.fillColor = colors.ApplyOpacity(fs.fillColor, float32(fv))
			}
		}
	},
}

// SetFromHTMLTag sets the styling parameters for simple HTML style tags.
// Returns true if handled.
func (s *Style) SetFromHTMLTag(tag string) bool {
	did := false
	switch tag {
	case "b", "strong":
		s.Weight = Bold
		did = true
	case "i", "em", "var", "cite":
		s.Slant = Italic
		did = true
	case "ins":
		fallthrough
	case "u":
		s.Decoration.SetFlag(true, Underline)
		did = true
	case "s", "del", "strike":
		s.Decoration.SetFlag(true, LineThrough)
		did = true
	case "small":
		s.Size = 0.8
		did = true
	case "big":
		s.Size = 1.2
		did = true
	case "xx-small", "x-small", "smallf", "medium", "large", "x-large", "xx-large":
		s.Size = FontSizes[tag]
		did = true
	case "mark":
		s.SetBackground(colors.ToUniform(colors.Scheme.Warn.Container))
		did = true
	case "abbr", "acronym":
		s.Decoration.SetFlag(true, DottedUnderline)
		did = true
	case "tt", "kbd", "samp", "code":
		s.Family = Monospace
		s.SetBackground(colors.ToUniform(colors.Scheme.SurfaceContainer))
		did = true
	}
	return did
}

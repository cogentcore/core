// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/styleprops"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// These functions set styles from map[string]any which are used for styling

// StyleFromProperty sets style field values based on the given property key and value
func (s *Style) StyleFromProperty(parent *Style, key string, val any, cc colors.Context) {
	var pfont *rich.Style
	var ptext *text.Style
	if parent != nil {
		pfont = &parent.Font
		ptext = &parent.Text
	}
	s.Font.StyleFromProperty(pfont, key, val, cc)
	s.Text.StyleFromProperty(ptext, key, val, cc)
	if sfunc, ok := styleLayoutFuncs[key]; ok {
		if parent != nil {
			sfunc(s, key, val, parent, cc)
		} else {
			sfunc(s, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := styleBorderFuncs[key]; ok {
		if parent != nil {
			sfunc(&s.Border, key, val, &parent.Border, cc)
		} else {
			sfunc(&s.Border, key, val, nil, cc)
		}
		return
	}
	if sfunc, ok := styleStyleFuncs[key]; ok {
		sfunc(s, key, val, parent, cc)
		return
	}
	// doesn't work with multiple shadows
	// if sfunc, ok := StyleShadowFuncs[key]; ok {
	// 	if parent != nil {
	// 		sfunc(&s.BoxShadow, key, val, &par.BoxShadow, cc)
	// 	} else {
	// 		sfunc(&s.BoxShadow, key, val, nil, cc)
	// 	}
	// 	return
	// }
}

////////  Style

// styleStyleFuncs are functions for styling the Style object itself
var styleStyleFuncs = map[string]styleprops.Func{
	"color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Color = parent.(*Style).Color
			} else if init {
				fs.Color = colors.Scheme.OnSurface
			}
			return
		}
		fs.Color = errors.Log1(gradient.FromAny(val, cc))
	},
	"background-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		fs := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				fs.Background = parent.(*Style).Background
			} else if init {
				fs.Background = nil
			}
			return
		}
		fs.Background = errors.Log1(gradient.FromAny(val, cc))
	},
	"opacity": styleprops.Float(float32(1),
		func(obj *Style) *float32 { return &obj.Opacity }),
}

////////  Layout

// styleLayoutFuncs are functions for styling the layout
// style properties; they are still stored on the main style object,
// but they are done separately to improve clarity
var styleLayoutFuncs = map[string]styleprops.Func{
	"display": styleprops.Enum(Flex,
		func(obj *Style) enums.EnumSetter { return &obj.Display }),
	"flex-direction": func(obj any, key string, val, parent any, cc colors.Context) {
		s := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				s.Direction = parent.(*Style).Direction
			} else if init {
				s.Direction = Row
			}
			return
		}
		str := reflectx.ToString(val)
		if str == "row" || str == "row-reverse" {
			s.Direction = Row
		} else {
			s.Direction = Column
		}
	},
	// TODO(kai/styproperties): multi-dim flex-grow
	"flex-grow": styleprops.Float(0, func(obj *Style) *float32 { return &obj.Grow.Y }),
	"wrap": styleprops.Bool(false,
		func(obj *Style) *bool { return &obj.Wrap }),
	"justify-content": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Content }),
	"justify-items": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Items }),
	"justify-self": styleprops.Enum(Auto,
		func(obj *Style) enums.EnumSetter { return &obj.Justify.Self }),
	"align-content": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Content }),
	"align-items": styleprops.Enum(Start,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Items }),
	"align-self": styleprops.Enum(Auto,
		func(obj *Style) enums.EnumSetter { return &obj.Align.Self }),
	"x": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Pos.X }),
	"y": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Pos.Y }),
	"width": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Min.X }),
	"height": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Min.Y }),
	"max-width": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Max.X }),
	"max-height": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.Max.Y }),
	"min-width": styleprops.Units(units.Dp(2),
		func(obj *Style) *units.Value { return &obj.Min.X }),
	"min-height": styleprops.Units(units.Dp(2),
		func(obj *Style) *units.Value { return &obj.Min.Y }),
	"margin": func(obj any, key string, val any, parent any, cc colors.Context) {
		s := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				s.Margin = parent.(*Style).Margin
			} else if init {
				s.Margin.Zero()
			}
			return
		}
		s.Margin.SetAny(val)
	},
	"padding": func(obj any, key string, val any, parent any, cc colors.Context) {
		s := obj.(*Style)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				s.Padding = parent.(*Style).Padding
			} else if init {
				s.Padding.Zero()
			}
			return
		}
		s.Padding.SetAny(val)
	},
	// TODO(kai/styproperties): multi-dim overflow
	"overflow": styleprops.Enum(OverflowAuto,
		func(obj *Style) enums.EnumSetter { return &obj.Overflow.Y }),
	"columns": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.Columns }),
	"row": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.Row }),
	"col": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.Col }),
	"row-span": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.RowSpan }),
	"col-span": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.ColSpan }),
	"z-index": styleprops.Int(int(0),
		func(obj *Style) *int { return &obj.ZIndex }),
	"scrollbar-width": styleprops.Units(units.Value{},
		func(obj *Style) *units.Value { return &obj.ScrollbarWidth }),
}

////////  Border

// styleBorderFuncs are functions for styling the Border object
var styleBorderFuncs = map[string]styleprops.Func{
	// SidesTODO: need to figure out how to get key and context information for side SetAny calls
	// with padding, margin, border, etc
	"border-style": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Style = parent.(*Border).Style
			} else if init {
				bs.Style.Set(BorderSolid)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			bs.Style.SetString(vt)
		case BorderStyles:
			bs.Style.Set(vt)
		case []BorderStyles:
			bs.Style.Set(vt...)
		default:
			iv, err := reflectx.ToInt(val)
			if err == nil {
				bs.Style.Set(BorderStyles(iv))
			} else {
				styleprops.SetError(key, val, err)
			}
		}
	},
	"border-width": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Width = parent.(*Border).Width
			} else if init {
				bs.Width.Zero()
			}
			return
		}
		bs.Width.SetAny(val)
	},
	"border-radius": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Radius = parent.(*Border).Radius
			} else if init {
				bs.Radius.Zero()
			}
			return
		}
		bs.Radius.SetAny(val)
	},
	"border-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Color = parent.(*Border).Color
			} else if init {
				bs.Color.Set(colors.Scheme.Outline)
			}
			return
		}
		// TODO(kai): support side-specific border colors
		bs.Color.Set(errors.Log1(gradient.FromAny(val, cc)))
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Outline

// styleOutlineFuncs are functions for styling the OutlineStyle object
var styleOutlineFuncs = map[string]styleprops.Func{
	"outline-style": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Style = parent.(*Border).Style
			} else if init {
				bs.Style.Set(BorderSolid)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			bs.Style.SetString(vt)
		case BorderStyles:
			bs.Style.Set(vt)
		case []BorderStyles:
			bs.Style.Set(vt...)
		default:
			iv, err := reflectx.ToInt(val)
			if err == nil {
				bs.Style.Set(BorderStyles(iv))
			} else {
				styleprops.SetError(key, val, err)
			}
		}
	},
	"outline-width": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Width = parent.(*Border).Width
			} else if init {
				bs.Width.Zero()
			}
			return
		}
		bs.Width.SetAny(val)
	},
	"outline-radius": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Radius = parent.(*Border).Radius
			} else if init {
				bs.Radius.Zero()
			}
			return
		}
		bs.Radius.SetAny(val)
	},
	"outline-color": func(obj any, key string, val any, parent any, cc colors.Context) {
		bs := obj.(*Border)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				bs.Color = parent.(*Border).Color
			} else if init {
				bs.Color.Set(colors.Scheme.Outline)
			}
			return
		}
		// TODO(kai): support side-specific border colors
		bs.Color.Set(errors.Log1(gradient.FromAny(val, cc)))
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Shadow

// styleShadowFuncs are functions for styling the Shadow object
var styleShadowFuncs = map[string]styleprops.Func{
	"box-shadow.offset-x": styleprops.Units(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.OffsetX }),
	"box-shadow.offset-y": styleprops.Units(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.OffsetY }),
	"box-shadow.blur": styleprops.Units(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.Blur }),
	"box-shadow.spread": styleprops.Units(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.Spread }),
	"box-shadow.color": func(obj any, key string, val any, parent any, cc colors.Context) {
		ss := obj.(*Shadow)
		if inh, init := styleprops.InhInit(val, parent); inh || init {
			if inh {
				ss.Color = parent.(*Shadow).Color
			} else if init {
				ss.Color = colors.Scheme.Shadow
			}
			return
		}
		ss.Color = errors.Log1(gradient.FromAny(val, cc))
	},
	"box-shadow.inset": styleprops.Bool(false,
		func(obj *Shadow) *bool { return &obj.Inset }),
}

// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"log/slog"

	"goki.dev/colors"
	"goki.dev/enums"
	"goki.dev/girl/units"
	"goki.dev/glop/num"
	"goki.dev/grr"
	"goki.dev/laser"
)

// StyleInhInit detects the style values of "inherit" and "initial",
// setting the corresponding bool return values
func StyleInhInit(val, par any) (inh, init bool) {
	if str, ok := val.(string); ok {
		switch str {
		case "inherit":
			return !laser.AnyIsNil(par), false
		case "initial":
			return false, true
		default:
			return false, false
		}
	}
	return false, false
}

// StyleFuncInt returns a style function for any numerical value
func StyleFuncInt[T any, F num.Integer](initVal F, getField func(obj *T) *F) StyleFunc {
	return func(obj any, key string, val any, par any, ctxt colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				*fp = *getField(par.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fv, _ := laser.ToInt(val)
		num.SetNumber(fp, fv)
	}
}

// StyleFuncFloat returns a style function for any numerical value
func StyleFuncFloat[T any, F num.Float](initVal F, getField func(obj *T) *F) StyleFunc {
	return func(obj any, key string, val any, par any, ctxt colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				*fp = *getField(par.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fv, _ := laser.ToFloat(val) // can represent any number, ToFloat is fast type switch
		num.SetNumber(fp, fv)
	}
}

// StyleFuncBool returns a style function for a bool value
func StyleFuncBool[T any](initVal bool, getField func(obj *T) *bool) StyleFunc {
	return func(obj any, key string, val any, par any, ctxt colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				*fp = *getField(par.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fv, _ := laser.ToBool(val)
		*fp = fv
	}
}

// StyleFuncUnits returns a style function for units.Value
func StyleFuncUnits[T any](initVal units.Value, getField func(obj *T) *units.Value) StyleFunc {
	return func(obj any, key string, val any, par any, ctxt colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				*fp = *getField(par.(*T))
			} else if init {
				*fp = initVal
			}
			return
		}
		fp.SetIFace(val, key)
	}
}

// StyleFuncEnum returns a style function for any enum value
func StyleFuncEnum[T any](initVal enums.Enum, getField func(obj *T) enums.EnumSetter) StyleFunc {
	return func(obj any, key string, val any, par any, ctxt colors.Context) {
		fp := getField(obj.(*T))
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fp.SetInt64(getField(par.(*T)).Int64())
			} else if init {
				fp.SetInt64(initVal.Int64())
			}
			return
		}
		if st, ok := val.(string); ok {
			fp.SetString(st)
			return
		}
		if en, ok := val.(enums.Enum); ok {
			fp.SetInt64(en.Int64())
			return
		}
		iv, _ := laser.ToInt(val)
		fp.SetInt64(int64(iv))
	}
}

// is:
// label.SetProp("background-color", "blue")
//
// should be:
// label.Style.Background.Color = colors.Blue
// label.ActStyle contains the actual style values, which reflects any properties that
// have been set via CSS or SetProp, and those set in Style which serves as the starting
// point for styling.

// These functions set styles from map[string]any which are used for styling

// StyleSetError reports that cannot set property of given key with given value due to given error
func StyleSetError(key string, val any, err error) {
	slog.Error("styles.Style: error setting value", "key", key, "value", val, "err", err)
}

type StyleFunc func(obj any, key string, val any, par any, ctxt colors.Context)

// StyleTextFuncs are functions for styling the Text object
var StyleTextFuncs = map[string]StyleFunc{
	"text-align": StyleFuncEnum(AlignStart,
		func(obj *Text) enums.EnumSetter { return &obj.Align }),
	"text-vertical-align": StyleFuncEnum(AlignStart,
		func(obj *Text) enums.EnumSetter { return &obj.AlignV }),
	"text-anchor": StyleFuncEnum(AnchorStart,
		func(obj *Text) enums.EnumSetter { return &obj.Anchor }),
	"letter-spacing": StyleFuncUnits(units.Value{},
		func(obj *Text) *units.Value { return &obj.LetterSpacing }),
	"word-spacing": StyleFuncUnits(units.Value{},
		func(obj *Text) *units.Value { return &obj.WordSpacing }),
	"line-height": StyleFuncUnits(LineHeightNormal,
		func(obj *Text) *units.Value { return &obj.LineHeight }),
	"white-space": StyleFuncEnum(WhiteSpaceNormal,
		func(obj *Text) enums.EnumSetter { return &obj.WhiteSpace }),
	"unicode-bidi": StyleFuncEnum(BidiNormal,
		func(obj *Text) enums.EnumSetter { return &obj.UnicodeBidi }),
	"direction": StyleFuncEnum(LRTB,
		func(obj *Text) enums.EnumSetter { return &obj.Direction }),
	"writing-mode": StyleFuncEnum(LRTB,
		func(obj *Text) enums.EnumSetter { return &obj.WritingMode }),
	"glyph-orientation-vertical": StyleFuncFloat(float32(1),
		func(obj *Text) *float32 { return &obj.OrientationVert }),
	"glyph-orientation-horizontal": StyleFuncFloat(float32(1),
		func(obj *Text) *float32 { return &obj.OrientationHoriz }),
	"text-indent": StyleFuncUnits(units.Value{},
		func(obj *Text) *units.Value { return &obj.Indent }),
	"para-spacing": StyleFuncUnits(units.Value{},
		func(obj *Text) *units.Value { return &obj.ParaSpacing }),
	"tab-size": StyleFuncInt(int(4),
		func(obj *Text) *int { return &obj.TabSize }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Font

// StyleFontFuncs are functions for styling the Font object
var StyleFontFuncs = map[string]StyleFunc{
	"font-size": func(obj any, key string, val any, par any, ctxt colors.Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Size = par.(*Font).Size
			} else if init {
				fs.Size.Set(12, units.UnitPt)
			}
			return
		}
		switch vt := val.(type) {
		case string:
			if psz, ok := FontSizePoints[vt]; ok {
				fs.Size = units.Pt(psz)
			} else {
				fs.Size.SetIFace(val, key) // also processes string
			}
		default:
			fs.Size.SetIFace(val, key)
		}
	},
	"font-family": func(obj any, key string, val any, par any, ctxt colors.Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Family = par.(*Font).Family
			} else if init {
				fs.Family = "" // font has defaults
			}
			return
		}
		fs.Family = laser.ToString(val)
	},
	"font-style": StyleFuncEnum(FontNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Style }),
	"font-weight": StyleFuncEnum(WeightNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Weight }),
	"font-stretch": StyleFuncEnum(FontStrNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Stretch }),
	"font-variant": StyleFuncEnum(FontVarNormal,
		func(obj *Font) enums.EnumSetter { return &obj.Variant }),
	"baseline-shift": StyleFuncEnum(ShiftBaseline,
		func(obj *Font) enums.EnumSetter { return &obj.Shift }),
	"text-decoration": func(obj any, key string, val any, par any, ctxt colors.Context) {
		fs := obj.(*Font)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Deco = par.(*Font).Deco
			} else if init {
				fs.Deco = DecoNone
			}
			return
		}
		switch vt := val.(type) {
		case string:
			if vt == "none" {
				fs.Deco = DecoNone
			} else {
				fs.Deco.SetString(vt)
			}
		case TextDecorations:
			fs.Deco = vt
		default:
			iv, err := laser.ToInt(val)
			if err == nil {
				fs.Deco = TextDecorations(iv)
			} else {
				StyleSetError(key, val, err)
			}
		}
	},
}

// StyleFontRenderFuncs are _extra_ functions for styling
// the FontRender object in addition to base Font
var StyleFontRenderFuncs = map[string]StyleFunc{
	"color": func(obj any, key string, val any, par any, ctxt colors.Context) {
		fs := obj.(*FontRender)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.Color = par.(*FontRender).Color
			} else if init {
				fs.Color = colors.Black
			}
			return
		}
		base := colors.Black
		if ctxt != nil {
			base = ctxt.Base()
		}
		fs.Color = grr.Log(colors.FromAny(val, base))
	},
	"background-color": func(obj any, key string, val any, par any, ctxt colors.Context) {
		fs := obj.(*FontRender)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				fs.BackgroundColor = par.(*FontRender).BackgroundColor
			} else if init {
				fs.BackgroundColor = colors.Full{}
			}
			return
		}
		grr.Log0(fs.BackgroundColor.SetAny(val, ctxt))
	},
	"opacity": StyleFuncFloat(float32(1),
		func(obj *FontRender) *float32 { return &obj.Opacity }),
}

/////////////////////////////////////////////////////////////////////////////////
//  Border

// StyleBorderFuncs are functions for styling the Border object
var StyleBorderFuncs = map[string]StyleFunc{
	// SidesTODO: need to figure out how to get key and context information for side SetAny calls
	// with padding, margin, border, etc
	"border-style": func(obj any, key string, val any, par any, ctxt colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Style = par.(*Border).Style
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
			iv, err := laser.ToInt(val)
			if err == nil {
				bs.Style.Set(BorderStyles(iv))
			} else {
				StyleSetError(key, val, err)
			}
		}
	},
	"border-width": func(obj any, key string, val any, par any, ctxt colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Width = par.(*Border).Width
			} else if init {
				bs.Width.Zero()
			}
			return
		}
		bs.Width.SetAny(val)
	},
	"border-radius": func(obj any, key string, val any, par any, ctxt colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Radius = par.(*Border).Radius
			} else if init {
				bs.Radius.Zero()
			}
			return
		}
		bs.Radius.SetAny(val)
	},
	"border-color": func(obj any, key string, val any, par any, ctxt colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Color = par.(*Border).Color
			} else if init {
				bs.Color.Set(colors.Black)
			}
			return
		}
		grr.Log0(bs.Color.SetAny(val, ctxt.Base()))
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Outline

// StyleOutlineFuncs are functions for styling the OutlineStyle object
var StyleOutlineFuncs = map[string]StyleFunc{
	"outline-style": func(obj any, key string, val any, par any, ctxt colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Style = par.(*Border).Style
			} else if init {
				bs.Style.Set(BorderNone)
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
			iv, err := laser.ToInt(val)
			if err == nil {
				bs.Style.Set(BorderStyles(iv))
			} else {
				StyleSetError(key, val, err)
			}
		}
	},
	"outline-width": func(obj any, key string, val any, par any, ctxt colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Width = par.(*Border).Width
			} else if init {
				bs.Width.Zero()
			}
			return
		}
		bs.Width.SetAny(val)
	},
	"outline-radius": func(obj any, key string, val any, par any, ctxt colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Radius = par.(*Border).Radius
			} else if init {
				bs.Radius.Zero()
			}
			return
		}
		bs.Radius.SetAny(val)
	},
	"outline-color": func(obj any, key string, val any, par any, ctxt colors.Context) {
		bs := obj.(*Border)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				bs.Color = par.(*Border).Color
			} else if init {
				bs.Color.Set(colors.Black)
			}
			return
		}
		grr.Log0(bs.Color.SetAny(val, ctxt.Base()))
	},
}

/////////////////////////////////////////////////////////////////////////////////
//  Shadow

// StyleShadowFuncs are functions for styling the Shadow object
var StyleShadowFuncs = map[string]StyleFunc{
	"box-shadow.h-offset": StyleFuncUnits(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.HOffset }),
	"box-shadow.v-offset": StyleFuncUnits(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.VOffset }),
	"box-shadow.blur": StyleFuncUnits(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.Blur }),
	"box-shadow.spread": StyleFuncUnits(units.Value{},
		func(obj *Shadow) *units.Value { return &obj.Spread }),
	"box-shadow.color": func(obj any, key string, val any, par any, ctxt colors.Context) {
		ss := obj.(*Shadow)
		if inh, init := StyleInhInit(val, par); inh || init {
			if inh {
				ss.Color = par.(*Shadow).Color
			} else if init {
				ss.Color = colors.Black
			}
			return
		}
		ss.Color = grr.Log(colors.FromAny(val, ctxt.Base()))
	},
	"box-shadow.inset": StyleFuncBool(false,
		func(obj *Shadow) *bool { return &obj.Inset }),
}

package gist

import (
	"goki.dev/ki/v2/kit"
)

// ColorSchemeTypes is an enum that contains
// the supported color scheme types
type ColorSchemeTypes int

const (
	// ColorSchemeLight is a light color scheme
	ColorSchemeLight ColorSchemeTypes = iota
	// ColorSchemeDark is a dark color scheme
	ColorSchemeDark

	ColorSchemesN
)

var TypeColorSchemeTypes = kit.Enums.AddEnumAltLower(ColorSchemesN, kit.NotBitFlag, StylePropProps, "ColorScheme")

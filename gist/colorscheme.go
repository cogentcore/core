package gist

import "github.com/goki/ki/kit"

// ColorSchemes contains the color schemes for an app.
// It contains a light and a dark color scheme.
type ColorSchemes struct {
	Light ColorScheme
	Dark  ColorScheme
}

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

// ColorScheme contains the colors for
// one color scheme (ex: light or dark).
type ColorScheme struct {
	// NOTE: the commented out fields are those mentioned in the Material 3
	// documentation but not actually used by their theme builder.
	// They will be added later once they are added to the theme builder
	// (see https://github.com/material-foundation/material-theme-builder/issues/187).
	// Mark any colors that should be changed to use the new colors with MTBTODO:

	// Primary is the base primary color applied to important elements
	Primary Color `desc:"Primary is the base primary color applied to important elements"`

	// OnPrimary is the color applied to content on top of Primary. It defaults to the contrast color of Primary.
	OnPrimary Color `desc:"OnPrimary is the color applied to content on top of Primary. It defaults to the contrast color of Primary."`

	// PrimaryContainer is the color applied to elements with less emphasis than Primary
	PrimaryContainer Color `desc:"PrimaryContainer is the color applied to elements with less emphasis than Primary"`

	// OnPrimaryContainer is the color applied to content on top of PrimaryContainer. It defaults to the contrast color of PrimaryContainer.
	OnPrimaryContainer Color `desc:"OnPrimaryContainer is the color applied to content on top of PrimaryContainer. It defaults to the contrast color of PrimaryContainer."`

	// Secondary is the base secondary color applied to less important elements
	Secondary Color `desc:"Secondary is the base secondary color applied to less important elements"`

	// OnSecondary is the color applied to content on top of Secondary. It defaults to the contrast color of Secondary.
	OnSecondary Color `desc:"OnSecondary is the color applied to content on top of Secondary. It defaults to the contrast color of Secondary."`

	// SecondaryContainer is the color applied to elements with less emphasis than Secondary
	SecondaryContainer Color `desc:"SecondaryContainer is the color applied to elements with less emphasis than Secondary"`

	// OnSecondaryContainer is the color applied to content on top of SecondaryContainer. It defaults to the contrast color of SecondaryContainer.
	OnSecondaryContainer Color `desc:"OnSecondaryContainer is the color applied to content on top of SecondaryContainer. It defaults to the contrast color of SecondaryContainer."`

	// Tertiary is the base tertiary color applied as an accent to highlight elements and screate contrast between other colors
	Tertiary Color `desc:"Tertiary is the base tertiary color applied as an accent to highlight elements and screate contrast between other colors"`

	// OnTertiary is the color applied to content on top of Tertiary. It defaults to the contrast color of Tertiary.
	OnTertiary Color `desc:"OnTertiary is the color applied to content on top of Tertiary. It defaults to the contrast color of Tertiary."`

	// TertiaryContainer is the color applied to elements with less emphasis than Tertiary
	TertiaryContainer Color `desc:"TertiaryContainer is the color applied to elements with less emphasis than Tertiary"`

	// OnTertiaryContainer is the color applied to content on top of TertiaryContainer. It defaults to the contrast color of TertiaryContainer.
	OnTertiaryContainer Color `desc:"OnTertiaryContainer is the color applied to content on top of TertiaryContainer. It defaults to the contrast color of TertiaryContainer."`

	// Error is the base error color applied to elements that indicate an error or danger
	Error Color `desc:"Error is the base error color applied to elements that indicate an error or danger"`

	// OnError is the color applied to content on top of Error. It defaults to the contrast color of Error.
	OnError Color `desc:"OnError is the color applied to content on top of Error. It defaults to the contrast color of Error."`

	// ErrorContainer is the color applied to elements with less emphasis than Error
	ErrorContainer Color `desc:"ErrorContainer is the color applied to elements with less emphasis than Error"`

	// OnErrorContainer is the color applied to content on top of ErrorContainer. It defaults to the contrast color of ErrorContainer.
	OnErrorContainer Color `desc:"OnErrorContainer is the color applied to content on top of ErrorContainer. It defaults to the contrast color of ErrorContainer."`

	// Outline is the color applied to borders to create emphasized boundaries that need to have sufficient contrast
	Outline Color `desc:"Outline is the color applied to borders to create emphasized boundaries that need to have sufficient contrast"`

	// OutlineVariant is the color applied to create decorative boundaries
	OutlineVariant Color `desc:"OutlineVariant is the color applied to create decorative boundaries"`

	// Background is the color applied to the background of the app and other low-emphasis areas
	Background Color `desc:"Background is the color applied to the background of the app and other low-emphasis areas"`

	// OnBackground is the color applied to content on top of Background
	OnBackground Color `desc:"OnBackground is the color applied to content on top of Background"`

	// SurfaceDim is the color applied to elements that will always have the dimmest surface color (see Surface for more information)
	SurfaceDim Color `desc:"SurfaceDim is the color applied to elements that will always have the dimmest surface color (see Surface for more information)"`

	// Surface is the color applied to contained areas, like the background of an app
	Surface Color `desc:"Surface is the color applied to contained areas, like the background of an app"`

	// SurfaceBright is the color applied to elements that will always have the brightest surface color (see Surface for more information)
	SurfaceBright Color `desc:"SurfaceBright is the color applied to elements that will always have the brightest surface color (see Surface for more information)"`

	// OnSurface is the color applied to content on top of Surface elements
	OnSurface Color `desc:"OnSurface is the color applied to content on top of Surface elements"`

	// SurfaceVariant is the color applied to contained areas that contrast standard Surface elements
	SurfaceVariant Color `desc:"SurfaceVariant is the color applied to contained areas that contrast standard Surface elements"`

	// OnSurfaceVariant is the color applied to content on top of SurfaceVariant elements
	OnSurfaceVariant Color `desc:"OnSurfaceVariant is the color applied to content on top of SurfaceVariant elements"`

	// SurfaceContainerLowest is the color applied to surface container elements that have the lowest emphasis (see SurfaceContainer for more information)
	SurfaceContainerLowest Color `desc:"SurfaceContainerLowest is the color applied to surface container elements that have the lowest emphasis (see SurfaceContainer for more information)"`

	// SurfaceContainerLow is the color applied to surface container elements that have lower emphasis (see SurfaceContainer for more information)
	SurfaceContainerLow Color `desc:"SurfaceContainerLow is the color applied to surface container elements that have lower emphasis (see SurfaceContainer for more information)"`

	// SurfaceContainer is the color applied to container elements that contrast elements with the surface color
	SurfaceContainer Color `desc:"SurfaceContainer is the color applied to container elements that contrast elements with the surface color"`

	// SurfaceContainerHigh is the color applied to surface container elements that have higher emphasis (see SurfaceContainer for more information)
	SurfaceContainerHigh Color `desc:"SurfaceContainerHigh is the color applied to surface container elements that have higher emphasis (see SurfaceContainer for more information)"`

	// SurfaceContainerHighest is the color applied to surface container elements that have the highest emphasis (see SurfaceContainer for more information)
	SurfaceContainerHighest Color `desc:"SurfaceContainerHighest is the color applied to surface container elements that have the highest emphasis (see SurfaceContainer for more information)"`

	// InverseSurface is the color applied to elements to make them the reverse color of the surrounding elements and create a contrasting effect
	InverseSurface Color `desc:"InverseSurface is the color applied to elements to make them the reverse color of the surrounding elements and create a contrasting effect"`

	// InverseOnSurface is the color applied to content on top of InverseSurface
	InverseOnSurface Color `desc:"InverseOnSurface is the color applied to content on top of InverseSurface"`

	// InversePrimary is the color applied to interactive elements on top of InverseSurface
	InversePrimary Color `desc:"InversePrimary is the color applied to interactive elements on top of InverseSurface"`

	// Shadow is the color applied to shadows
	Shadow Color `desc:"Shadow is the color applied to shadows"`

	// SurfaceTint is the color applied to tint surfaces
	SurfaceTint Color `desc:"SurfaceTint is the color applied to tint surfaces"`

	// Scrim is the color applied to scrims (semi-transparent overlays)
	Scrim Color `desc:"Scrim is the color applied to scrims (semi-transparent overlays)"`

	// PrimaryFixed          Color `desc:"PrimaryFixed is a primary fill color that stays the same regardless of color scheme type (light/dark)"`
	// PrimaryFixedDim       Color `desc:"PrimaryFixedDim is a higher-emphasis, dimmer primary fill color that stays the same regardless of color scheme type (light/dark)"`
	// OnPrimaryFixed        Color `desc:"OnPrimaryFixed is the color applied to high-emphasis content on top of PrimaryFixed"`
	// OnPrimaryFixedVariant Color `desc:"OnPrimaryFixedVariant is the color applied to low-emphasis content on top of PrimaryFixed"`

	// SecondaryFixed          Color `desc:"SecondaryFixed is a secondary fill color that stays the same regardless of color scheme type (light/dark)"`
	// SecondaryFixedDim       Color `desc:"SecondaryFixedDim is a higher-emphasis, dimmer secondary fill color that stays the same regardless of color scheme type (light/dark)"`
	// OnSecondaryFixed        Color `desc:"OnSecondaryFixed is the color applied to high-emphasis content on top of SecondaryFixed"`
	// OnSecondaryFixedVariant Color `desc:"OnSecondaryFixedVariant is the color applied to low-emphasis content on top of SecondaryFixed"`

	// TertiaryFixed          Color `desc:"TertiaryFixed is a tertiary fill color that stays the same regardless of color scheme type (light/dark)"`
	// TertiaryFixedDim       Color `desc:"TertiaryFixedDim is a higher-emphasis, dimmer tertiary fill color that stays the same regardless of color scheme type (light/dark)"`
	// OnTertiaryFixed        Color `desc:"OnTertiaryFixed is the color applied to high-emphasis content on top of TertiaryFixed"`
	// OnTertiaryFixedVariant Color `desc:"OnTertiaryFixedVariant is the color applied to low-emphasis content on top of TertiaryFixed"`
}

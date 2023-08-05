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

//go:generate stringer -type=ColorSchemeTypes

// ColorScheme contains the colors for
// one color scheme (ex: light or dark).
type ColorScheme struct {
	Primary            Color `desc:"Primary is the base primary color applied to important elements"`                                                                       // Primary is the base primary color applied to important elements
	OnPrimary          Color `desc:"OnPrimary is the color applied to content on top of Primary. It defaults to the contrast color of Primary."`                            // OnPrimary is the color applied to content on top of Primary. It defaults to the contrast color of Primary.
	PrimaryContainer   Color `desc:"PrimaryContainer is the color applied to elements with less emphasis than Primary"`                                                     // PrimaryContainer is the color applied to elements with less emphasis than Primary
	OnPrimaryContainer Color `desc:"OnPrimaryContainer is the color applied to content on top of PrimaryContainer. It defaults to the contrast color of PrimaryContainer."` // OnPrimaryContainer is the color applied to content on top of PrimaryContainer. It defaults to the contrast color of PrimaryContainer.

	Secondary            Color `desc:"Secondary is the base secondary color applied to less important elements"`                                                                    // Secondary is the base secondary color applied to less important elements
	OnSecondary          Color `desc:"OnSecondary is the color applied to content on top of Secondary. It defaults to the contrast color of Secondary."`                            // OnSecondary is the color applied to content on top of Secondary. It defaults to the contrast color of Secondary.
	SecondaryContainer   Color `desc:"SecondaryContainer is the color applied to elements with less emphasis than Secondary"`                                                       // SecondaryContainer is the color applied to elements with less emphasis than Secondary
	OnSecondaryContainer Color `desc:"OnSecondaryContainer is the color applied to content on top of SecondaryContainer. It defaults to the contrast color of SecondaryContainer."` // OnSecondaryContainer is the color applied to content on top of SecondaryContainer. It defaults to the contrast color of SecondaryContainer.

	Tertiary            Color `desc:"Tertiary is the base tertiary color applied as an accent to highlight elements and screate contrast between other colors"`                 // Tertiary is the base tertiary color applied as an accent to highlight elements and screate contrast between other colors
	OnTertiary          Color `desc:"OnTertiary is the color applied to content on top of Tertiary. It defaults to the contrast color of Tertiary."`                            // OnTertiary is the color applied to content on top of Tertiary. It defaults to the contrast color of Tertiary.
	TertiaryContainer   Color `desc:"TertiaryContainer is the color applied to elements with less emphasis than Tertiary"`                                                      // TertiaryContainer is the color applied to elements with less emphasis than Tertiary
	OnTertiaryContainer Color `desc:"OnTertiaryContainer is the color applied to content on top of TertiaryContainer. It defaults to the contrast color of TertiaryContainer."` // OnTertiaryContainer is the color applied to content on top of TertiaryContainer. It defaults to the contrast color of TertiaryContainer.

	Error            Color `desc:"Error is the base error color applied to elements that indicate an error or danger"`                                              // Error is the base error color applied to elements that indicate an error or danger
	OnError          Color `desc:"OnError is the color applied to content on top of Error. It defaults to the contrast color of Error."`                            // OnError is the color applied to content on top of Error. It defaults to the contrast color of Error.
	ErrorContainer   Color `desc:"ErrorContainer is the color applied to elements with less emphasis than Error"`                                                   // ErrorContainer is the color applied to elements with less emphasis than Error
	OnErrorContainer Color `desc:"OnErrorContainer is the color applied to content on top of ErrorContainer. It defaults to the contrast color of ErrorContainer."` // OnErrorContainer is the color applied to content on top of ErrorContainer. It defaults to the contrast color of ErrorContainer.

	Outline        Color `desc:"Outline is the color applied to borders to create emphasized boundaries that need to have sufficient contrast"` // Outline is the color applied to borders to create emphasized boundaries that need to have sufficient contrast
	OutlineVariant Color `desc:"OutlineVariant is the color applied to create decorative boundaries"`                                           // OutlineVariant is the color applied to create decorative boundaries

	Background   Color `desc:"Background is the color applied to the background of the app and other low-emphasis areas"` // Background is the color applied to the background of the app and other low-emphasis areas
	OnBackground Color `desc:"OnBackground is the color applied to content on top of Background"`                         // OnBackground is the color applied to content on top of Background

	Surface          Color `desc:"Surface is the color applied to contained areas, like the background of an app"`                 // Surface is the color applied to contained areas, like the background of an app
	OnSurface        Color `desc:"OnSurface is the color applied to content on top of Surface elements"`                           // OnSurface is the color applied to content on top of Surface elements
	SurfaceVariant   Color `desc:"SurfaceVariant is the color applied to contained areas that contrast standard Surface elements"` // SurfaceVariant is the color applied to contained areas that contrast standard Surface elements
	OnSurfaceVariant Color `desc:"OnSurfaceVariant is the color applied to content on top of SurfaceVariant elements"`             // OnSurfaceVariant is the color applied to content on top of SurfaceVariant elements

	InverseSurface   Color `desc:"InverseSurface is the color applied to elements to make them the reverse color of the surrounding elements and create a contrasting effect"` // InverseSurface is the color applied to elements to make them the reverse color of the surrounding elements and create a contrasting effect
	InverseOnSurface Color `desc:"InverseOnSurface is the color applied to content on top of InverseSurface"`                                                                  // InverseOnSurface is the color applied to content on top of InverseSurface
	InversePrimary   Color `desc:"InversePrimary is the color applied to interactive elements on top of InverseSurface"`                                                       // InversePrimary is the color applied to interactive elements on top of InverseSurface

	Shadow      Color `desc:"Shadow is the color applied to shadows"`            // Shadow is the color applied to shadows
	SurfaceTint Color `desc:"SurfaceTint is the color applied to tint surfaces"` // SurfaceTint is the color applied to tint surfaces

	Scrim Color `desc:"Scrim is the color applied to scrims (semi-transparent overlays)"` // Scrim is the color applied to scrims (semi-transparent overlays)

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

// Defaults applies the default values to the color keys
func (cs *ColorScheme) Defaults() {

}

// Init sets all of the color scheme values based on the
// values of the color key values
func (cs *ColorScheme) Init() {

}

// Code generated by "core generate -add-types -setters"; DO NOT EDIT.

package rich

import (
	"cogentcore.org/core/enums"
)

var _FamilyValues = []Family{0, 1, 2, 3, 4, 5, 6, 7, 8}

// FamilyN is the highest valid value for type Family, plus one.
const FamilyN Family = 9

var _FamilyValueMap = map[string]Family{`sans-serif`: 0, `serif`: 1, `monospace`: 2, `cursive`: 3, `fantasy`: 4, `maths`: 5, `emoji`: 6, `fangsong`: 7, `custom`: 8}

var _FamilyDescMap = map[Family]string{0: `SansSerif is a font without serifs, where glyphs have plain stroke endings, without ornamentation. Example sans-serif fonts include Arial, Helvetica, Open Sans, Fira Sans, Lucida Sans, Lucida Sans Unicode, Trebuchet MS, Liberation Sans, and Nimbus Sans L.`, 1: `Serif is a small line or stroke attached to the end of a larger stroke in a letter. In serif fonts, glyphs have finishing strokes, flared or tapering ends. Examples include Times New Roman, Lucida Bright, Lucida Fax, Palatino, Palatino Linotype, Palladio, and URW Palladio.`, 2: `Monospace fonts have all glyphs with he same fixed width. Example monospace fonts include Fira Mono, DejaVu Sans Mono, Menlo, Consolas, Liberation Mono, Monaco, and Lucida Console.`, 3: `Cursive glyphs generally have either joining strokes or other cursive characteristics beyond those of italic typefaces. The glyphs are partially or completely connected, and the result looks more like handwritten pen or brush writing than printed letter work. Example cursive fonts include Brush Script MT, Brush Script Std, Lucida Calligraphy, Lucida Handwriting, and Apple Chancery.`, 4: `Fantasy fonts are primarily decorative fonts that contain playful representations of characters. Example fantasy fonts include Papyrus, Herculanum, Party LET, Curlz MT, and Harrington.`, 5: `Maths fonts are for displaying mathematical expressions, for example superscript and subscript, brackets that cross several lines, nesting expressions, and double-struck glyphs with distinct meanings.`, 6: `Emoji fonts are specifically designed to render emoji.`, 7: `Fangsong are a particular style of Chinese characters that are between serif-style Song and cursive-style Kai forms. This style is often used for government documents.`, 8: `Custom is a custom font name that can be set in Context.`}

var _FamilyMap = map[Family]string{0: `sans-serif`, 1: `serif`, 2: `monospace`, 3: `cursive`, 4: `fantasy`, 5: `maths`, 6: `emoji`, 7: `fangsong`, 8: `custom`}

// String returns the string representation of this Family value.
func (i Family) String() string { return enums.String(i, _FamilyMap) }

// SetString sets the Family value from its string representation,
// and returns an error if the string is invalid.
func (i *Family) SetString(s string) error { return enums.SetString(i, s, _FamilyValueMap, "Family") }

// Int64 returns the Family value as an int64.
func (i Family) Int64() int64 { return int64(i) }

// SetInt64 sets the Family value from an int64.
func (i *Family) SetInt64(in int64) { *i = Family(in) }

// Desc returns the description of the Family value.
func (i Family) Desc() string { return enums.Desc(i, _FamilyDescMap) }

// FamilyValues returns all possible values for the type Family.
func FamilyValues() []Family { return _FamilyValues }

// Values returns all possible values for the type Family.
func (i Family) Values() []enums.Enum { return enums.Values(_FamilyValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Family) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Family) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "Family") }

var _SlantsValues = []Slants{0, 1}

// SlantsN is the highest valid value for type Slants, plus one.
const SlantsN Slants = 2

var _SlantsValueMap = map[string]Slants{`normal`: 0, `italic`: 1}

var _SlantsDescMap = map[Slants]string{0: `A face that is neither italic not obliqued.`, 1: `A form that is generally cursive in nature or slanted. This groups what is usually called Italic or Oblique.`}

var _SlantsMap = map[Slants]string{0: `normal`, 1: `italic`}

// String returns the string representation of this Slants value.
func (i Slants) String() string { return enums.String(i, _SlantsMap) }

// SetString sets the Slants value from its string representation,
// and returns an error if the string is invalid.
func (i *Slants) SetString(s string) error { return enums.SetString(i, s, _SlantsValueMap, "Slants") }

// Int64 returns the Slants value as an int64.
func (i Slants) Int64() int64 { return int64(i) }

// SetInt64 sets the Slants value from an int64.
func (i *Slants) SetInt64(in int64) { *i = Slants(in) }

// Desc returns the description of the Slants value.
func (i Slants) Desc() string { return enums.Desc(i, _SlantsDescMap) }

// SlantsValues returns all possible values for the type Slants.
func SlantsValues() []Slants { return _SlantsValues }

// Values returns all possible values for the type Slants.
func (i Slants) Values() []enums.Enum { return enums.Values(_SlantsValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Slants) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Slants) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "Slants") }

var _WeightsValues = []Weights{0, 1, 2, 3, 4, 5, 6, 7, 8}

// WeightsN is the highest valid value for type Weights, plus one.
const WeightsN Weights = 9

var _WeightsValueMap = map[string]Weights{`thin`: 0, `extra-light`: 1, `light`: 2, `normal`: 3, `medium`: 4, `semibold`: 5, `bold`: 6, `extra-bold`: 7, `black`: 8}

var _WeightsDescMap = map[Weights]string{0: `Thin weight (100), the thinnest value.`, 1: `Extra light weight (200).`, 2: `Light weight (300).`, 3: `Normal (400).`, 4: `Medium weight (500, higher than normal).`, 5: `Semibold weight (600).`, 6: `Bold weight (700).`, 7: `Extra-bold weight (800).`, 8: `Black weight (900), the thickest value.`}

var _WeightsMap = map[Weights]string{0: `thin`, 1: `extra-light`, 2: `light`, 3: `normal`, 4: `medium`, 5: `semibold`, 6: `bold`, 7: `extra-bold`, 8: `black`}

// String returns the string representation of this Weights value.
func (i Weights) String() string { return enums.String(i, _WeightsMap) }

// SetString sets the Weights value from its string representation,
// and returns an error if the string is invalid.
func (i *Weights) SetString(s string) error {
	return enums.SetString(i, s, _WeightsValueMap, "Weights")
}

// Int64 returns the Weights value as an int64.
func (i Weights) Int64() int64 { return int64(i) }

// SetInt64 sets the Weights value from an int64.
func (i *Weights) SetInt64(in int64) { *i = Weights(in) }

// Desc returns the description of the Weights value.
func (i Weights) Desc() string { return enums.Desc(i, _WeightsDescMap) }

// WeightsValues returns all possible values for the type Weights.
func WeightsValues() []Weights { return _WeightsValues }

// Values returns all possible values for the type Weights.
func (i Weights) Values() []enums.Enum { return enums.Values(_WeightsValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Weights) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Weights) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "Weights") }

var _StretchValues = []Stretch{0, 1, 2, 3, 4, 5, 6, 7, 8}

// StretchN is the highest valid value for type Stretch, plus one.
const StretchN Stretch = 9

var _StretchValueMap = map[string]Stretch{`ultra-condensed`: 0, `extra-condensed`: 1, `condensed`: 2, `semi-condensed`: 3, `normal`: 4, `semi-expanded`: 5, `expanded`: 6, `extra-expanded`: 7, `ultra-expanded`: 8}

var _StretchDescMap = map[Stretch]string{0: `Ultra-condensed width (50%), the narrowest possible.`, 1: `Extra-condensed width (62.5%).`, 2: `Condensed width (75%).`, 3: `Semi-condensed width (87.5%).`, 4: `Normal width (100%).`, 5: `Semi-expanded width (112.5%).`, 6: `Expanded width (125%).`, 7: `Extra-expanded width (150%).`, 8: `Ultra-expanded width (200%), the widest possible.`}

var _StretchMap = map[Stretch]string{0: `ultra-condensed`, 1: `extra-condensed`, 2: `condensed`, 3: `semi-condensed`, 4: `normal`, 5: `semi-expanded`, 6: `expanded`, 7: `extra-expanded`, 8: `ultra-expanded`}

// String returns the string representation of this Stretch value.
func (i Stretch) String() string { return enums.String(i, _StretchMap) }

// SetString sets the Stretch value from its string representation,
// and returns an error if the string is invalid.
func (i *Stretch) SetString(s string) error {
	return enums.SetString(i, s, _StretchValueMap, "Stretch")
}

// Int64 returns the Stretch value as an int64.
func (i Stretch) Int64() int64 { return int64(i) }

// SetInt64 sets the Stretch value from an int64.
func (i *Stretch) SetInt64(in int64) { *i = Stretch(in) }

// Desc returns the description of the Stretch value.
func (i Stretch) Desc() string { return enums.Desc(i, _StretchDescMap) }

// StretchValues returns all possible values for the type Stretch.
func StretchValues() []Stretch { return _StretchValues }

// Values returns all possible values for the type Stretch.
func (i Stretch) Values() []enums.Enum { return enums.Values(_StretchValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Stretch) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Stretch) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "Stretch") }

var _DecorationsValues = []Decorations{0, 1, 2, 3, 4, 5, 6, 7}

// DecorationsN is the highest valid value for type Decorations, plus one.
const DecorationsN Decorations = 8

var _DecorationsValueMap = map[string]Decorations{`underline`: 0, `overline`: 1, `line-through`: 2, `dotted-underline`: 3, `link`: 4, `fill-color`: 5, `stroke-color`: 6, `background`: 7}

var _DecorationsDescMap = map[Decorations]string{0: `Underline indicates to place a line below text.`, 1: `Overline indicates to place a line above text.`, 2: `LineThrough indicates to place a line through text.`, 3: `DottedUnderline is used for abbr tag.`, 4: `Link indicates a hyperlink, which identifies this span for functional interactions such as hovering and clicking. It does not specify the styling.`, 5: `FillColor means that the fill color of the glyph is set to FillColor, which encoded in the rune following the style rune, rather than the default. The standard font rendering uses this fill color (compare to StrokeColor).`, 6: `StrokeColor means that the stroke color of the glyph is set to StrokeColor, which is encoded in the rune following the style rune. This is normally not rendered: it looks like an outline of the glyph at larger font sizes, it will make smaller font sizes look significantly thicker.`, 7: `Background means that the background region behind the text is colored to Background, which is encoded in the rune following the style rune. The background is not normally colored.`}

var _DecorationsMap = map[Decorations]string{0: `underline`, 1: `overline`, 2: `line-through`, 3: `dotted-underline`, 4: `link`, 5: `fill-color`, 6: `stroke-color`, 7: `background`}

// String returns the string representation of this Decorations value.
func (i Decorations) String() string { return enums.BitFlagString(i, _DecorationsValues) }

// BitIndexString returns the string representation of this Decorations value
// if it is a bit index value (typically an enum constant), and
// not an actual bit flag value.
func (i Decorations) BitIndexString() string { return enums.String(i, _DecorationsMap) }

// SetString sets the Decorations value from its string representation,
// and returns an error if the string is invalid.
func (i *Decorations) SetString(s string) error { *i = 0; return i.SetStringOr(s) }

// SetStringOr sets the Decorations value from its string representation
// while preserving any bit flags already set, and returns an
// error if the string is invalid.
func (i *Decorations) SetStringOr(s string) error {
	return enums.SetStringOr(i, s, _DecorationsValueMap, "Decorations")
}

// Int64 returns the Decorations value as an int64.
func (i Decorations) Int64() int64 { return int64(i) }

// SetInt64 sets the Decorations value from an int64.
func (i *Decorations) SetInt64(in int64) { *i = Decorations(in) }

// Desc returns the description of the Decorations value.
func (i Decorations) Desc() string { return enums.Desc(i, _DecorationsDescMap) }

// DecorationsValues returns all possible values for the type Decorations.
func DecorationsValues() []Decorations { return _DecorationsValues }

// Values returns all possible values for the type Decorations.
func (i Decorations) Values() []enums.Enum { return enums.Values(_DecorationsValues) }

// HasFlag returns whether these bit flags have the given bit flag set.
func (i *Decorations) HasFlag(f enums.BitFlag) bool { return enums.HasFlag((*int64)(i), f) }

// SetFlag sets the value of the given flags in these flags to the given value.
func (i *Decorations) SetFlag(on bool, f ...enums.BitFlag) { enums.SetFlag((*int64)(i), on, f...) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Decorations) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Decorations) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "Decorations")
}

var _SpecialsValues = []Specials{0, 1, 2, 3}

// SpecialsN is the highest valid value for type Specials, plus one.
const SpecialsN Specials = 4

var _SpecialsValueMap = map[string]Specials{`nothing`: 0, `super`: 1, `sub`: 2, `math`: 3}

var _SpecialsDescMap = map[Specials]string{0: `Nothing special.`, 1: `Super indicates super-scripted text.`, 2: `Sub indicates sub-scripted text.`, 3: `Math indicates a LaTeX formatted math sequence.`}

var _SpecialsMap = map[Specials]string{0: `nothing`, 1: `super`, 2: `sub`, 3: `math`}

// String returns the string representation of this Specials value.
func (i Specials) String() string { return enums.String(i, _SpecialsMap) }

// SetString sets the Specials value from its string representation,
// and returns an error if the string is invalid.
func (i *Specials) SetString(s string) error {
	return enums.SetString(i, s, _SpecialsValueMap, "Specials")
}

// Int64 returns the Specials value as an int64.
func (i Specials) Int64() int64 { return int64(i) }

// SetInt64 sets the Specials value from an int64.
func (i *Specials) SetInt64(in int64) { *i = Specials(in) }

// Desc returns the description of the Specials value.
func (i Specials) Desc() string { return enums.Desc(i, _SpecialsDescMap) }

// SpecialsValues returns all possible values for the type Specials.
func SpecialsValues() []Specials { return _SpecialsValues }

// Values returns all possible values for the type Specials.
func (i Specials) Values() []enums.Enum { return enums.Values(_SpecialsValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Specials) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Specials) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "Specials") }

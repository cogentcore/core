// Code generated by "goki generate -add-types"; DO NOT EDIT.

package xyz

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"sync/atomic"

	"goki.dev/enums"
	"goki.dev/ki"
)

var _LightColorsValues = []LightColors{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}

// LightColorsN is the highest valid value
// for type LightColors, plus one.
const LightColorsN LightColors = 15

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the enumgen command to generate them again.
func _LightColorsNoOp() {
	var x [1]struct{}
	_ = x[DirectSun-(0)]
	_ = x[CarbonArc-(1)]
	_ = x[Halogen-(2)]
	_ = x[Tungsten100W-(3)]
	_ = x[Tungsten40W-(4)]
	_ = x[Candle-(5)]
	_ = x[Overcast-(6)]
	_ = x[FluorWarm-(7)]
	_ = x[FluorStd-(8)]
	_ = x[FluorCool-(9)]
	_ = x[FluorFull-(10)]
	_ = x[FluorGrow-(11)]
	_ = x[MercuryVapor-(12)]
	_ = x[SodiumVapor-(13)]
	_ = x[MetalHalide-(14)]
}

var _LightColorsNameToValueMap = map[string]LightColors{
	`DirectSun`:    0,
	`CarbonArc`:    1,
	`Halogen`:      2,
	`Tungsten100W`: 3,
	`Tungsten40W`:  4,
	`Candle`:       5,
	`Overcast`:     6,
	`FluorWarm`:    7,
	`FluorStd`:     8,
	`FluorCool`:    9,
	`FluorFull`:    10,
	`FluorGrow`:    11,
	`MercuryVapor`: 12,
	`SodiumVapor`:  13,
	`MetalHalide`:  14,
}

var _LightColorsDescMap = map[LightColors]string{
	0:  ``,
	1:  ``,
	2:  ``,
	3:  ``,
	4:  ``,
	5:  ``,
	6:  ``,
	7:  ``,
	8:  ``,
	9:  ``,
	10: ``,
	11: ``,
	12: ``,
	13: ``,
	14: ``,
}

var _LightColorsMap = map[LightColors]string{
	0:  `DirectSun`,
	1:  `CarbonArc`,
	2:  `Halogen`,
	3:  `Tungsten100W`,
	4:  `Tungsten40W`,
	5:  `Candle`,
	6:  `Overcast`,
	7:  `FluorWarm`,
	8:  `FluorStd`,
	9:  `FluorCool`,
	10: `FluorFull`,
	11: `FluorGrow`,
	12: `MercuryVapor`,
	13: `SodiumVapor`,
	14: `MetalHalide`,
}

// String returns the string representation
// of this LightColors value.
func (i LightColors) String() string {
	if str, ok := _LightColorsMap[i]; ok {
		return str
	}
	return strconv.FormatInt(int64(i), 10)
}

// SetString sets the LightColors value from its
// string representation, and returns an
// error if the string is invalid.
func (i *LightColors) SetString(s string) error {
	if val, ok := _LightColorsNameToValueMap[s]; ok {
		*i = val
		return nil
	}
	return errors.New(s + " is not a valid value for type LightColors")
}

// Int64 returns the LightColors value as an int64.
func (i LightColors) Int64() int64 {
	return int64(i)
}

// SetInt64 sets the LightColors value from an int64.
func (i *LightColors) SetInt64(in int64) {
	*i = LightColors(in)
}

// Desc returns the description of the LightColors value.
func (i LightColors) Desc() string {
	if str, ok := _LightColorsDescMap[i]; ok {
		return str
	}
	return i.String()
}

// LightColorsValues returns all possible values
// for the type LightColors.
func LightColorsValues() []LightColors {
	return _LightColorsValues
}

// Values returns all possible values
// for the type LightColors.
func (i LightColors) Values() []enums.Enum {
	res := make([]enums.Enum, len(_LightColorsValues))
	for i, d := range _LightColorsValues {
		res[i] = d
	}
	return res
}

// IsValid returns whether the value is a
// valid option for type LightColors.
func (i LightColors) IsValid() bool {
	_, ok := _LightColorsMap[i]
	return ok
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i LightColors) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *LightColors) UnmarshalText(text []byte) error {
	if err := i.SetString(string(text)); err != nil {
		log.Println("LightColors.UnmarshalText:", err)
	}
	return nil
}

var _NodeFlagsValues = []NodeFlags{7, 8, 9}

// NodeFlagsN is the highest valid value
// for type NodeFlags, plus one.
const NodeFlagsN NodeFlags = 10

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the enumgen command to generate them again.
func _NodeFlagsNoOp() {
	var x [1]struct{}
	_ = x[WorldMatrixUpdated-(7)]
	_ = x[VectorsUpdated-(8)]
	_ = x[Invisible-(9)]
}

var _NodeFlagsNameToValueMap = map[string]NodeFlags{
	`WorldMatrixUpdated`: 7,
	`VectorsUpdated`:     8,
	`Invisible`:          9,
}

var _NodeFlagsDescMap = map[NodeFlags]string{
	7: `WorldMatrixUpdated means that the Pose.WorldMatrix has been updated`,
	8: `VectorsUpdated means that the rendering vectors information is updated`,
	9: `Invisible marks this node as invisible`,
}

var _NodeFlagsMap = map[NodeFlags]string{
	7: `WorldMatrixUpdated`,
	8: `VectorsUpdated`,
	9: `Invisible`,
}

// String returns the string representation
// of this NodeFlags value.
func (i NodeFlags) String() string {
	str := ""
	for _, ie := range ki.FlagsValues() {
		if i.HasFlag(ie) {
			ies := ie.BitIndexString()
			if str == "" {
				str = ies
			} else {
				str += "|" + ies
			}
		}
	}
	for _, ie := range _NodeFlagsValues {
		if i.HasFlag(ie) {
			ies := ie.BitIndexString()
			if str == "" {
				str = ies
			} else {
				str += "|" + ies
			}
		}
	}
	return str
}

// BitIndexString returns the string
// representation of this NodeFlags value
// if it is a bit index value
// (typically an enum constant), and
// not an actual bit flag value.
func (i NodeFlags) BitIndexString() string {
	if str, ok := _NodeFlagsMap[i]; ok {
		return str
	}
	return ki.Flags(i).BitIndexString()
}

// SetString sets the NodeFlags value from its
// string representation, and returns an
// error if the string is invalid.
func (i *NodeFlags) SetString(s string) error {
	*i = 0
	return i.SetStringOr(s)
}

// SetStringOr sets the NodeFlags value from its
// string representation while preserving any
// bit flags already set, and returns an
// error if the string is invalid.
func (i *NodeFlags) SetStringOr(s string) error {
	flgs := strings.Split(s, "|")
	for _, flg := range flgs {
		if val, ok := _NodeFlagsNameToValueMap[flg]; ok {
			i.SetFlag(true, &val)
		} else if flg == "" {
			continue
		} else {
			err := (*ki.Flags)(i).SetStringOr(flg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Int64 returns the NodeFlags value as an int64.
func (i NodeFlags) Int64() int64 {
	return int64(i)
}

// SetInt64 sets the NodeFlags value from an int64.
func (i *NodeFlags) SetInt64(in int64) {
	*i = NodeFlags(in)
}

// Desc returns the description of the NodeFlags value.
func (i NodeFlags) Desc() string {
	if str, ok := _NodeFlagsDescMap[i]; ok {
		return str
	}
	return ki.Flags(i).Desc()
}

// NodeFlagsValues returns all possible values
// for the type NodeFlags.
func NodeFlagsValues() []NodeFlags {
	es := ki.FlagsValues()
	res := make([]NodeFlags, len(es))
	for i, e := range es {
		res[i] = NodeFlags(e)
	}
	res = append(res, _NodeFlagsValues...)
	return res
}

// Values returns all possible values
// for the type NodeFlags.
func (i NodeFlags) Values() []enums.Enum {
	es := ki.FlagsValues()
	les := len(es)
	res := make([]enums.Enum, les+len(_NodeFlagsValues))
	for i, d := range es {
		res[i] = d
	}
	for i, d := range _NodeFlagsValues {
		res[i+les] = d
	}
	return res
}

// IsValid returns whether the value is a
// valid option for type NodeFlags.
func (i NodeFlags) IsValid() bool {
	_, ok := _NodeFlagsMap[i]
	if !ok {
		return ki.Flags(i).IsValid()
	}
	return ok
}

// HasFlag returns whether these
// bit flags have the given bit flag set.
func (i NodeFlags) HasFlag(f enums.BitFlag) bool {
	return atomic.LoadInt64((*int64)(&i))&(1<<uint32(f.Int64())) != 0
}

// SetFlag sets the value of the given
// flags in these flags to the given value.
func (i *NodeFlags) SetFlag(on bool, f ...enums.BitFlag) {
	var mask int64
	for _, v := range f {
		mask |= 1 << v.Int64()
	}
	in := int64(*i)
	if on {
		in |= mask
		atomic.StoreInt64((*int64)(i), in)
	} else {
		in &^= mask
		atomic.StoreInt64((*int64)(i), in)
	}
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i NodeFlags) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *NodeFlags) UnmarshalText(text []byte) error {
	if err := i.SetString(string(text)); err != nil {
		log.Println("NodeFlags.UnmarshalText:", err)
	}
	return nil
}

var _RenderClassesValues = []RenderClasses{0, 1, 2, 3, 4, 5, 6}

// RenderClassesN is the highest valid value
// for type RenderClasses, plus one.
const RenderClassesN RenderClasses = 7

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the enumgen command to generate them again.
func _RenderClassesNoOp() {
	var x [1]struct{}
	_ = x[RClassNone-(0)]
	_ = x[RClassOpaqueTexture-(1)]
	_ = x[RClassOpaqueUniform-(2)]
	_ = x[RClassOpaqueVertex-(3)]
	_ = x[RClassTransTexture-(4)]
	_ = x[RClassTransUniform-(5)]
	_ = x[RClassTransVertex-(6)]
}

var _RenderClassesNameToValueMap = map[string]RenderClasses{
	`None`:          0,
	`OpaqueTexture`: 1,
	`OpaqueUniform`: 2,
	`OpaqueVertex`:  3,
	`TransTexture`:  4,
	`TransUniform`:  5,
	`TransVertex`:   6,
}

var _RenderClassesDescMap = map[RenderClasses]string{
	0: ``,
	1: ``,
	2: ``,
	3: ``,
	4: ``,
	5: ``,
	6: ``,
}

var _RenderClassesMap = map[RenderClasses]string{
	0: `None`,
	1: `OpaqueTexture`,
	2: `OpaqueUniform`,
	3: `OpaqueVertex`,
	4: `TransTexture`,
	5: `TransUniform`,
	6: `TransVertex`,
}

// String returns the string representation
// of this RenderClasses value.
func (i RenderClasses) String() string {
	if str, ok := _RenderClassesMap[i]; ok {
		return str
	}
	return strconv.FormatInt(int64(i), 10)
}

// SetString sets the RenderClasses value from its
// string representation, and returns an
// error if the string is invalid.
func (i *RenderClasses) SetString(s string) error {
	if val, ok := _RenderClassesNameToValueMap[s]; ok {
		*i = val
		return nil
	}
	return errors.New(s + " is not a valid value for type RenderClasses")
}

// Int64 returns the RenderClasses value as an int64.
func (i RenderClasses) Int64() int64 {
	return int64(i)
}

// SetInt64 sets the RenderClasses value from an int64.
func (i *RenderClasses) SetInt64(in int64) {
	*i = RenderClasses(in)
}

// Desc returns the description of the RenderClasses value.
func (i RenderClasses) Desc() string {
	if str, ok := _RenderClassesDescMap[i]; ok {
		return str
	}
	return i.String()
}

// RenderClassesValues returns all possible values
// for the type RenderClasses.
func RenderClassesValues() []RenderClasses {
	return _RenderClassesValues
}

// Values returns all possible values
// for the type RenderClasses.
func (i RenderClasses) Values() []enums.Enum {
	res := make([]enums.Enum, len(_RenderClassesValues))
	for i, d := range _RenderClassesValues {
		res[i] = d
	}
	return res
}

// IsValid returns whether the value is a
// valid option for type RenderClasses.
func (i RenderClasses) IsValid() bool {
	_, ok := _RenderClassesMap[i]
	return ok
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i RenderClasses) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *RenderClasses) UnmarshalText(text []byte) error {
	if err := i.SetString(string(text)); err != nil {
		log.Println("RenderClasses.UnmarshalText:", err)
	}
	return nil
}

var _ScFlagsValues = []ScFlags{7, 8, 9, 10}

// ScFlagsN is the highest valid value
// for type ScFlags, plus one.
const ScFlagsN ScFlags = 11

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the enumgen command to generate them again.
func _ScFlagsNoOp() {
	var x [1]struct{}
	_ = x[ScUpdating-(7)]
	_ = x[ScNeedsConfig-(8)]
	_ = x[ScNeedsUpdate-(9)]
	_ = x[ScNeedsRender-(10)]
}

var _ScFlagsNameToValueMap = map[string]ScFlags{
	`ScUpdating`:    7,
	`ScNeedsConfig`: 8,
	`ScNeedsUpdate`: 9,
	`ScNeedsRender`: 10,
}

var _ScFlagsDescMap = map[ScFlags]string{
	7:  `ScUpdating means scene is in the process of updating: set for any kind of tree-level update. skip any further update passes until it goes off.`,
	8:  `ScNeedsConfig means that a GPU resource (Lights, Texture, Meshes, or more complex Nodes that require ConfigNodes) has been changed and a Config call is required.`,
	9:  `ScNeedsUpdate means that Node Pose has changed and an update pass is required to update matrix and bounding boxes.`,
	10: `ScNeedsRender means that something has been updated (minimally the Camera pose) and a new Render is required.`,
}

var _ScFlagsMap = map[ScFlags]string{
	7:  `ScUpdating`,
	8:  `ScNeedsConfig`,
	9:  `ScNeedsUpdate`,
	10: `ScNeedsRender`,
}

// String returns the string representation
// of this ScFlags value.
func (i ScFlags) String() string {
	str := ""
	for _, ie := range ki.FlagsValues() {
		if i.HasFlag(ie) {
			ies := ie.BitIndexString()
			if str == "" {
				str = ies
			} else {
				str += "|" + ies
			}
		}
	}
	for _, ie := range _ScFlagsValues {
		if i.HasFlag(ie) {
			ies := ie.BitIndexString()
			if str == "" {
				str = ies
			} else {
				str += "|" + ies
			}
		}
	}
	return str
}

// BitIndexString returns the string
// representation of this ScFlags value
// if it is a bit index value
// (typically an enum constant), and
// not an actual bit flag value.
func (i ScFlags) BitIndexString() string {
	if str, ok := _ScFlagsMap[i]; ok {
		return str
	}
	return ki.Flags(i).BitIndexString()
}

// SetString sets the ScFlags value from its
// string representation, and returns an
// error if the string is invalid.
func (i *ScFlags) SetString(s string) error {
	*i = 0
	return i.SetStringOr(s)
}

// SetStringOr sets the ScFlags value from its
// string representation while preserving any
// bit flags already set, and returns an
// error if the string is invalid.
func (i *ScFlags) SetStringOr(s string) error {
	flgs := strings.Split(s, "|")
	for _, flg := range flgs {
		if val, ok := _ScFlagsNameToValueMap[flg]; ok {
			i.SetFlag(true, &val)
		} else if flg == "" {
			continue
		} else {
			err := (*ki.Flags)(i).SetStringOr(flg)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Int64 returns the ScFlags value as an int64.
func (i ScFlags) Int64() int64 {
	return int64(i)
}

// SetInt64 sets the ScFlags value from an int64.
func (i *ScFlags) SetInt64(in int64) {
	*i = ScFlags(in)
}

// Desc returns the description of the ScFlags value.
func (i ScFlags) Desc() string {
	if str, ok := _ScFlagsDescMap[i]; ok {
		return str
	}
	return ki.Flags(i).Desc()
}

// ScFlagsValues returns all possible values
// for the type ScFlags.
func ScFlagsValues() []ScFlags {
	es := ki.FlagsValues()
	res := make([]ScFlags, len(es))
	for i, e := range es {
		res[i] = ScFlags(e)
	}
	res = append(res, _ScFlagsValues...)
	return res
}

// Values returns all possible values
// for the type ScFlags.
func (i ScFlags) Values() []enums.Enum {
	es := ki.FlagsValues()
	les := len(es)
	res := make([]enums.Enum, les+len(_ScFlagsValues))
	for i, d := range es {
		res[i] = d
	}
	for i, d := range _ScFlagsValues {
		res[i+les] = d
	}
	return res
}

// IsValid returns whether the value is a
// valid option for type ScFlags.
func (i ScFlags) IsValid() bool {
	_, ok := _ScFlagsMap[i]
	if !ok {
		return ki.Flags(i).IsValid()
	}
	return ok
}

// HasFlag returns whether these
// bit flags have the given bit flag set.
func (i ScFlags) HasFlag(f enums.BitFlag) bool {
	return atomic.LoadInt64((*int64)(&i))&(1<<uint32(f.Int64())) != 0
}

// SetFlag sets the value of the given
// flags in these flags to the given value.
func (i *ScFlags) SetFlag(on bool, f ...enums.BitFlag) {
	var mask int64
	for _, v := range f {
		mask |= 1 << v.Int64()
	}
	in := int64(*i)
	if on {
		in |= mask
		atomic.StoreInt64((*int64)(i), in)
	} else {
		in &^= mask
		atomic.StoreInt64((*int64)(i), in)
	}
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i ScFlags) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *ScFlags) UnmarshalText(text []byte) error {
	if err := i.SetString(string(text)); err != nil {
		log.Println("ScFlags.UnmarshalText:", err)
	}
	return nil
}

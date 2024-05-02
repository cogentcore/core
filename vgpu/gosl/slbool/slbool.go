// Copyright (c) 2019, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package slbool defines a HLSL friendly int32 Bool type.
The standard HLSL bool type causes obscure errors,
and the int32 obeys the 4 byte basic alignment requirements.

gosl automatically converts this Go code into appropriate HLSL code.
*/
package slbool

// Bool is an HLSL friendly int32 Bool type.
type Bool int32

const (
	// False is the [Bool] false value
	False Bool = 0
	// True is the [Bool] true value
	True Bool = 1
)

// Bool returns the Bool as a standard Go bool
func (b Bool) Bool() bool {
	return b == True
}

// IsTrue returns whether the bool is true
func (b Bool) IsTrue() bool {
	return b == True
}

// IsFalse returns whether the bool is false
func (b Bool) IsFalse() bool {
	return b == False
}

// SetBool sets the Bool from a standard Go bool
func (b *Bool) SetBool(bb bool) {
	*b = FromBool(bb)
}

// String returns the bool as a string ("true"/"false")
func (b Bool) String() string {
	if b.IsTrue() {
		return "true"
	}
	return "false"
}

// FromString sets the bool from the given string
func (b *Bool) FromString(s string) {
	if s == "true" || s == "True" {
		b.SetBool(true)
	} else {
		b.SetBool(false)
	}

}

// MarshalText implements the [encoding/text.Marshaler] interface
func (b Bool) MarshalText() ([]byte, error) { return []byte(b.String()), nil }

// UnmarshalText implements the [encoding/text.Unmarshaler] interface
func (b *Bool) UnmarshalText(s []byte) error { b.FromString(string(s)); return nil }

// IsTrue returns whether the given bool is true
func IsTrue(b Bool) bool {
	return b == True
}

// IsFalse returns whether the given bool is false
func IsFalse(b Bool) bool {
	return b == False
}

// FromBool returns the given Go bool as a [Bool]
func FromBool(b bool) Bool {
	if b {
		return True
	}
	return False
}

// Copyright (c) 2023, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package econfig

import (
	"github.com/goki/ki/kit"
)

// TestEnum is an enum type for testing
type TestEnum int32

// note: we need to add the Layer extension to avoid naming
// conflicts between layer, projection and other things.

const (
	TestValue1 TestEnum = iota

	TestValue2

	TestEnumN
)

// important: must use 'go install github.com/goki/stringer@latest' with FromString method

//go:generate stringer -type=TestEnum

// This registers the enum with Kit which then allows full set / get from strings.
var KiT_TestEnum = kit.Enums.AddEnum(TestEnumN, kit.NotBitFlag, nil)

func (ev TestEnum) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *TestEnum) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// TOML uses Text interface
func (ev TestEnum) MarshalText() ([]byte, error)  { return kit.EnumMarshalText(ev) }
func (ev *TestEnum) UnmarshalText(b []byte) error { return kit.EnumUnmarshalText(ev, b) }

// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/c2h5oh/datasize
// Copyright (c) 2016 Maciej Lisiewski

// Package datasize provides a data size type and constants.
package datasize

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Size represents a data size.
type Size uint64

const (
	B  Size = 1
	KB      = B << 10
	MB      = KB << 10
	GB      = MB << 10
	TB      = GB << 10
	PB      = TB << 10
	EB      = PB << 10

	fnUnmarshalText string = "UnmarshalText"
	maxUint64       uint64 = (1 << 64) - 1
	cutoff          uint64 = maxUint64 / 10
)

var ErrBits = errors.New("unit with capital unit prefix and lower case unit (b) - bits, not bytes")

func (b Size) Bytes() uint64 {
	return uint64(b)
}

func (b Size) KBytes() float64 {
	v := b / KB
	r := b % KB
	return float64(v) + float64(r)/float64(KB)
}

func (b Size) MBytes() float64 {
	v := b / MB
	r := b % MB
	return float64(v) + float64(r)/float64(MB)
}

func (b Size) GBytes() float64 {
	v := b / GB
	r := b % GB
	return float64(v) + float64(r)/float64(GB)
}

func (b Size) TBytes() float64 {
	v := b / TB
	r := b % TB
	return float64(v) + float64(r)/float64(TB)
}

func (b Size) PBytes() float64 {
	v := b / PB
	r := b % PB
	return float64(v) + float64(r)/float64(PB)
}

func (b Size) EBytes() float64 {
	v := b / EB
	r := b % EB
	return float64(v) + float64(r)/float64(EB)
}

// String returns a human-readable representation of the data size.
func (b Size) String() string {
	switch {
	case b > EB:
		return fmt.Sprintf("%.1f EB", b.EBytes())
	case b > PB:
		return fmt.Sprintf("%.1f PB", b.PBytes())
	case b > TB:
		return fmt.Sprintf("%.1f TB", b.TBytes())
	case b > GB:
		return fmt.Sprintf("%.1f GB", b.GBytes())
	case b > MB:
		return fmt.Sprintf("%.1f MB", b.MBytes())
	case b > KB:
		return fmt.Sprintf("%.1f KB", b.KBytes())
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// MachineString returns a machine-friendly representation of the data size.
func (b Size) MachineString() string {
	switch {
	case b == 0:
		return "0B"
	case b%EB == 0:
		return fmt.Sprintf("%dEB", b/EB)
	case b%PB == 0:
		return fmt.Sprintf("%dPB", b/PB)
	case b%TB == 0:
		return fmt.Sprintf("%dTB", b/TB)
	case b%GB == 0:
		return fmt.Sprintf("%dGB", b/GB)
	case b%MB == 0:
		return fmt.Sprintf("%dMB", b/MB)
	case b%KB == 0:
		return fmt.Sprintf("%dKB", b/KB)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func (b Size) MarshalText() ([]byte, error) {
	return []byte(b.MachineString()), nil
}

func (b *Size) UnmarshalText(t []byte) error {
	var val uint64
	var unit string

	// copy for error message
	t0 := t

	var c byte
	var i int

ParseLoop:
	for i < len(t) {
		c = t[i]
		switch {
		case '0' <= c && c <= '9':
			if val > cutoff {
				goto Overflow
			}

			c = c - '0'
			val *= 10

			if val > val+uint64(c) {
				// val+v overflows
				goto Overflow
			}
			val += uint64(c)
			i++

		default:
			if i == 0 {
				goto SyntaxError
			}
			break ParseLoop
		}
	}

	unit = strings.TrimSpace(string(t[i:]))
	switch unit {
	case "Kb", "Mb", "Gb", "Tb", "Pb", "Eb":
		goto BitsError
	}
	unit = strings.ToLower(unit)
	switch unit {
	case "", "b", "byte":
		// do nothing - already in bytes

	case "k", "kb", "kilo", "kilobyte", "kilobytes":
		if val > maxUint64/uint64(KB) {
			goto Overflow
		}
		val *= uint64(KB)

	case "m", "mb", "mega", "megabyte", "megabytes":
		if val > maxUint64/uint64(MB) {
			goto Overflow
		}
		val *= uint64(MB)

	case "g", "gb", "giga", "gigabyte", "gigabytes":
		if val > maxUint64/uint64(GB) {
			goto Overflow
		}
		val *= uint64(GB)

	case "t", "tb", "tera", "terabyte", "terabytes":
		if val > maxUint64/uint64(TB) {
			goto Overflow
		}
		val *= uint64(TB)

	case "p", "pb", "peta", "petabyte", "petabytes":
		if val > maxUint64/uint64(PB) {
			goto Overflow
		}
		val *= uint64(PB)

	case "E", "EB", "e", "eb", "eB":
		if val > maxUint64/uint64(EB) {
			goto Overflow
		}
		val *= uint64(EB)

	default:
		goto SyntaxError
	}

	*b = Size(val)
	return nil

Overflow:
	*b = Size(maxUint64)
	return &strconv.NumError{fnUnmarshalText, string(t0), strconv.ErrRange}

SyntaxError:
	*b = 0
	return &strconv.NumError{fnUnmarshalText, string(t0), strconv.ErrSyntax}

BitsError:
	*b = 0
	return &strconv.NumError{fnUnmarshalText, string(t0), ErrBits}
}

func Parse(t []byte) (Size, error) {
	var v Size
	err := v.UnmarshalText(t)
	return v, err
}

func MustParse(t []byte) Size {
	v, err := Parse(t)
	if err != nil {
		panic(err)
	}
	return v
}

func ParseString(s string) (Size, error) {
	return Parse([]byte(s))
}

func MustParseString(s string) Size {
	return MustParse([]byte(s))
}

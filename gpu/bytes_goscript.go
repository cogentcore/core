// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && goscript

package gpu

import (
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"syscall/js"
	"unsafe"
)

// sliceToBytes returns the bytes of the given slice. GoScript holds slices
// of fixed-size numbers such as float32 and uint32 in JavaScript typed arrays,
// whose memory is viewed as bytes without copying, as wgpu.ToBytes does in
// native and wasm builds. Any other slice, such as a slice of structs, is
// encoded into a little-endian copy.
func sliceToBytes[E any](src []E) []byte {
	if b, ok := sliceBytes(src); ok {
		return b
	}
	switch s := any(src).(type) {
	case []float32:
		b := make([]byte, 4*len(s))
		for i, v := range s {
			binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(v))
		}
		return b
	case []uint32:
		b := make([]byte, 4*len(s))
		for i, v := range s {
			binary.LittleEndian.PutUint32(b[4*i:], v)
		}
		return b
	case []int32:
		b := make([]byte, 4*len(s))
		for i, v := range s {
			binary.LittleEndian.PutUint32(b[4*i:], uint32(v))
		}
		return b
	case []float64:
		b := make([]byte, 8*len(s))
		for i, v := range s {
			binary.LittleEndian.PutUint64(b[8*i:], math.Float64bits(v))
		}
		return b
	}
	sv := reflect.ValueOf(src)
	size := int(sv.Type().Elem().Size())
	b := make([]byte, size*len(src))
	for i := range len(src) {
		if err := encodeValue(b[i*size:], sv.Index(i)); err != nil {
			panic("gpu: cannot convert slice to bytes: " + err.Error())
		}
	}
	return b
}

// sliceBytes returns the bytes of the memory of the given slice and true
// when GoScript holds it in a JavaScript typed array, which it does for the
// slices it makes of fixed-size numbers up to 32 bits and of float64.
func sliceBytes[E any](src []E) ([]byte, bool) {
	if b, ok := any(src).([]byte); ok {
		return b, true
	}
	if len(src) == 0 || !typedArrayKind(reflect.TypeFor[E]().Kind()) {
		return nil, false
	}
	array := js.ValueOf(src)
	if !array.Get("BYTES_PER_ELEMENT").Truthy() {
		return nil, false
	}
	n := array.Get("byteLength").Int()
	return unsafe.Slice((*byte)(unsafe.Pointer(&src[0])), n), true
}

// typedArrayKind reports whether GoScript holds slices of elements of the
// given kind in JavaScript typed arrays.
func typedArrayKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int8, reflect.Uint8, reflect.Int16, reflect.Uint16,
		reflect.Int32, reflect.Uint32, reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// encodeValue writes the given value into b in little-endian order,
// using the memory layout of the value, including any padding.
func encodeValue(b []byte, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			b[0] = 1
		}
	case reflect.Int8:
		b[0] = byte(v.Int())
	case reflect.Uint8:
		b[0] = byte(v.Uint())
	case reflect.Int16:
		binary.LittleEndian.PutUint16(b, uint16(v.Int()))
	case reflect.Uint16:
		binary.LittleEndian.PutUint16(b, uint16(v.Uint()))
	case reflect.Int32:
		binary.LittleEndian.PutUint32(b, uint32(v.Int()))
	case reflect.Uint32:
		binary.LittleEndian.PutUint32(b, uint32(v.Uint()))
	case reflect.Int64, reflect.Int:
		binary.LittleEndian.PutUint64(b, uint64(v.Int()))
	case reflect.Uint64, reflect.Uint, reflect.Uintptr:
		binary.LittleEndian.PutUint64(b, v.Uint())
	case reflect.Float32:
		binary.LittleEndian.PutUint32(b, math.Float32bits(float32(v.Float())))
	case reflect.Float64:
		binary.LittleEndian.PutUint64(b, math.Float64bits(v.Float()))
	case reflect.Array:
		size := int(v.Type().Elem().Size())
		for i := range v.Len() {
			if err := encodeValue(b[i*size:], v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			if err := encodeValue(b[t.Field(i).Offset:], v.Field(i)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported kind %s", v.Kind())
	}
	return nil
}

// decodeValue sets the given value from the little-endian bytes in b,
// using the memory layout of the value, including any padding.
func decodeValue(b []byte, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(b[0] != 0)
	case reflect.Int8:
		v.SetInt(int64(int8(b[0])))
	case reflect.Uint8:
		v.SetUint(uint64(b[0]))
	case reflect.Int16:
		v.SetInt(int64(int16(binary.LittleEndian.Uint16(b))))
	case reflect.Uint16:
		v.SetUint(uint64(binary.LittleEndian.Uint16(b)))
	case reflect.Int32:
		v.SetInt(int64(int32(binary.LittleEndian.Uint32(b))))
	case reflect.Uint32:
		v.SetUint(uint64(binary.LittleEndian.Uint32(b)))
	case reflect.Int64, reflect.Int:
		v.SetInt(int64(binary.LittleEndian.Uint64(b)))
	case reflect.Uint64, reflect.Uint, reflect.Uintptr:
		v.SetUint(binary.LittleEndian.Uint64(b))
	case reflect.Float32:
		v.SetFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(b))))
	case reflect.Float64:
		v.SetFloat(math.Float64frombits(binary.LittleEndian.Uint64(b)))
	case reflect.Array:
		size := int(v.Type().Elem().Size())
		for i := range v.Len() {
			if err := decodeValue(b[i*size:], v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			if err := decodeValue(b[t.Field(i).Offset:], v.Field(i)); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported kind %s", v.Kind())
	}
	return nil
}

// bytesToSlice decodes the given little-endian bytes into the given slice.
func bytesToSlice[E any](src []byte, dest []E) error {
	switch d := any(dest).(type) {
	case []byte:
		copy(d, src)
		return nil
	case []float32:
		for i := range min(len(d), len(src)/4) {
			d[i] = math.Float32frombits(binary.LittleEndian.Uint32(src[4*i:]))
		}
		return nil
	case []uint32:
		for i := range min(len(d), len(src)/4) {
			d[i] = binary.LittleEndian.Uint32(src[4*i:])
		}
		return nil
	case []int32:
		for i := range min(len(d), len(src)/4) {
			d[i] = int32(binary.LittleEndian.Uint32(src[4*i:]))
		}
		return nil
	case []float64:
		for i := range min(len(d), len(src)/8) {
			d[i] = math.Float64frombits(binary.LittleEndian.Uint64(src[8*i:]))
		}
		return nil
	}
	dv := reflect.ValueOf(dest)
	size := int(dv.Type().Elem().Size())
	for i := range min(len(dest), len(src)/size) {
		if err := decodeValue(src[i*size:], dv.Index(i)); err != nil {
			return fmt.Errorf("gpu: cannot convert bytes to slice: %w", err)
		}
	}
	return nil
}

// readToSlice copies the read buffer data of the given value into the
// given slice: directly into its memory when GoScript holds it in a
// JavaScript typed array, and otherwise by decoding the bytes.
func readToSlice[E any](vl *Value, dest []E) error {
	if b, ok := sliceBytes(dest); ok {
		return vl.ReadToBytes(b)
	}
	var zero E
	b := make([]byte, len(dest)*int(reflect.TypeOf(zero).Size()))
	if err := vl.ReadToBytes(b); err != nil {
		return err
	}
	return bytesToSlice(b, dest)
}

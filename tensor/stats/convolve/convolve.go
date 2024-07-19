// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package convolve

//go:generate core generate

import (
	"errors"

	"cogentcore.org/core/base/slicesx"
)

// Slice32 convolves given kernel with given source slice, putting results in
// destination, which is ensured to be the same size as the source slice,
// using existing capacity if available, and otherwise making a new slice.
// The kernel should be normalized, and odd-sized do it is symmetric about 0.
// Returns an error if sizes are not valid.
// No parallelization is used -- see Slice32Parallel for very large slices.
// Edges are handled separately with renormalized kernels -- they can be
// clipped from dest by excluding the kernel half-width from each end.
func Slice32(dest *[]float32, src []float32, kern []float32) error {
	sz := len(src)
	ksz := len(kern)
	if ksz == 0 || sz == 0 {
		return errors.New("convolve.Slice32: kernel or source are empty")
	}
	if ksz%2 == 0 {
		return errors.New("convolve.Slice32: kernel is not odd sized")
	}
	if sz < ksz {
		return errors.New("convolve.Slice32: source must be > kernel in size")
	}
	khalf := (ksz - 1) / 2
	*dest = slicesx.SetLength(*dest, sz)
	for i := khalf; i < sz-khalf; i++ {
		var sum float32
		for j := 0; j < ksz; j++ {
			sum += src[(i-khalf)+j] * kern[j]
		}
		(*dest)[i] = sum
	}
	for i := 0; i < khalf; i++ {
		var sum, ksum float32
		for j := 0; j <= khalf+i; j++ {
			ki := (j + khalf) - i // 0: 1+kh, 1: etc
			si := i + (ki - khalf)
			// fmt.Printf("i: %d  j: %d  ki: %d  si: %d\n", i, j, ki, si)
			sum += src[si] * kern[ki]
			ksum += kern[ki]
		}
		(*dest)[i] = sum / ksum
	}
	for i := sz - khalf; i < sz; i++ {
		var sum, ksum float32
		ei := sz - i - 1
		for j := 0; j <= khalf+ei; j++ {
			ki := ((ksz - 1) - (j + khalf)) + ei
			si := i + (ki - khalf)
			// fmt.Printf("i: %d  j: %d  ki: %d  si: %d  ei: %d\n", i, j, ki, si, ei)
			sum += src[si] * kern[ki]
			ksum += kern[ki]
		}
		(*dest)[i] = sum / ksum
	}
	return nil
}

// Slice64 convolves given kernel with given source slice, putting results in
// destination, which is ensured to be the same size as the source slice,
// using existing capacity if available, and otherwise making a new slice.
// The kernel should be normalized, and odd-sized do it is symmetric about 0.
// Returns an error if sizes are not valid.
// No parallelization is used -- see Slice64Parallel for very large slices.
// Edges are handled separately with renormalized kernels -- they can be
// clipped from dest by excluding the kernel half-width from each end.
func Slice64(dest *[]float64, src []float64, kern []float64) error {
	sz := len(src)
	ksz := len(kern)
	if ksz == 0 || sz == 0 {
		return errors.New("convolve.Slice64: kernel or source are empty")
	}
	if ksz%2 == 0 {
		return errors.New("convolve.Slice64: kernel is not odd sized")
	}
	if sz < ksz {
		return errors.New("convolve.Slice64: source must be > kernel in size")
	}
	khalf := (ksz - 1) / 2
	*dest = slicesx.SetLength(*dest, sz)
	for i := khalf; i < sz-khalf; i++ {
		var sum float64
		for j := 0; j < ksz; j++ {
			sum += src[(i-khalf)+j] * kern[j]
		}
		(*dest)[i] = sum
	}
	for i := 0; i < khalf; i++ {
		var sum, ksum float64
		for j := 0; j <= khalf+i; j++ {
			ki := (j + khalf) - i // 0: 1+kh, 1: etc
			si := i + (ki - khalf)
			// fmt.Printf("i: %d  j: %d  ki: %d  si: %d\n", i, j, ki, si)
			sum += src[si] * kern[ki]
			ksum += kern[ki]
		}
		(*dest)[i] = sum / ksum
	}
	for i := sz - khalf; i < sz; i++ {
		var sum, ksum float64
		ei := sz - i - 1
		for j := 0; j <= khalf+ei; j++ {
			ki := ((ksz - 1) - (j + khalf)) + ei
			si := i + (ki - khalf)
			// fmt.Printf("i: %d  j: %d  ki: %d  si: %d  ei: %d\n", i, j, ki, si, ei)
			sum += src[si] * kern[ki]
			ksum += kern[ki]
		}
		(*dest)[i] = sum / ksum
	}
	return nil
}

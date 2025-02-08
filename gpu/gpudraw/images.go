// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

// imgrec records tracking data for a specific image
type imgrec struct {
	// img is the image, or texture
	img any

	// index is where it is used in the Values list.
	index int

	// used flag indicates whether this image was used on last pass.
	// all unused images are recycled.
	used bool
}

// images manages the current set of images
type images struct {
	// all records all the images used.
	all map[any]*imgrec

	// capacity is the total number of Values slots currently available.
	capacity int

	// free is a list of all the available Values indexes
	// up to the current number of total Values,
	// initialized in *reverse* sorted order, so the last item
	// is consumed first (saves repacking the list every time,
	// works like a stack).
	free []int

	// used is the sequential list of images used this pass,
	// in one-to-one correspondence with image use in opList.
	used []*imgrec
}

// init does initialization for AllocChunk total images
func (im *images) init() {
	n := AllocChunk
	im.all = make(map[any]*imgrec, n)
	im.free = make([]int, n)
	for i := range n {
		im.free[i] = n - 1 - i
	}
	im.used = make([]*imgrec, 0, n)
}

// start gets ready for the next pass, resetting used
// flags and recycling unused indexes.
func (im *images) start() {
	im.used = im.used[:0]
	for _, ir := range im.all {
		if !ir.used {
			im.free = append(im.free, ir.index)
			delete(im.all, ir.img)
		} else {
			ir.used = false
		}
	}
}

// use uses the given image, either re-using or taking a new spot.
// returns the value index to store it at, and whether it already existed.
// If there are no more free spots, then the index will be at the
// previous capacity level, which should be len(Values), at which
// point the drawer can allocate the current capacity amount.
func (im *images) use(img any) (int, bool) {
	if ir, ok := im.all[img]; ok {
		ir.used = true
		im.used = append(im.used, ir)
		return ir.index, true
	}
	ni := -1
	nf := len(im.free)
	if nf == 0 {
		ni = im.capacity
		im.capacity += AllocChunk
		for i := ni + 1; i < im.capacity; i++ {
			im.free = append(im.free, im.capacity-1-i)
		}
	} else {
		ni = im.free[nf-1]
		im.free = im.free[:nf-1]
	}
	ir := &imgrec{img: img, index: ni, used: true}
	im.all[img] = ir
	im.used = append(im.used, ir)
	return ni, false
}

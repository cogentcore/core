// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vgpu

import (
	"fmt"
	"image"
	"sort"

	"github.com/goki/ki/ints"
	"github.com/goki/mat32"
)

// ImgIdxs contains the indexes where a given Image value is allocated
type ImgIdxs struct {
	PctSize mat32.Vec2 `desc:"percent size of this image relative to max size allocated"`
	ImgIdx  int        `desc:"index of the image (0-15)"`
	LayIdx  int        `desc:"layer within image (0-127)"`
}

func (ii *ImgIdxs) Set(imgi, layi int, sz, msz image.Point) {
	ii.ImgIdx = imgi
	ii.LayIdx = layi
	ii.PctSize.X = float32(sz.X) / float32(msz.X)
	ii.PctSize.Y = float32(sz.Y) / float32(msz.Y)
}

// SizeN represents number of images at each size
type SizeN struct {
	Size image.Point
	N    int
}

// ImageMgr manages allocation of images by size within the MaxTexturesPerSet
// (16) constraint.  There are 128 Layers per Image, which we alloc
type ImageMgr struct {
	AllocSizes []image.Point       `desc:"list of allocated image sizes"`
	ValAlloc   [][]int             `desc:"allocation of image indexes by size"`
	ValIdxs    []*ImgIdxs          `desc:"allocation image value indexes to image indexes"`
	AllSizes   map[image.Point]int `desc:"map of all unique sizes"`
}

// PctDiff returns the percent difference vs. max of two vals
func PctDiff(a, b int) float32 {
	mx := ints.MaxInt(a, b)
	if mx == 0 {
		return 0
	}
	d := ints.AbsInt(a - b)
	return float32(d) / float32(mx)
}

// Alloc allocates vals as a function of size
func (ir *ImageMgr) Alloc(vals *Vals) {
	nv := len(vals.Vals)
	if nv == 0 {
		return
	}
	order := make([]int, nv)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(i, j int) bool {
		visz := vals.Vals[order[i]].Texture.Format.Size
		vjsz := vals.Vals[order[j]].Texture.Format.Size
		// pctdx := PctDiff(visz.X, vjsz.X)
		// pctdy := PctDiff(visz.Y, vjsz.Y)
		// if pctdx < .2 && pctdy < .2 { // if close, sort by X
		return visz.X < vjsz.X
		// }
		// iarea := visz.X * visz.Y
		// jarea := vjsz.X * vjsz.Y
		// return iarea < jarea
	})

	ir.AllocSizes = make([]image.Point, 0, nv)
	ir.AllSizes = make(map[image.Point]int, nv)
	for _, vi := range order {
		vsz := vals.Vals[vi].Texture.Format.Size
		n, has := ir.AllSizes[vsz]
		n++
		if !has {
			ir.AllocSizes = append(ir.AllocSizes, vsz)
		}
		ir.AllSizes[vsz] = n
	}
	if len(ir.AllSizes) >= MaxTexturesPerSet {
		prvalloc := make([]image.Point, 0, nv)
		copy(prvalloc[0:len(ir.AllocSizes)], ir.AllocSizes)
		ir.AllocSizes = ir.AllocSizes[:1]
		lstsz := prvalloc[0]
		// group together by area, progresively until under thr
		areathr := float32(0.1)
		for {
			for i, sz := range prvalloc {
				if i == 0 {
					continue
				}
				pd := PctDiff(lstsz.X*lstsz.Y, sz.X*sz.Y)
				if pd < areathr {
					continue
				}
				lstsz = sz
				ir.AllocSizes = append(ir.AllocSizes, sz)
			}
			if len(ir.AllSizes) < MaxTexturesPerSet {
				break
			}
			copy(prvalloc[0:len(ir.AllocSizes)], ir.AllocSizes)
			ir.AllocSizes = ir.AllocSizes[:1]
			lstsz = prvalloc[0]
			areathr += .1
		}
	}
	ng := len(ir.AllocSizes)
	lst := order[len(order)-1]
	ir.AllocSizes[ng-1] = vals.Vals[lst].Texture.Format.Size
	gi := 0
	li := 0
	gsz := ir.AllocSizes[0]
	ir.ValIdxs = make([]*ImgIdxs, nv)
	ir.ValAlloc = make([][]int, ng)
	for i, vi := range order {
		vsz := vals.Vals[vi].Texture.Format.Size
		if vsz.X <= gsz.X && vsz.Y <= gsz.Y {
			li = len(ir.ValAlloc[gi])
			ir.ValAlloc[gi] = append(ir.ValAlloc[gi], vi)
		} else {
			li = 0
			gi++
			gsz = ir.AllocSizes[gi]
			ir.ValAlloc[gi] = append(ir.ValAlloc[gi], vi)
		}
		ii := &ImgIdxs{}
		ii.Set(gi, li, vsz, gsz)
		ir.ValIdxs[vi] = ii
		fmt.Printf("idx: %2d  img: %2d  sz: %v  gsz: %v  gi: %2d  li: %2d\n", i, vi, vsz, gsz, gi, li)
	}
}

// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clust

import (
	"math"
)

// DistFunc is a clustering distance function that evaluates aggregate distance
// between nodes, given the indexes of leaves in a and b clusters
// which are indexs into an ntot x ntot similarity (distance) matrix smat.
// maxd is the maximum distance value in the smat, which is needed by the
// ContrastDist function and perhaps others.
type DistFunc func(aix, bix []int, ntot int, maxd float64, smat []float64) float64

// MinDist is the minimum-distance or single-linkage weighting function for comparing
// two clusters a and b, given by their list of indexes.
// ntot is total number of nodes, and smat is the square similarity matrix [ntot x ntot].
func MinDist(aix, bix []int, ntot int, maxd float64, smat []float64) float64 {
	md := math.MaxFloat64
	for _, ai := range aix {
		for _, bi := range bix {
			d := smat[ai*ntot+bi]
			if d < md {
				md = d
			}
		}
	}
	return md
}

// MaxDist is the maximum-distance or complete-linkage weighting function for comparing
// two clusters a and b, given by their list of indexes.
// ntot is total number of nodes, and smat is the square similarity matrix [ntot x ntot].
func MaxDist(aix, bix []int, ntot int, maxd float64, smat []float64) float64 {
	md := -math.MaxFloat64
	for _, ai := range aix {
		for _, bi := range bix {
			d := smat[ai*ntot+bi]
			if d > md {
				md = d
			}
		}
	}
	return md
}

// AvgDist is the average-distance or average-linkage weighting function for comparing
// two clusters a and b, given by their list of indexes.
// ntot is total number of nodes, and smat is the square similarity matrix [ntot x ntot].
func AvgDist(aix, bix []int, ntot int, maxd float64, smat []float64) float64 {
	md := 0.0
	n := 0
	for _, ai := range aix {
		for _, bi := range bix {
			d := smat[ai*ntot+bi]
			md += d
			n++
		}
	}
	if n > 0 {
		md /= float64(n)
	}
	return md
}

// ContrastDist computes maxd + (average within distance - average between distance)
// for two clusters a and b, given by their list of indexes.
// avg between is average distance between all items in a & b versus all outside that.
// ntot is total number of nodes, and smat is the square similarity matrix [ntot x ntot].
// maxd is the maximum distance and is needed to ensure distances are positive.
func ContrastDist(aix, bix []int, ntot int, maxd float64, smat []float64) float64 {
	wd := AvgDist(aix, bix, ntot, maxd, smat)
	nab := len(aix) + len(bix)
	abix := append(aix, bix...)
	abmap := make(map[int]struct{}, ntot-nab)
	for _, ix := range abix {
		abmap[ix] = struct{}{}
	}
	oix := make([]int, ntot-nab)
	octr := 0
	for ix := 0; ix < ntot; ix++ {
		if _, has := abmap[ix]; !has {
			oix[octr] = ix
			octr++
		}
	}
	bd := AvgDist(abix, oix, ntot, maxd, smat)
	return maxd + (wd - bd)
}

// StdDists are standard clustering distance functions
type StdDists int32 //enums:enum

const (
	// Min is the minimum-distance or single-linkage weighting function
	Min StdDists = iota

	// Max is the maximum-distance or complete-linkage weighting function
	Max

	// Avg is the average-distance or average-linkage weighting function
	Avg

	// Contrast computes maxd + (average within distance - average between distance)
	Contrast
)

// StdFunc returns a standard distance function as specified
func StdFunc(std StdDists) DistFunc {
	switch std {
	case Min:
		return MinDist
	case Max:
		return MaxDist
	case Avg:
		return AvgDist
	case Contrast:
		return ContrastDist
	}
	return nil
}

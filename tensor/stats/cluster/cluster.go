// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cluster

//go:generate core generate

import (
	"fmt"
	"math"
	"math/rand"

	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/stats"
)

// todo: all of this data goes into the datafs
// Cluster makes a new dir, stuffs results in there!
// need a global "cwd" that it uses, so basically you cd
// to a dir, then cal it.

// Node is one node in the cluster
type Node struct {
	// index into original distance matrix; only valid for for terminal leaves.
	Index int

	// Distance value for this node, i.e., how far apart were all the kids from
	// each other when this node was created. is 0 for leaf nodes
	Dist float64

	// ParDist is total aggregate distance from parents; The X axis offset at which our cluster starts.
	ParDist float64

	// Y is y-axis value for this node; if a parent, it is the average of its kids Y's,
	// otherwise it counts down.
	Y float64

	// Kids are child nodes under this one.
	Kids []*Node
}

// IsLeaf returns true if node is a leaf of the tree with no kids
func (nn *Node) IsLeaf() bool {
	return len(nn.Kids) == 0
}

// Sprint prints to string
func (nn *Node) Sprint(labels *tensor.Indexed, depth int) string {
	if nn.IsLeaf() && labels != nil {
		return labels.Tensor.String1D(nn.Index) + " "
	}
	sv := fmt.Sprintf("\n%v%v: ", indent.Tabs(depth), nn.Dist)
	for _, kn := range nn.Kids {
		sv += kn.Sprint(labels, depth+1)
	}
	return sv
}

// Indexes collects all the indexes in this node
func (nn *Node) Indexes(ix []int, ctr *int) {
	if nn.IsLeaf() {
		ix[*ctr] = nn.Index
		(*ctr)++
	} else {
		for _, kn := range nn.Kids {
			kn.Indexes(ix, ctr)
		}
	}
}

// NewNode merges two nodes into a new node
func NewNode(na, nb *Node, dst float64) *Node {
	nn := &Node{Dist: dst}
	nn.Kids = []*Node{na, nb}
	return nn
}

// TODO: this call signature does not fit with standard
// not sure how one might pack Node into a tensor

// Cluster implements agglomerative clustering, based on a
// distance matrix dmat, e.g., as computed by metric.Matrix method,
// using a metric that increases in value with greater dissimilarity.
// labels provides an optional String tensor list of labels for the elements
// of the distance matrix.
// This calls InitAllLeaves to initialize the root node with all of the leaves,
// and then Glom to do the iterative agglomerative clustering process.
// If you want to start with pre-defined initial clusters,
// then call Glom with a root node so-initialized.
func Cluster(funcName string, dmat, labels *tensor.Indexed) *Node {
	ntot := dmat.Tensor.DimSize(0) // number of leaves
	root := InitAllLeaves(ntot)
	return Glom(root, funcName, dmat)
}

// InitAllLeaves returns a standard root node initialized with all of the leaves.
func InitAllLeaves(ntot int) *Node {
	root := &Node{}
	root.Kids = make([]*Node, ntot)
	for i := 0; i < ntot; i++ {
		root.Kids[i] = &Node{Index: i}
	}
	return root
}

// Glom does the iterative agglomerative clustering,
// based on a raw similarity matrix as given,
// using a root node that has already been initialized
// with the starting clusters, which is all of the
// leaves by default, but could be anything if you want
// to start with predefined clusters.
func Glom(root *Node, funcName string, dmat *tensor.Indexed) *Node {
	ntot := dmat.Tensor.DimSize(0) // number of leaves
	mout := tensor.NewFloatScalar(0)
	stats.MaxFunc(tensor.NewIndexed(tensor.New1DViewOf(dmat.Tensor)), mout)
	maxd := mout.Tensor.Float1D(0)
	// indexes in each group
	aidx := make([]int, ntot)
	bidx := make([]int, ntot)
	for {
		var ma, mb []int
		mval := math.MaxFloat64
		for ai, ka := range root.Kids {
			actr := 0
			ka.Indexes(aidx, &actr)
			aix := aidx[0:actr]
			for bi := 0; bi < ai; bi++ {
				kb := root.Kids[bi]
				bctr := 0
				kb.Indexes(bidx, &bctr)
				bix := bidx[0:bctr]
				dv := Call(funcName, aix, bix, ntot, maxd, dmat.Tensor)
				if dv < mval {
					mval = dv
					ma = []int{ai}
					mb = []int{bi}
				} else if dv == mval { // do all ties at same time
					ma = append(ma, ai)
					mb = append(mb, bi)
				}
			}
		}
		ni := 0
		if len(ma) > 1 {
			ni = rand.Intn(len(ma))
		}
		na := ma[ni]
		nb := mb[ni]
		nn := NewNode(root.Kids[na], root.Kids[nb], mval)
		for i := len(root.Kids) - 1; i >= 0; i-- {
			if i == na || i == nb {
				root.Kids = append(root.Kids[:i], root.Kids[i+1:]...)
			}
		}
		root.Kids = append(root.Kids, nn)
		if len(root.Kids) == 1 {
			break
		}
	}
	return root
}

// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package physics

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

// Contact is one pairwise point of contact between two bodies.
// Contacts are represented in spherical terms relative to the
// spherical BBox of A and B.
type Contact struct {

	// one body
	A Body

	// the other body
	B Body

	// normal pointing from center of B to center of A
	NormB math32.Vector3

	// point on spherical shell of B where A is contacting
	PtB math32.Vector3

	// distance from PtB along NormB to contact point on spherical shell of A
	Dist float32
}

// UpdtDist updates the distance information for the contact
func (c *Contact) UpdtDist() {

}

// Contacts is a slice list of contacts
type Contacts []*Contact

// New adds a new contact to the list
func (cs *Contacts) New(a, b Body) *Contact {
	c := &Contact{A: a, B: b}
	*cs = append(*cs, c)
	return c
}

// BodyVelBBoxIntersects returns the list of potential contact nodes between a and b
// (could be the same or different groups) that have intersecting velocity-projected
// bounding boxes.  In general a should be dynamic bodies and b either dynamic or static.
// This is the broad first-pass filtering.
func BodyVelBBoxIntersects(a, b Node) Contacts {
	var cts Contacts
	a.WalkDown(func(k tree.Node) bool {
		aii, ai := AsNode(k)
		if aii == nil {
			return false // going into a different type of thing, bail
		}
		if aii.EveNodeType() != BODY {
			return true
		}
		abod := aii.AsBody() // only consider bodies from a

		b.WalkDown(func(k tree.Node) bool {
			bii, bi := AsNode(k)
			if bii == nil {
				return false // going into a different type of thing, bail
			}
			if !ai.BBox.IntersectsVelBox(&bi.BBox) {
				return false // done
			}
			if bii.EveNodeType() == BODY {
				cts.New(abod, bii.AsBody())
				return false // done
			}
			return true // keep going
		})

		return false
	})
	return cts
}

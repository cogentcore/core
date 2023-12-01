// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uri

// Resource is a source of URIs
type Resource func() URIs

// Resources is a list of resources
type Resources []Resource

// Add adds the given resource
func (rs *Resources) Add(r Resource) *Resources {
	*rs = append(*rs, r)
	return rs
}

func (rs *Resources) Generate() URIs {
	var ul URIs
	for _, r := range *rs {
		u := r()
		if len(u) == 0 {
			continue
		}
		ul = append(ul, u...)
	}
	return ul
}

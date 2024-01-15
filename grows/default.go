// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grows

// NonDefaultFields returns a map representing all of the fields of the given
// struct (or pointer to a struct) that have values different than their default
// values as specified by the 'default' struct tag. This map can then be saved
// using jsons, tomls, xmls, yamls, or another package.
func NonDefaultFields(v any) map[string]any {
	return nil
}

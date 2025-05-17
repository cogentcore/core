// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import (
	"fmt"
	"slices"
	"strings"

	"cogentcore.org/core/base/keylist"
)

// KeyList is an ordered list of citation [Item]s,
// which should be used to collect items by unique citation keys.
type KeyList struct {
	keylist.List[string, *Item]
}

// NewKeyList returns a KeyList from given list of [Item]s.
func NewKeyList(items []Item) *KeyList {
	kl := &KeyList{}
	for i := range items {
		it := &items[i]
		kl.Add(it.CitationKey, it)
	}
	return kl
}

// AlphaKeys returns an alphabetically sorted list of keys.
func (kl *KeyList) AlphaKeys() []string {
	ks := slices.Clone(kl.Keys)
	slices.Sort(ks)
	return ks
}

// PrettyString pretty prints the items using default style.
func (kl *KeyList) PrettyString() string {
	var w strings.Builder
	for _, it := range kl.Values {
		w.WriteString(fmt.Sprintf("%s [%s]:\n", it.CitationKey, it.Type))
		w.WriteString(string(Ref(DefaultStyle, it).Join()) + "\n\n")
	}
	return w.String()
}

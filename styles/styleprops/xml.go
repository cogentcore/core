// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styleprops

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/reflectx"
)

// FromXMLString sets style properties from XML style string, which contains ';'
// separated name: value pairs
func FromXMLString(style string, properties map[string]any) {
	st := strings.Split(style, ";")
	for _, s := range st {
		kv := strings.Split(s, ":")
		n := len(kv)
		if n >= 2 {
			k := strings.TrimSpace(strings.ToLower(kv[n-2]))
			if n == 3 { // prefixed name
				k = strings.TrimSpace(strings.ToLower(kv[0])) + ":" + k
			}
			v := strings.TrimSpace(kv[n-1])
			properties[k] = v
		}
	}
}

// ToXMLString returns an XML style string from given style properties map
// using ';' separated name: value pairs.
func ToXMLString(properties map[string]any) string {
	var sb strings.Builder
	for k, v := range properties {
		if k == "transform" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s:%s;", k, reflectx.ToString(v)))
	}
	return sb.String()
}

// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"strings"
)

// ContainerCond is the condition of an @container rule: the name of the
// container that it queries ("" for the nearest one), and its query (as
// (width <= 799px)), and that of an @container rule that it is in.
type ContainerCond struct {
	Name, Query string
	Outer       *ContainerCond
}

// ParseContainerCond returns the condition of an @container rule of the
// given prelude (as "spotlight (width <= 799px)"), in the given one.
func ParseContainerCond(prelude string, outer *ContainerCond) *ContainerCond {
	prelude = strings.TrimSpace(prelude)
	cc := &ContainerCond{Query: prelude, Outer: outer}
	if i := strings.IndexAny(prelude, "( "); i > 0 && !strings.HasPrefix(strings.ToLower(prelude), "not") {
		name := strings.TrimSpace(prelude[:i])
		if name != "" && !strings.ContainsAny(name, "()") {
			cc.Name = name
			cc.Query = strings.TrimSpace(prelude[i:])
		}
	}
	return cc
}

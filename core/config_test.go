// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"testing"
)

func TestConfig(t *testing.T) {
	var c Config
	c.Add("parts", nil, nil)
	c.Add("parts/icon", nil, nil)
	c.Add("parts/icon/parts", nil, nil)
	c.Add("parts/text", nil, nil)
	c.Add("tree", nil, nil)
	c.Add("tree/child1", nil, nil)
	c.Add("tree/child2", nil, nil)
	fmt.Println(c.String())
}

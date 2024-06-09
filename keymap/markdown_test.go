// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keymap

import (
	"fmt"
	"testing"
)

func TestMarkdown(t *testing.T) {
	str := AvailableMaps.MarkdownDoc()
	fmt.Println(str)
}

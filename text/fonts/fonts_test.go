// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fonts

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
	"github.com/stretchr/testify/assert"
)

func TestFixMedium(t *testing.T) {
	t.Skip("font debugging")
	const (
		nameFontFamily         tables.NameID = 1
		nameFontSubfamily      tables.NameID = 2
		namePreferredFamily    tables.NameID = 16 // or Typographic Family
		namePreferredSubfamily tables.NameID = 17 // or Typographic Subfamily
		nameWWSFamily          tables.NameID = 21 //
		nameWWSSubfamily       tables.NameID = 22 //
	)

	b, err := os.ReadFile("noto/NotoSans-Medium.ttf")
	assert.NoError(t, err)
	faces, err := font.ParseTTC(bytes.NewReader(b))
	assert.NoError(t, err)
	for i, fnt := range faces {
		d := fnt.Describe()
		fmt.Println("index:", i, "family:", d.Family, "Aspect:", d.Aspect)

		lds, err := ot.NewLoaders(bytes.NewReader(b))
		assert.NoError(t, err)
		ld := lds[i]
		buffer, _ := ld.RawTableTo(ot.MustNewTag("name"), nil)
		names, _, _ := tables.ParseName(buffer)
		// fmt.Println(names)
		fmt.Println("ff:", names.Name(nameFontFamily))
	}
}

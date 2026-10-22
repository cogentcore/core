// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command pelletier writes testdata/pelletier.toml for the tomlx
// compatibility tests, using pelletier/go-toml/v2 with the same encoder
// settings as the previous version of tomlx. It needs the test types,
// so run it from this directory with:
//
//	sed 's/^package tomlx/package main/' ../../compattypes_test.go > types.go
//	go run . ../pelletier.toml
//	rm types.go
package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"

	"github.com/pelletier/go-toml/v2"
)

func main() {
	v := compatValue()
	var b bytes.Buffer
	err := toml.NewEncoder(&b).SetIndentTables(true).SetArraysMultiline(true).Encode(v)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[1], b.Bytes(), 0666); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// check that the previous version reads it back (except for EmptyMap,
	// which it reads as nil, and NilSlice, which it reads as empty).
	var r compatData
	if err := toml.Unmarshal(b.Bytes(), &r); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	r.EmptyMap, r.NilSlice = v.EmptyMap, v.NilSlice
	if !reflect.DeepEqual(v, &r) {
		fmt.Println("pelletier/go-toml/v2 did not read back the same value")
		os.Exit(1)
	}
}

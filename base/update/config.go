// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package update

import "cogentcore.org/core/base/namer"

// ConfigItem represents one configuration element for specifying a slice of
// elements by unique names
type ConfigItem[T namer.Namer] struct {
	Name string
	New  func(name string) T
}

type Config[T namer.Namer] []ConfigItem[T]

func UpdateConfig[T namer.Namer](s []T, c Config[T]) (r []T, mods bool) {
	return Update(s, len(c),
		func(i int) string { return c[i].Name },
		func(name string, i int) T { return c[i].New(name) })
}

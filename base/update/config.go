// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package update

import "cogentcore.org/core/base/namer"

// ConfigItem represents one configuration element for specifying a slice of
// elements by unique names.  Items must have Name() and SetName() methods,
// so that the New function has a minimal signature.
type ConfigItem[T namer.SetNamer] struct {
	Name string
	New  func() T
}

// Config is a slice of ConfigItems for specifying the configuration
// of elements in a slice, and a New function for making any elements
// as needed.
type Config[T namer.SetNamer] []ConfigItem[T]

// Add adds a ConfigItem for given name and new function.
func (c *Config[T]) Add(name string, newEl func() T) {
	*c = append(*c, ConfigItem[T]{Name: name, New: newEl})
}

// UpdateConfig updates the given slice to match elements specified by
// the given Config configuration, using efficient, minimal updates.
// returns the updated slice, and a bool indicating any modifications.
func UpdateConfig[T namer.SetNamer](s []T, c Config[T]) (r []T, mods bool) {
	return Update(s, len(c),
		func(i int) string { return c[i].Name },
		func(name string, i int) T { ne := c[i].New(); ne.SetName(name); return ne })
}

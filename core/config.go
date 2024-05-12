// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "cogentcore.org/core/base/ordmap"

// ConfigItem represents a Widget configuration element,
// with one New function to create and configure a new child Widget,
// and an Update function to update the state of that child Widget.
type ConfigItem struct {
	// New returns a new Widget of the correct type for this element,
	// fully configured and ready for use.
	New func() Widget

	// Update updates the widget based on current state, so that it
	// propertly renders the correct information.
	Update func(w Widget)
}

// Config is an ordered list of [ConfigItem]s,
// ordered in progressive hierarchical order
// so that parents are listed before any children.
// The Key of the map is the path of the element.
type Config ordmap.Map[string, ConfigItem]

// Add
func (c *Config) Add(path string, nw func() Widget, updt func(w Widget)) {

}

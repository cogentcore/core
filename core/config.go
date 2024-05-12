// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "strings"

// ConfigItem represents a Widget configuration element,
// with one New function to create and configure a new child Widget,
// and an Update function to update the state of that child Widget.
// Contains a list of Config for its children, which is all sorted
// out as items are added.
type ConfigItem struct {
	// Path is the / delimited path to the element
	Path string

	// New returns a new Widget of the correct type for this element,
	// fully configured and ready for use.
	New func() Widget

	// Update updates the widget based on current state, so that it
	// propertly renders the correct information.
	Update func(w Widget)

	// Config for Children elements.
	Children Config
}

// Config is an ordered list of [ConfigItem]s,
// ordered in progressive hierarchical order
// so that parents are listed before any children.
// The Key of the map is the path of the element.
type Config []*ConfigItem

// Add adds given config.  This should be called on the root level Config
// list.  Any items with nested paths are added to Children lists as
// appropriate.
func (c *Config) Add(path string, nw func() Widget, updt func(w Widget)) {
	itm := &ConfigItem{Path: path, New: nw, Update: updt}
	plist := strings.Split(path, "/")
	if len(plist) == 1 {
		*c = append(*c, itm)
		return
	}
	next := c.FindMakeChild(plist[0])
	next.AddSubItem(plist[1:], itm)
}

// AddSubItem adds given sub item
func (c *ConfigItem) AddSubItem(path []string, itm *ConfigItem) {
}

func (c *Config) FindMakeChild(name string) *ConfigItem {
	return nil
}

// ConfigWidget is the base implementation of [Widget.ConfigWidget] that
// configures the widget by doing steps that apply to all widgets and then
// calling [Widget.Config] for widget-specific configuration steps.
func (wb *WidgetBase) ConfigWidget() {
	if wb.ValueUpdate != nil {
		wb.ValueUpdate()
	}
	wb.This().(Widget).Config()
}

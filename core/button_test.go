// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
	"github.com/stretchr/testify/assert"
)

func TestButtonText(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Download")
	b.AssertRender(t, "button/text")
}

func TestButtonIcon(t *testing.T) {
	b := NewBody()
	NewButton(b).SetIcon(icons.Download)
	b.AssertRender(t, "button/icon")
}

func TestButtonTextIcon(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Download").SetIcon(icons.Download)
	b.AssertRender(t, "button/text-icon")
}

func TestButtonClick(t *testing.T) {
	b := NewBody()
	clicked := false
	bt := NewButton(b).OnClick(func(e events.Event) {
		clicked = true
	})
	bt.Send(events.Click)
	assert.True(t, clicked)
}

func TestButtonMenu(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Share").SetIcon(icons.Share).SetMenu(func(m *Scene) {
		NewButton(m).SetText("Copy link")
		NewButton(m).SetText("Send message")
	})
	b.AssertRender(t, "button/menu")
}

func TestButtonMenuClick(t *testing.T) {
	b := NewBody()
	b.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(20), units.Em(10))
	})
	bt := NewButton(b).SetText("Share").SetIcon(icons.Share).SetMenu(func(m *Scene) {
		NewButton(m).SetText("Copy link")
		NewButton(m).SetText("Send message")
	})
	b.AssertScreenRender(t, "button/menu-click", func() {
		bt.Send(events.Click)
	})
}

func TestButton(t *testing.T) {
	for typi, typ := range ButtonTypesValues() {
		for stri, str := range testStrings {
			for ici, ic := range testIcons {
				if stri == 0 && ici == 0 {
					continue
				}
				for sti, st := range testStates {
					// we do not test other types and states of the rest
					// of the strings, as that is a waste of
					// testing time
					if stri > 1 && (typi > 0 || sti > 0) {
						continue
					}
					b := NewBody()
					bt := NewButton(b).SetType(typ).SetText(str).SetIcon(ic).SetState(true, st...)
					nm := testName("button", typ, str, ic, bt.Styles.State)
					b.AssertRender(t, nm)
				}
			}
		}
	}
}

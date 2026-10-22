// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

func TestBind(t *testing.T) {
	b := NewBody()
	Bind("Gopher", NewTextField(b))
	b.AssertRender(t, "bind/basic")
}

func TestBindUpdate(t *testing.T) {
	b := NewBody()
	name := "Gopher"
	tf := Bind(&name, NewTextField(b))
	b.AssertRender(t, "bind/update", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		name = "Cogent Core"
		tf.Update()
	})
}

func TestBindChange(t *testing.T) {
	b := NewBody()
	name := "Gopher"

	tf := Bind(&name, NewTextField(b))
	b.AssertRender(t, "bind/change", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.HandleEvent(events.NewKey(events.KeyChord, 'G', 0, 0))
		tf.HandleEvent(events.NewKey(events.KeyChord, 'o', 0, 0))
		tf.HandleEvent(events.NewKey(events.KeyChord, ' ', 0, 0))
		assert.Equal(t, "Gopher", name)
		tf.HandleEvent(events.NewKey(events.KeyChord, 0, key.CodeReturnEnter, 0))
		assert.Equal(t, "Go Gopher", name)
	})
}

func TestBindSpinner(t *testing.T) {
	b := NewBody()
	Bind("1.4", NewSpinner(b))
	b.AssertRender(t, "bind/spinner")
}

func TestBindSlider(t *testing.T) {
	b := NewBody()
	Bind(0.65, NewSpinner(b))
	b.AssertRender(t, "bind/slider")
}

func TestBindMeter(t *testing.T) {
	b := NewBody()
	Bind(15*time.Second, NewMeter(b)).SetMin(float32(time.Second)).SetMax(float32(time.Minute))
	b.AssertRender(t, "bind/meter")
}

func TestBindText(t *testing.T) {
	b := NewBody()
	Bind("Hello", NewText(b))
	b.AssertRender(t, "bind/text")
}

type validator string

func (v validator) Validate() error {
	if !strings.Contains(string(v), "@") {
		return errors.New("must have an @")
	}
	return nil
}

func TestValidatorValid(t *testing.T) {
	b := NewBody()
	tf := Bind(validator("my@string"), NewTextField(b))
	b.AssertRender(t, "bind/validator-valid", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tf.SendChange() // trigger validation
	})
}

func TestValidatorInvalid(t *testing.T) {
	b := NewBody()
	v := Bind(validator("my string"), NewTextField(b))
	b.AssertRender(t, "bind/validator-invalid", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		v.SendChange() // trigger validation
	})
}

// Accepting the dialog of a value whose widget has been deleted meanwhile,
// as when the form that has it is remade, does nothing instead of crashing.
func TestValueDialogDeletedWidget(t *testing.T) {
	b := NewBody()
	fb := NewFileButton(b)
	b.runAndShowNewWindow()
	rw := b.Scene.RenderWindow()
	b.waitNoEvents(rw)

	accepted := false
	b.AsyncLock()
	openValueDialog(fb, func(d *Body) {
		NewText(d).SetText("Dialog")
	}, func() {
		accepted = true
	})
	b.AsyncUnlock()
	b.waitNoEvents(rw)

	b.AsyncLock()
	var ok *Button
	rw.mains.top().Scene.WalkDown(func(n tree.Node) bool {
		if bt := AsButton(n); bt != nil && bt.Text == "OK" {
			ok = bt
			return tree.Break
		}
		return tree.Continue
	})
	fb.Delete()
	if assert.NotNil(t, ok) {
		ok.Send(events.Click)
	}
	b.AsyncUnlock()
	b.waitNoEvents(rw)
	assert.False(t, accepted)

	rc := b.Scene.renderContext()
	rc.Lock()
	b.Close()
	rc.Unlock()
}

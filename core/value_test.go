// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
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
		name = "Cogent Core"
		tf.Update()
	})
}

func TestBindChange(t *testing.T) {
	b := NewBody()
	name := "Gopher"

	tf := Bind(&name, NewTextField(b))
	b.AssertRender(t, "bind/change", func() {
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

/* TODO(config)

type validator string

func (v *validator) Validate() error {
	if !strings.Contains(string(*v), "@") {
		return errors.New("must have an @")
	}
	return nil
}

func TestValidatorValid(t *testing.T) {
	b := NewBody()
	v := NewValue(b, validator("my@string"))
	b.AssertRender(t, "text/validator-valid", func() {
		v.AsWidgetBase().SendChange() // trigger validation
	})
}

func TestValidatorInvalid(t *testing.T) {
	b := NewBody()
	v := NewValue(b, validator("my string"))
	b.AssertRender(t, "text/validator-invalid", func() {
		v.AsWidgetBase().SendChange() // trigger validation
	})
}

type fieldValidator struct {
	Name  string
	Email string
}

func (v *fieldValidator) ValidateField(field string) error {
	switch field {
	case "Name":
		if !strings.Contains(v.Name, " ") {
			return errors.New("need full name")
		}
	case "Email":
		if !strings.Contains(v.Email, "@") || !strings.Contains(v.Email, ".") {
			return errors.New("must have a . and @")
		}
	}
	return nil
}

func TestFieldValidatorValid(t *testing.T) {
	b := NewBody()
	v := NewForm(b).SetStruct(&fieldValidator{Name: "Go Gopher", Email: "me@example.com"})
	b.AssertRender(t, "text/field-validator-valid", func() {
		v.WidgetWalkDown(func(kwi Widget, kwb *WidgetBase) bool {
			kwb.SendChange() // trigger validation
			return tree.Continue
		})
	})
}

func TestFieldValidatorInvalid(t *testing.T) {
	b := NewBody()
	v := NewForm(b).SetStruct(&fieldValidator{Name: "Go Gopher", Email: "me@example"})
	b.AssertRender(t, "text/field-validator-invalid", func() {
		v.WidgetWalkDown(func(kwi Widget, kwb *WidgetBase) bool {
			kwb.SendChange() // trigger validation
			return tree.Continue
		})
	})
}
*/

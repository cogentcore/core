// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coredom

import (
	"strings"
	"testing"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
)

func TestInlineContainer(t *testing.T) {
	b := gi.NewBody()
	grr.Test(t, ReadHTMLString(NewContext(), b, `<button>Test</button>`))
	if tag := b.Child(0).Prop("tag"); tag != "body" {
		t.Errorf("expected first child to be body but got %v", tag)
	}
	if !strings.Contains(b.Child(0).Child(0).Name(), "inline") {
		t.Errorf("expected inline container for h1 but got %v", b.Child(0))
	}
}

func TestNoInlineContainer(t *testing.T) {
	b := gi.NewBody()
	grr.Test(t, ReadHTMLString(NewContext(), b, `<h1>Test</h1>`))
	if tag := b.Child(0).Prop("tag"); tag != "body" {
		t.Errorf("expected first child to be body but got %v", tag)
	}
	if strings.Contains(b.Child(0).Child(0).Name(), "inline") {
		t.Errorf("expected no inline container for h1 but got %v", b.Child(0))
	}
}

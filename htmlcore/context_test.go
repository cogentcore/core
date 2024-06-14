// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
)

func TestInlineContainer(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadHTMLString(NewContext(), b, `<button>Test</button>`))
	if tag := b.Child(0).AsTree().Property("tag"); tag != "body" {
		t.Errorf("expected first child to be body but got %v", tag)
	}
	if !strings.Contains(b.Child(0).AsTree().Child(0).AsTree().Name, "inline") {
		t.Errorf("expected inline container for h1 but got %v", b.Child(0))
	}
}

func TestNoInlineContainer(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadHTMLString(NewContext(), b, `<h1>Test</h1>`))
	if tag := b.Child(0).AsTree().Property("tag"); tag != "body" {
		t.Errorf("expected first child to be body but got %v", tag)
	}
	if strings.Contains(b.Child(0).AsTree().Child(0).AsTree().Name, "inline") {
		t.Errorf("expected no inline container for h1 but got %v", b.Child(0))
	}
}

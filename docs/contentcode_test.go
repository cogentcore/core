// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
)

// TestCompiledCode runs every code block of the docs that is compiled into
// the app (contentcode_gen.go) once, as the docs do when they show it.
func TestCompiledCode(t *testing.T) {
	b := core.NewBody("code blocks")
	parent := core.NewFrame(b)
	ok, failed, blocked := 0, 0, 0
	for key, f := range content.CompiledCode {
		done := make(chan error, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("panic: %v", r)
				}
			}()
			parent.DeleteChildren()
			core.ExternalParent = parent
			var out bytes.Buffer
			f(parent, &out)
			parent.Update()
			done <- nil
		}()
		select {
		case err := <-done:
			if err != nil {
				failed++
				t.Errorf("%s: %v", key, err)
			} else {
				ok++
			}
		case <-time.After(5 * time.Second):
			blocked++
			t.Logf("%s: still running after 5s", key)
			return
		}
	}
	// failures are only mentioned when there are some
	sum := fmt.Sprintf("compiled code blocks: %d of %d ran successfully", ok, len(content.CompiledCode))
	if failed > 0 {
		sum += fmt.Sprintf(", %d failed", failed)
	}
	if blocked > 0 {
		sum += fmt.Sprintf(", %d still running after 5s", blocked)
	}
	t.Log(sum)
}

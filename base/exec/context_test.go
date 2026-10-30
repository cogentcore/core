// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRunIOContext tests that a command is interrupted when its context
// is done, and that one that ignores the interrupt is killed after
// [InterruptWait].
func TestRunIOContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	defer func(w time.Duration) { InterruptWait = w }(InterruptWait)
	InterruptWait = 500 * time.Millisecond
	c := &Config{}
	run := func(script string) (time.Duration, error) {
		ctx, cancel := context.WithCancel(context.Background())
		time.AfterFunc(200*time.Millisecond, cancel)
		start := time.Now()
		err := c.RunIOContext(ctx, NewCmdIO(c), "sh", "-c", script)
		return time.Since(start), err
	}
	took, err := run("sleep 10")
	assert.Error(t, err)
	assert.Less(t, took, 500*time.Millisecond, "interrupted")

	took, err = run(`trap "" INT; sleep 10`)
	assert.Error(t, err)
	assert.Greater(t, took, 600*time.Millisecond, "ignores the interrupt")
	assert.Less(t, took, 2*time.Second, "killed after InterruptWait")

	// without a context
	assert.NoError(t, c.RunIOContext(nil, NewCmdIO(c), "true"))
}

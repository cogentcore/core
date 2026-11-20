// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"testing"
	"time"

	"cogentcore.org/core/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotifier tests showing notifications now and at their times, with
// the ones scheduled before replaced by each sync, and clicks.
func TestNotifier(t *testing.T) {
	nt := newNotifier()
	var nr system.Notifier = nt
	allowed := make(chan bool, 1)
	nr.RequestPermission(func(ok bool) { allowed <- ok })
	assert.True(t, <-allowed)

	require.NoError(t, nr.Show(&system.Notification{ID: "now", Title: "Now"}))
	assert.Equal(t, []string{"now"}, ids(nt.Shown()))

	nt.ClearShown()
	now := time.Now()
	at := func(d time.Duration, id string) *system.Notification {
		return &system.Notification{ID: id, Title: id, Time: now.Add(d)}
	}
	require.NoError(t, nr.Sync([]*system.Notification{at(time.Hour, "replaced")}))
	require.NoError(t, nr.Sync([]*system.Notification{
		at(60*time.Millisecond, "b"), at(-time.Second, "past"), at(30*time.Millisecond, "a"),
	}))
	assert.Eventually(t, func() bool { return len(nt.Shown()) == 2 }, 5*time.Second, 5*time.Millisecond)
	assert.Equal(t, []string{"a", "b"}, ids(nt.Shown()), "in order, without the past and replaced ones")

	nt.ClearShown()
	require.NoError(t, nr.Sync([]*system.Notification{at(30*time.Millisecond, "canceled")}))
	require.NoError(t, nr.Sync(nil))
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, nt.Shown(), "canceled")

	clicked := make(chan string, 1)
	nr.OnClick(func(id string) { clicked <- id })
	nt.Clicked("a")
	assert.Equal(t, "a", <-clicked)
}

// TestScheduleOrder tests the notifications that are scheduled.
func TestScheduleOrder(t *testing.T) {
	now := time.Now()
	var ns []*system.Notification
	for i := range system.MaxScheduled + 10 {
		ns = append(ns, &system.Notification{ID: string(rune('A' + i)), Time: now.Add(time.Duration(100-i) * time.Minute)})
	}
	ns = append(ns, nil, &system.Notification{ID: "past", Time: now.Add(-time.Minute)}, &system.Notification{ID: "zero"})
	sched := system.ScheduleOrder(ns, now)
	require.Len(t, sched, system.MaxScheduled)
	assert.Equal(t, ns[system.MaxScheduled+9].ID, sched[0].ID, "the soonest")
	for i := 1; i < len(sched); i++ {
		assert.True(t, sched[i-1].Time.Before(sched[i].Time))
	}
}

// ids returns the IDs of the given notifications.
func ids(ns []system.Notification) []string {
	var s []string
	for _, n := range ns {
		s = append(s, n.ID)
	}
	return s
}

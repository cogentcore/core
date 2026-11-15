// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/text/parse/complete"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLanguages is the list of words for the test completer.
var testLanguages = []string{"Go", "Golang", "Gopher", "Python", "Rust"}

// testMatch returns the test languages that start with the last word
// of the given text.
func testMatch(data any, text string, posLine, posChar int) complete.Matches {
	seed := text[strings.LastIndex(text, " ")+1:]
	var md complete.Matches
	md.Seed = seed
	for _, l := range data.([]string) {
		if strings.HasPrefix(l, seed) {
			md.Matches = append(md.Matches, complete.Completion{Text: l, Desc: "the " + l + " language"})
		}
	}
	return md
}

// testEdit replaces the seed word with the completion.
func testEdit(data any, text string, cursorPos int, comp complete.Completion, seed string) complete.Edit {
	return complete.EditWord(text, cursorPos, comp.Text, seed)
}

// TestTextFieldComplete tests showing the completions of a text field
// in a popup, and choosing one, which replaces the seed.
func TestTextFieldComplete(t *testing.T) {
	wait := TimingSettings.CompleteWaitDuration
	TimingSettings.CompleteWaitDuration = 0 // shows at once
	defer func() { TimingSettings.CompleteWaitDuration = wait }()

	b, tf := newEditTextField("")
	tf.SetCompleter(testLanguages, testMatch, testEdit)
	AssertRender(t, b, "text-field/complete", Locked(b, func() {
		tf.SetFocus()
		TypeText(tf, "I like Go")
		tf.offerComplete()
	}), Locked(b, func() {
		c := tf.complete
		require.NotNil(t, c.stage, "the popup is shown")
		assert.Equal(t, "Go", c.Seed)
		assert.Len(t, c.completions, 3)
		assert.Equal(t, "the Gopher language", c.GetCompletion("Gopher").Desc)
		assert.Equal(t, "", c.GetCompletion("Rust").Text, "not a completion")
	}), Locked(b, func() {
		gopher := tf.complete.stage.Scene.Child(2).(*Button)
		assert.Equal(t, "Gopher", gopher.Text)
		gopher.Send(events.Click)
		assert.Equal(t, "I like Gopher", string(tf.editText))
		assert.Equal(t, len("I like Gopher"), tf.CursorPos())
		assert.Nil(t, tf.complete.stage, "the popup is closed")
	}))
}

// TestCompleteDelay tests that completions are shown after a delay,
// which is cancelled by cancelling the completer.
func TestCompleteDelay(t *testing.T) {
	wait := TimingSettings.CompleteWaitDuration
	TimingSettings.CompleteWaitDuration = 20 * time.Millisecond
	defer func() { TimingSettings.CompleteWaitDuration = wait }()

	b, tf := newEditTextField("")
	tf.SetCompleter(testLanguages, testMatch, testEdit)
	shown := make(chan bool, 1)
	AssertRender(t, b, "text-field/complete-delay", Locked(b, func() {
		tf.SetFocus()
		TypeText(tf, "Py")
		tf.offerComplete()
		assert.True(t, tf.complete.IsAboutToShow())
	}), func() {
		for range 200 {
			b.AsyncLock()
			st := tf.complete.stage
			b.AsyncUnlock()
			if st != nil {
				shown <- true
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		shown <- false
	}, Locked(b, func() {
		assert.True(t, <-shown, "shown after the delay")
		assert.False(t, tf.complete.IsAboutToShow())
		assert.Len(t, tf.complete.completions, 1)
		assert.True(t, tf.complete.Cancel())
		assert.False(t, tf.complete.Cancel(), "already cancelled")
	}))
}

// TestCompleteLookup tests that a lookup calls the lookup function, and
// that the default edit of a completion is the completion.
func TestCompleteLookup(t *testing.T) {
	c := NewComplete()
	looked := ""
	c.Lookup("x", 0, 1, NewScene(), image.Point{})
	c.SetLookupFunc(func(data any, text string, posLine, posChar int) complete.Lookup {
		looked = text
		return complete.Lookup{}
	})
	c.Lookup("fmt.Println", 2, 5, nil, image.Point{})
	assert.Equal(t, "", looked, "no lookup without a scene")
	c.Lookup("fmt.Println", 2, 5, NewScene(), image.Point{})
	assert.Equal(t, "fmt.Println", looked)

	ed := CompleteEditText("fmt.Pr", 6, "Println", "Pr")
	assert.Equal(t, "Println", ed.NewText)
}

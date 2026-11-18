// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUserSpell returns a built-in spell checker with a user dictionary
// file in a temporary directory, which has the word "zorbly".
func newUserSpell(t *testing.T) (*SpellData, string) {
	t.Helper()
	UseNative = false
	t.Cleanup(func() { UseNative = true })
	fn := filepath.Join(t.TempDir(), "user.txt")
	require.NoError(t, os.WriteFile(fn, []byte("zorbly\n"), 0o644))
	sp := NewSpell(fn)
	require.NotNil(t, sp)
	require.False(t, sp.IsNative())
	return sp, fn
}

// TestUserDict tests learning and deleting words, which are saved in the
// user dictionary file, and opening it again when it changes.
func TestUserDict(t *testing.T) {
	sp, fn := newUserSpell(t)
	_, known := sp.CheckWord("zorbly")
	assert.True(t, known, "a word of the user dictionary")
	assert.Contains(t, sp.model.UserDict, "zorbly")

	sp.AddWord("Florbit")
	_, known = sp.CheckWord("florbit")
	assert.True(t, known, "learned")
	d, err := OpenDict(fn)
	require.NoError(t, err)
	assert.Contains(t, d, "florbit", "saved")

	sp.DeleteWord("florbit")
	_, known = sp.CheckWord("florbit")
	assert.False(t, known, "deleted")
	d, err = OpenDict(fn)
	require.NoError(t, err)
	assert.NotContains(t, d, "florbit", "the deleted word is not saved")
	assert.Contains(t, d, "zorbly")

	// another program (or instance) changes the file, which is opened
	// again (this deadlocked before)
	require.NoError(t, os.WriteFile(fn, []byte("zorbly\nquimble\n"), 0o644))
	done := make(chan error, 1)
	go func() { done <- sp.OpenUserCheck() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("OpenUserCheck deadlocked")
	}
	assert.Contains(t, sp.model.UserDict, "quimble")
	_, known = sp.CheckWord("quimble")
	assert.True(t, known, "learned by the other program")
	require.NoError(t, sp.SaveUserIfLearn(), "nothing learned: checks the file")

	var nilsp *SpellData
	assert.NoError(t, nilsp.OpenUserCheck())
	assert.NoError(t, nilsp.SaveUserIfLearn())
	assert.False(t, HasNative() && !UseNative)
}

// TestOpenUserMissing tests a user dictionary file that does not exist.
func TestOpenUserMissing(t *testing.T) {
	UseNative = false
	defer func() { UseNative = true }()
	sp := NewSpell(filepath.Join(t.TempDir(), "none.txt"))
	require.NotNil(t, sp)
	assert.Empty(t, sp.model.UserDict)
	assert.NoError(t, sp.OpenUserCheck(), "no words learned yet")
}

// TestUserDictShared tests two spell checkers (as of two apps) with the
// same user dictionary file, which each keep the words of the other when
// they save it, and know them when they open it again.
func TestUserDictShared(t *testing.T) {
	a, fn := newUserSpell(t)
	b := NewSpell(fn)
	a.AddWord("glorp")
	b.AddWord("snirkle")
	b.DeleteWord("zorbly")
	d, err := OpenDict(fn)
	require.NoError(t, err)
	assert.Equal(t, []string{"glorp", "snirkle"}, d.List(), "the words of both")
	_, known := b.CheckWord("glorp")
	assert.True(t, known, "read when saving")

	require.NoError(t, a.OpenUserCheck())
	for w, want := range map[string]bool{"glorp": true, "snirkle": true, "zorbly": false} {
		_, known := a.CheckWord(w)
		assert.Equal(t, want, known, w)
	}
	a.AddWord("the") // a word of the dictionary is not learned
	d, err = OpenDict(fn)
	require.NoError(t, err)
	assert.NotContains(t, d, "the")
}

// TestModelEdits tests the Levenshtein distance and the suggestions of
// the model for a new word.
func TestModelEdits(t *testing.T) {
	a, b := "kitten", "sitting"
	assert.Equal(t, 3, Levenshtein(&a, &b))
	e := ""
	assert.Equal(t, 6, Levenshtein(&a, &e))
	assert.Equal(t, 0, Levenshtein(&a, &a))

	md := NewModel()
	md.SetDicts(NewDictFromList([]string{"apple", "banana"}), Dict{})
	md.AddWord("cherry")
	md.AddWord("cherry") // already known
	assert.Contains(t, md.Suggestions("chery", 5), "cherry")
	md.Delete("cherry")
	assert.NotContains(t, md.Suggestions("chery", 5), "cherry")
	assert.NotContains(t, md.UserDict, "cherry")
}

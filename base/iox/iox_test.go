// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iox

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	jsonDecoder = NewDecoderFunc(json.NewDecoder)
	jsonEncoder = NewEncoderFunc(json.NewEncoder)
)

// failEncoder is an [Encoder] that always fails.
type failEncoder struct{}

func (failEncoder) Encode(v any) error { return errors.New("fail") }

// TestEncodeDecode tests the generic functions with a JSON encoder and
// decoder, including their errors.
func TestEncodeDecode(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "a.json")
	v := map[string]int{"a": 1}
	require.NoError(t, Save(v, fn, jsonEncoder))
	b, err := WriteBytes(v, jsonEncoder)
	require.NoError(t, err)
	assert.Equal(t, "{\"a\":1}\n", string(b))
	var buf bytes.Buffer
	require.NoError(t, Write(v, &buf, jsonEncoder))
	assert.Equal(t, b, buf.Bytes())

	r := map[string]int{}
	require.NoError(t, Open(&r, fn, jsonDecoder))
	assert.Equal(t, v, r)
	require.NoError(t, ReadBytes(&r, []byte(`{"b":2}`), jsonDecoder))
	assert.Equal(t, 2, r["b"])
	fn2 := filepath.Join(dir, "b.json")
	require.NoError(t, os.WriteFile(fn2, []byte(`{"c":3}`), 0666))
	r = map[string]int{}
	require.NoError(t, OpenFiles(&r, []string{fn, fn2}, jsonDecoder))
	assert.Equal(t, map[string]int{"a": 1, "c": 3}, r, "merged")
	assert.Error(t, OpenFiles(&r, []string{fn, "missing.json"}, jsonDecoder))
	assert.Error(t, Open(&r, filepath.Join(dir, "missing.json"), jsonDecoder))

	fsys := fstest.MapFS{"a.json": {Data: b}, "b.json": {Data: []byte(`{"d":4}`)}}
	r = map[string]int{}
	require.NoError(t, OpenFS(&r, fsys, "a.json", jsonDecoder))
	require.NoError(t, OpenFilesFS(&r, fsys, []string{"a.json", "b.json"}, jsonDecoder))
	assert.Equal(t, map[string]int{"a": 1, "d": 4}, r)
	assert.Error(t, OpenFilesFS(&r, fsys, []string{"missing.json"}, jsonDecoder))
	assert.Error(t, OpenFS(&r, fsys, "missing.json", jsonDecoder))

	fail := EncoderFunc(func(w io.Writer) Encoder { return failEncoder{} })
	assert.Error(t, Save(v, filepath.Join(dir, "f.json"), fail))
	_, err = WriteBytes(v, fail)
	assert.Error(t, err)
	assert.Error(t, Save(v, filepath.Join(dir, "no-dir", "a.json"), jsonEncoder))
	assert.Error(t, Read(&r, bytes.NewReader([]byte("{")), jsonDecoder))
}

// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatsRoundTrip tests writing and reading images in all of the
// formats that can be written, and the formats of extensions.
func TestFormatsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	img := testImage()
	for _, f := range []Formats{PNG, JPEG, GIF, TIFF, BMP} {
		var buf bytes.Buffer
		require.NoError(t, Write(img, &buf, f), f.String())
		rimg, rf, err := Read(bytes.NewReader(buf.Bytes()))
		require.NoError(t, err, f.String())
		assert.Equal(t, f, rf, f.String())
		assert.Equal(t, img.Bounds(), rimg.Bounds(), f.String())

		ext := map[Formats]string{PNG: ".png", JPEG: ".jpg", GIF: ".gif", TIFF: ".tif", BMP: ".bmp"}[f]
		fn := filepath.Join(dir, "a"+ext)
		require.NoError(t, Save(img, fn), f.String())
		_, of, err := Open(fn)
		require.NoError(t, err, f.String())
		assert.Equal(t, f, of)
	}
	assert.Error(t, Write(img, &bytes.Buffer{}, WebP), "webp is only read")
	assert.Error(t, Save(img, filepath.Join(dir, "a.xyz")))
	assert.Error(t, Save(img, filepath.Join(dir, "no-dir", "a.png")))
	_, _, err := Open(filepath.Join(dir, "missing.png"))
	assert.Error(t, err)
	_, _, err = Read(bytes.NewReader([]byte("not an image")))
	assert.Error(t, err)

	pb, err := os.ReadFile(filepath.Join(dir, "a.png"))
	require.NoError(t, err)
	fsys := fstest.MapFS{"a.png": {Data: pb}}
	fimg, ff, err := OpenFS(fsys, "a.png")
	require.NoError(t, err)
	assert.Equal(t, PNG, ff)
	_, content, _, _, _, _ := ImagesEqual(img, fimg, 0)
	assert.True(t, content, "png is lossless")
	_, _, err = OpenFS(fsys, "missing.png")
	assert.Error(t, err)

	for ext, f := range map[string]Formats{".PNG": PNG, "jpeg": JPEG, "tiff": TIFF, ".webp": WebP} {
		got, err := ExtToFormat(ext)
		assert.NoError(t, err, ext)
		assert.Equal(t, f, got, ext)
	}
	_, err = ExtToFormat("")
	assert.Error(t, err)
	_, err = ExtToFormat(".doc")
	assert.Error(t, err)
}

// TestBase64 tests encoding and decoding images as base64, including
// split into lines, and lines joined with spaces.
func TestBase64Lines(t *testing.T) {
	img := testImage()
	eb, mime := ToBase64(img, PNG)
	assert.Equal(t, "image/png", mime)
	lines := Base64SplitLines(eb)
	for _, ln := range bytes.Split(bytes.TrimSpace(lines), []byte("\n")) {
		assert.LessOrEqual(t, len(ln), 76)
	}
	got, f, err := FromBase64(lines)
	require.NoError(t, err)
	assert.Equal(t, PNG, f)
	assert.Equal(t, img.Bounds(), got.Bounds())

	spaced := bytes.ReplaceAll(bytes.TrimSpace(lines), []byte("\n"), []byte(" "))
	got, _, err = FromBase64(spaced)
	require.NoError(t, err, "lines joined with spaces")
	assert.Equal(t, img.Bounds(), got.Bounds())

	_, _, err = FromBase64([]byte("!!!not base64"))
	assert.Error(t, err)
	assert.Equal(t, []byte("\n"), Base64SplitLines(nil))
}

// TestJSONImage tests encoding images in JSON, and nil images.
func TestJSONImage(t *testing.T) {
	img := testImage()
	b, err := NewJSON(img).MarshalJSON()
	require.NoError(t, err)
	var js JSON
	require.NoError(t, js.UnmarshalJSON(b))
	require.NotNil(t, js.Image)
	assert.Equal(t, img.Bounds(), js.Image.Bounds())

	b, err = NewJSON(nil).MarshalJSON()
	require.NoError(t, err)
	require.NoError(t, js.UnmarshalJSON(b))
	assert.Nil(t, js.Image)
	assert.Error(t, js.UnmarshalJSON([]byte("{")))
	assert.Error(t, js.UnmarshalJSON([]byte(`{"Width":2,"Height":2,"Image":"AAAA"}`)), "not an image")
	assert.Nil(t, js.Image)
}

// wrapped is a [Wrapped] image for the tests.
type wrapped struct {
	*image.RGBA
	updates int
}

func (w *wrapped) Update()                 { w.updates++ }
func (w *wrapped) Underlying() image.Image { return w.RGBA }

// TestWrapped tests updating and unwrapping wrapped images.
func TestWrapped(t *testing.T) {
	img := testImage()
	w := &wrapped{RGBA: img}
	Update(w)
	Update(img)
	assert.Equal(t, 1, w.updates)
	assert.Same(t, img, Unwrap(w))
	assert.Same(t, img, Unwrap(img))
	assert.Same(t, img, AsRGBA(w), "a shallow RGBA is the same")
	cl := CloneAsRGBA(w)
	assert.NotSame(t, img, cl)
	assert.Equal(t, img.Pix, cl.Pix)

	var buf bytes.Buffer
	require.NoError(t, Write(w, &buf, PNG), "the underlying image is written")
}

// recordT is a [TestingT] that records its errors.
type recordT struct {
	errs []string
}

func (r *recordT) Errorf(format string, args ...any) {
	r.errs = append(r.errs, format)
}

// TestCompare tests comparing images and colors, the difference image, and
// Assert, which saves new and failed images, and in update mode keeps
// images with the same pixels.
func TestCompare(t *testing.T) {
	assert.True(t, Uint8Equal(10, 12, 2))
	assert.False(t, Uint8Equal(10, 13, 2))
	assert.False(t, Uint8Equal(13, 10, 2))
	c := color.RGBA{10, 20, 30, 255}
	assert.True(t, ColorsEqual(c, color.RGBA{11, 20, 30, 255}, 1))
	for _, d := range []color.RGBA{{20, 20, 30, 255}, {10, 30, 30, 255}, {10, 20, 40, 255}, {10, 20, 30, 200}} {
		assert.False(t, ColorsEqual(c, d, 1), "%v", d)
	}

	a := testImage()
	b := CloneAsRGBA(a)
	b.Set(1, 1, color.RGBA{255, 255, 255, 255})
	bounds, content, _, _, x, y := ImagesEqual(a, b, 0)
	assert.True(t, bounds)
	assert.False(t, content)
	assert.Equal(t, image.Pt(1, 1), image.Pt(x, y))
	bounds, _, _, _, _, _ = ImagesEqual(a, image.NewRGBA(image.Rect(0, 0, 1, 1)), 0)
	assert.False(t, bounds)
	di := DiffImage(a, b)
	assert.Equal(t, color.RGBA{0, 0, 0, 255}, di.At(0, 0), "no difference")
	assert.NotEqual(t, color.RGBA{0, 0, 0, 255}, di.At(1, 1))

	t.Chdir(t.TempDir())
	rt := &recordT{}
	Assert(rt, a, "cmp")
	assert.Empty(t, rt.errs, "a new image is saved")
	_, err := os.Stat(filepath.Join("testdata", "cmp.png"))
	require.NoError(t, err)
	Assert(rt, a, "cmp")
	assert.Empty(t, rt.errs, "the same image")
	Assert(rt, b, "cmp", 0)
	assert.Len(t, rt.errs, 1, "a different image")
	_, err = os.Stat(filepath.Join("testdata", "cmp.fail.png"))
	assert.NoError(t, err, "the failed image is saved")
	_, err = os.Stat(filepath.Join("testdata", "cmp.diff.png"))
	assert.NoError(t, err, "and the difference")
	Assert(rt, image.NewRGBA(image.Rect(0, 0, 1, 1)), "cmp")
	assert.Len(t, rt.errs, 2, "different bounds")

	t.Run("update", func(t *testing.T) {
		UpdateTestImagesFor(t)
		assert.True(t, UpdateTestImages)
		rt := &recordT{}
		Assert(rt, b, "cmp")
		assert.Empty(t, rt.errs)
		_, err := os.Stat(filepath.Join("testdata", "cmp.fail.png"))
		assert.True(t, errors.Is(err, os.ErrNotExist), "the fail image is removed on update")
		got, _, err := Open(filepath.Join("testdata", "cmp.png"))
		require.NoError(t, err)
		_, content, _, _, _, _ := ImagesEqual(b, got, 0)
		assert.True(t, content, "updated")
	})
	assert.Equal(t, updateTestImages, UpdateTestImages, "restored")
}

// TestBase64SplitLinesExact tests splitting base64 text whose length is a
// multiple of the line length, which has no empty last line.
func TestBase64SplitLinesExact(t *testing.T) {
	b := bytes.Repeat([]byte("a"), 152)
	lines := strings.Split(string(Base64SplitLines(b)), "\n")
	assert.Equal(t, []string{strings.Repeat("a", 76), strings.Repeat("a", 76), ""}, lines)
	short := strings.Split(string(Base64SplitLines([]byte("abc"))), "\n")
	assert.Equal(t, []string{"abc", ""}, short)
}

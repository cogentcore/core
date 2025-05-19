// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagex

//go:generate core generate

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Formats are the supported image encoding / decoding formats
type Formats int32 //enums:enum

// The supported image encoding formats
const (
	None Formats = iota
	PNG
	JPEG
	GIF
	TIFF
	BMP
	WebP
)

// ExtToFormat returns a Format based on a filename extension,
// which can start with a . or not
func ExtToFormat(ext string) (Formats, error) {
	if len(ext) == 0 {
		return None, errors.New("ExtToFormat: ext is empty")
	}
	if ext[0] == '.' {
		ext = ext[1:]
	}
	ext = strings.ToLower(ext)
	switch ext {
	case "png":
		return PNG, nil
	case "jpg", "jpeg":
		return JPEG, nil
	case "gif":
		return GIF, nil
	case "tif", "tiff":
		return TIFF, nil
	case "bmp":
		return BMP, nil
	case "webp":
		return WebP, nil
	}
	return None, fmt.Errorf("ExtToFormat: extension %q not recognized", ext)
}

// Open opens an image from the given filename.
// The format is inferred automatically,
// and is returned using the Formats enum.
// png, jpeg, gif, tiff, bmp, and webp are supported.
func Open(filename string) (image.Image, Formats, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, None, err
	}
	defer file.Close()
	return Read(file)
}

// OpenFS opens an image from the given filename
// using the given [fs.FS] filesystem (e.g., for embed files).
// The format is inferred automatically,
// and is returned using the Formats enum.
// png, jpeg, gif, tiff, bmp, and webp are supported.
func OpenFS(fsys fs.FS, filename string) (image.Image, Formats, error) {
	file, err := fsys.Open(filename)
	if err != nil {
		return nil, None, err
	}
	defer file.Close()
	return Read(file)
}

// Read reads an image to the given reader,
// The format is inferred automatically,
// and is returned using the Formats enum.
// png, jpeg, gif, tiff, bmp, and webp are supported.
func Read(r io.Reader) (image.Image, Formats, error) {
	im, ext, err := image.Decode(r)
	if err != nil {
		return im, None, err
	}
	f, err := ExtToFormat(ext)
	return im, f, err
}

// Save saves the image to the given filename,
// with the format inferred from the filename.
// png, jpeg, gif, tiff, and bmp are supported.
func Save(im image.Image, filename string) error {
	ext := filepath.Ext(filename)
	f, err := ExtToFormat(ext)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	bw := bufio.NewWriter(file)
	defer bw.Flush()
	return Write(im, file, f)
}

// Write writes the image to the given writer using the given foramt.
// png, jpeg, gif, tiff, and bmp are supported.
// It [Unwrap]s any [Wrapped] images.
func Write(im image.Image, w io.Writer, f Formats) error {
	im = Unwrap(im)
	switch f {
	case PNG:
		return png.Encode(w, im)
	case JPEG:
		return jpeg.Encode(w, im, &jpeg.Options{Quality: 90})
	case GIF:
		return gif.Encode(w, im, nil)
	case TIFF:
		return tiff.Encode(w, im, nil)
	case BMP:
		return bmp.Encode(w, im)
	default:
		return fmt.Errorf("iox/imagex.Save: format %q not valid", f)
	}
}

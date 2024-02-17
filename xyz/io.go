// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"cogentcore.org/core/glop/dirs"
	"cogentcore.org/core/ki"
)

// Decoder parses 3D object / scene file(s) and imports into a Group or Scene.
// This interface is implemented by the different format-specific decoders.
type Decoder interface {
	// New returns a new instance of the decoder used for a specific decoding
	New() Decoder

	// Desc returns the description of this decoder
	Desc() string

	// SetFileFS sets the file name being used for decoding, or error if not found.
	// Returns a list of files that should be loaded along with the main one, if needed.
	// For example, .obj decoder adds a corresponding .mtl file.  In addition,
	// decoded files may specify further files (textures, etc) that must be located
	// relative to the same fsys directory.
	// All file operations use the fsys file system for access, and this should be a
	// Sub FS anchored at the directory where the filename is located.
	SetFileFS(fsys fs.FS, fname string) ([]string, error)

	// Decode reads the given data and decodes it, returning a new instance
	// of the Decoder that contains all the decoded info.
	// Some formats (e.g., Wavefront .obj) have separate .obj and .mtl files
	// which are passed as two reader args.
	Decode(rs []io.Reader) error

	// SetGroup sets the group to contain the decoded objects within the
	// given scene.
	SetGroup(sc *Scene, gp *Group)

	// HasScene returns true if this decoder has full scene information --
	// otherwise it only supports objects to be used in SetGroup.
	HasScene() bool

	// SetScene sets the scene according to the decoded data.
	SetScene(sc *Scene)
}

// Decoders is the master list of decoders, indexed by the primary extension.
// .obj = Wavefront object file -- only has mesh data, not scene info.
var Decoders = map[string]Decoder{}

// DecodeFile decodes the given file using a decoder based on the file
// extension.  Returns decoder instance with full decoded state.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
func DecodeFile(fname string) (Decoder, error) {
	dfs, fnm, err := dirs.DirFS(fname)
	if err != nil {
		return nil, err
	}
	return DecodeFileFS(dfs, fnm)
}

// DecodeFileFS decodes the given file from the given filesystem using a decoder based on the file
// extension.  Returns decoder instance with full decoded state.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
func DecodeFileFS(fsys fs.FS, fname string) (Decoder, error) {
	ext := filepath.Ext(fname)
	dt, has := Decoders[ext]
	if !has {
		return nil, fmt.Errorf("xyz.DecodeFile: file extension: %v not found in Decoders list for file %v", ext, fname)
	}
	dec := dt.New()
	fn := fname
	subdir, fn := filepath.Split(fn)
	var err error
	if subdir != "" {
		fsys, err = fs.Sub(fsys, subdir)
		if err != nil {
			return nil, fmt.Errorf("xyz.DecodeFile: file directory not found error: %v for file: %v", err, fname)
		}
	}
	files, err := dec.SetFileFS(fsys, fn)
	if err != nil {
		return nil, fmt.Errorf("xyz.DecodeFile: file not found error: %v for file: %v", err, fname)
	}
	nf := len(files)

	fs := make([]fs.File, nf)
	rs := make([]io.Reader, nf)
	defer func() {
		for _, fi := range fs {
			if fi != nil {
				fi.Close()
			}
		}
	}()

	for i, f := range files {
		fs[i], err = fsys.Open(f)
		if err != nil {
			return nil, err
		}
		rs[i] = fs[i]
	}
	err = dec.Decode(rs)
	if err != nil {
		return nil, err
	}
	return dec, nil
}

// OpenObj opens object(s) from given file into given group in scene,
// using a decoder based on the file extension.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
func (sc *Scene) OpenObj(fname string, gp *Group) error {
	dfs, fnm, err := dirs.DirFS(fname)
	if err != nil {
		return err
	}
	return sc.OpenObjFS(dfs, fnm, gp)
}

// OpenObjFS opens object(s) from given file in the given filesystem into given group in scene,
// using a decoder based on the file extension.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
func (sc *Scene) OpenObjFS(fsys fs.FS, fname string, gp *Group) error {
	dec, err := DecodeFileFS(fsys, fname)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetGroup(sc, gp)
	sc.Config() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

// OpenNewObj opens object(s) from given file into a new group
// under given parent, using a decoder based on the file extension.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
func (sc *Scene) OpenNewObj(fname string, parent ki.Ki) (*Group, error) {
	dfs, fnm, err := dirs.DirFS(fname)
	if err != nil {
		return nil, err
	}
	return sc.OpenNewObjFS(dfs, fnm, parent)
}

// OpenNewObjFS opens object(s) from given file in the given filesystem into a new group
// under given parent, using a decoder based on the file extension.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
func (sc *Scene) OpenNewObjFS(fsys fs.FS, fname string, parent ki.Ki) (*Group, error) {
	dec, err := DecodeFileFS(fsys, fname)
	if err != nil {
		return nil, err
	}
	updt := sc.UpdateStart()
	_, fn := filepath.Split(fname)
	gp := NewGroup(parent, fn)
	dec.SetGroup(sc, gp)
	if sc.IsConfiged() { // has already been configed
		sc.Config()
	}
	sc.UpdateEnd(updt)
	return gp, nil
}

// OpenToLibrary opens object(s) from given file into the scene's Library
// using a decoder based on the file extension.  The library key name
// must be unique, and is given by libnm -- if empty, then the filename (only)
// without extension is used.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
func (sc *Scene) OpenToLibrary(fname string, libnm string) (*Group, error) {
	dfs, fnm, err := dirs.DirFS(fname)
	if err != nil {
		return nil, err
	}
	return sc.OpenToLibraryFS(dfs, fnm, libnm)
}

// OpenToLibraryFS opens object(s) from given file in the given filesystem into the scene's Library
// using a decoder based on the file extension.  The library key name
// must be unique, and is given by libnm -- if empty, then the filename (only)
// without extension is used.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
func (sc *Scene) OpenToLibraryFS(fsys fs.FS, fname string, libnm string) (*Group, error) {
	dec, err := DecodeFileFS(fsys, fname)
	if err != nil {
		return nil, err
	}
	if libnm == "" {
		_, fn := filepath.Split(fname)
		ext := filepath.Ext(fn)
		libnm = strings.TrimSuffix(fn, ext)
	}
	gp := sc.NewInLibrary(libnm)
	dec.SetGroup(sc, gp)
	return gp, nil
}

// OpenScene opens a scene from the given file, using a decoder based on
// the file extension in first file name.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
//	Does not support full scene data so only objects are loaded
//	into a new group in scene.
func (sc *Scene) OpenScene(fname string) error {
	dfs, fnm, err := dirs.DirFS(fname)
	if err != nil {
		return err
	}
	return sc.OpenSceneFS(dfs, fnm)
}

// OpenSceneFS opens a scene from the given file in the given filesystem, using a decoder based on
// the file extension in first file name.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
//	Does not support full scene data so only objects are loaded
//	into a new group in scene.
func (sc *Scene) OpenSceneFS(fsys fs.FS, fname string) error {
	dec, err := DecodeFileFS(fsys, fname)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetScene(sc)
	sc.Config() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

// ReadObj reads object(s) from given reader(s) into given group in scene,
// using a decoder based on the extension of the given file name --
// even though the file name is not directly used to read the file, it is
// required for naming and decoding selection.  This method can be used
// for loading data embedded in an executable for example.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	is the 2nd reader arg, or a default material is used.
func (sc *Scene) ReadObj(fname string, rs []io.Reader, gp *Group) error {
	ext := filepath.Ext(fname)
	dt, has := Decoders[ext]
	if !has {
		return fmt.Errorf("xyz.ReadObj: file extension: %v not found in Decoders list", ext)
	}
	dfs, fnm, err := dirs.DirFS(fname)
	dec := dt.New()
	dec.SetFileFS(dfs, fnm)
	err = dec.Decode(rs)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetGroup(sc, gp)
	sc.Config() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

// ReadScene reads scene from given reader(s), using a decoder based on the
// file name extension -- even though the file name is not directly used
// to read the file, it is required for naming and decoding selection.
// This method can be used for loading data embedded in an executable for example.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//
//	must have same name as .obj, or a default material is used.
//	Does not support full scene data so only objects are loaded
//	into a new group in scene.
func (sc *Scene) ReadScene(fname string, rs []io.Reader, gp *Group) error {
	ext := filepath.Ext(fname)
	dt, has := Decoders[ext]
	if !has {
		return fmt.Errorf("xyz.ReadScene: file extension: %v not found in Decoders list", ext)
	}
	dfs, fnm, err := dirs.DirFS(fname)
	dec := dt.New()
	dec.SetFileFS(dfs, fnm)
	err = dec.Decode(rs)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetScene(sc)
	sc.Config() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

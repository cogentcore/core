// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/ki/ki"
)

// Decoder parses 3D object / scene file(s) and imports into a Group or Scene.
// This interface is implemented by the different format-specific decoders.
type Decoder interface {
	// New returns a new instance of the decoder used for a specific decoding
	New() Decoder

	// Desc returns the description of this decoder
	Desc() string

	// SetFile sets the file name being used for decoding -- needed in case
	// of loading other files such as textures / materials from the same directory.
	// Returns a list of files that should be loaded along with the main one, if needed.
	// For example, .obj decoder adds a corresponding .mtl file.
	SetFile(fname string) []string

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
//        must have same name as .obj, or a default material is used.
func DecodeFile(fname string) (Decoder, error) {
	ext := filepath.Ext(fname)
	dt, has := Decoders[ext]
	if !has {
		return nil, fmt.Errorf("gi3d.DecodeFile: file extension: %v not found in Decoders list for file %v", ext, fname)
	}
	dec := dt.New()
	files := dec.SetFile(fname)
	nf := len(files)

	var err error
	fs := make([]*os.File, nf)
	rs := make([]io.Reader, nf)
	defer func() {
		for _, fi := range fs {
			if fi != nil {
				fi.Close()
			}
		}
	}()

	for i, f := range files {
		fs[i], err = os.Open(f)
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
//        must have same name as .obj, or a default material is used.
func (sc *Scene) OpenObj(fname string, gp *Group) error {
	dec, err := DecodeFile(fname)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetGroup(sc, gp)
	sc.Init3D() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

// OpenNewObj opens object(s) from given file into a new group
// under given parent, using a decoder based on the file extension.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//        must have same name as .obj, or a default material is used.
func (sc *Scene) OpenNewObj(fname string, parent ki.Ki) (*Group, error) {
	dec, err := DecodeFile(fname)
	if err != nil {
		return nil, err
	}
	updt := sc.UpdateStart()
	_, fn := filepath.Split(fname)
	gp := AddNewGroup(sc, parent, fn)
	dec.SetGroup(sc, gp)
	sc.Init3D() // needed after loading
	sc.UpdateEnd(updt)
	return gp, nil
}

// OpenToLibrary opens object(s) from given file into the scene's Library
// using a decoder based on the file extension.  The library key name
// must be unique, and is given by libnm -- if empty, then the filename (only)
// without extension is used.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//        must have same name as .obj, or a default material is used.
func (sc *Scene) OpenToLibrary(fname string, libnm string) (*Group, error) {
	dec, err := DecodeFile(fname)
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

// OpenScene opens a scene from given file, using a decoder based on
// the file extension in first file name.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//        must have same name as .obj, or a default material is used.
//        Does not support full scene data so only objects are loaded
//        into a new group in scene.
func (sc *Scene) OpenScene(fname string) error {
	dec, err := DecodeFile(fname)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetScene(sc)
	sc.Init3D() // needed after loading
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
//        is the 2nd reader arg, or a default material is used.
func (sc *Scene) ReadObj(fname string, rs []io.Reader, gp *Group) error {
	ext := filepath.Ext(fname)
	dt, has := Decoders[ext]
	if !has {
		return fmt.Errorf("gi3d.ReadObj: file extension: %v not found in Decoders list", ext)
	}
	dec := dt.New()
	dec.SetFile(fname)
	err := dec.Decode(rs)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetGroup(sc, gp)
	sc.Init3D() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

// ReadScene reads scene from given reader(s), using a decoder based on the
// file name extension -- even though the file name is not directly used
// to read the file, it is required for naming and decoding selection.
// This method can be used for loading data embedded in an executable for example.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which
//        must have same name as .obj, or a default material is used.
//        Does not support full scene data so only objects are loaded
//        into a new group in scene.
func (sc *Scene) ReadScene(fname string, rs []io.Reader, gp *Group) error {
	ext := filepath.Ext(fname)
	dt, has := Decoders[ext]
	if !has {
		return fmt.Errorf("gi3d.ReadScene: file extension: %v not found in Decoders list", ext)
	}
	dec := dt.New()
	dec.SetFile(fname)
	err := dec.Decode(rs)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetScene(sc)
	sc.Init3D() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

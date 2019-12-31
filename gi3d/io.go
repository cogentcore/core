// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goki/ki/ki"
)

// Decoder parses 3D object / scene file(s) and imports into a Group or Scene.
// This interface is implemented by the different format-specific decoders.
type Decoder interface {
	// New returns a new instance of the decoder used for a specific decoding
	New() Decoder

	// Desc returns the description of this decoder
	Desc() string

	// SetFiles sets the file names being used for decoding -- needed in case
	// of loading other files such as textures / materials from the same directory.
	// Also potentially modifies the list of files to suggest other files
	// that should be loaded along with those passed.
	// For example, .obj decoder adds a corresponding .mtl file if one is not
	// otherwise passed.
	SetFiles(files []string) []string

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

// DecodeFile decodes the given file(s) using the decoder based on the file
// extension in first file name.  Returns decoder instance with full decoded state.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which can
//        be specified as second file name -- otherwise auto-searched based on
//        .obj filename, or a default material is used.
func DecodeFile(files []string) (Decoder, error) {
	nf := len(files)
	if nf == 0 {
		return nil, errors.New("gi3d.DecodeFile: no files passed")
	}
	fn := files[0]
	ext := filepath.Ext(fn)
	dt, has := Decoders[ext]
	if !has {
		return nil, fmt.Errorf("gi3d.DecodeFile: file extension: %v not found in Decoders list", ext)
	}
	dec := dt.New()
	files = dec.SetFiles(files)
	nf = len(files)

	var err error
	fs := make([]*os.File, nf)
	rs := make([]io.Reader, nf)
	defer func() {
		for _, r := range fs {
			if r != nil {
				r.Close()
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

// OpenObj open the given object(s) from given file(s) into given group in scene,
// using the decoder based on the file extension in first file name.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which can
//        be specified as second file name -- otherwise auto-searched based on
//        .obj filename, or a default material is used.
func (sc *Scene) OpenObj(files []string, gp *Group) error {
	dec, err := DecodeFile(files)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetGroup(sc, gp)
	sc.Init3D() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

// OpenNewObj open the given object(s) from given file(s) into a new group
// under given parent, using the decoder based on the file extension in first file name.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which can
//        be specified as second file name -- otherwise auto-searched based on
//        .obj filename, or a default material is used.
func (sc *Scene) OpenNewObj(files []string, parent ki.Ki) (*Group, error) {
	dec, err := DecodeFile(files)
	if err != nil {
		return nil, err
	}
	updt := sc.UpdateStart()
	_, fn := filepath.Split(files[0])
	gp := AddNewGroup(sc, parent, fn)
	dec.SetGroup(sc, gp)
	sc.Init3D() // needed after loading
	sc.UpdateEnd(updt)
	return gp, nil
}

// OpenScene open the given scene from given file(s),
// using the decoder based on the file extension in first file name.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which can
//        be specified as second file name -- otherwise auto-searched based on
//        .obj filename, or a default material is used.  Does not support full scene
//        data so only objects are loaded into a new group in scene.
func (sc *Scene) OpenScene(files []string) error {
	dec, err := DecodeFile(files)
	if err != nil {
		return err
	}
	updt := sc.UpdateStart()
	dec.SetScene(sc)
	sc.Init3D() // needed after loading
	sc.UpdateEnd(updt)
	return nil
}

// ReadObj reads the given object(s) from given reader(s) into given group in scene,
// using the decoder based on the given file extension.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which can
//        be specified as second file name -- otherwise auto-searched based on
//        .obj filename, or a default material is used.
func (sc *Scene) ReadObj(ext string, rs []io.Reader, gp *Group) error {
	dt, has := Decoders[ext]
	if !has {
		return fmt.Errorf("gi3d.ReadObj: file extension: %v not found in Decoders list", ext)
	}
	dec := dt.New()
	err := dec.Decode(rs)
	if err != nil {
		return err
	}
	dec.SetGroup(sc, gp)
	return nil
}

// ReadScene open the given scene from given file(s),
// using the decoder based on the file extension in first file name.
// Supported formats include:
// .obj = Wavefront OBJ format, including associated materials (.mtl) which can
//        be specified as second file name -- otherwise auto-searched based on
//        .obj filename, or a default material is used.  Does not support full scene
//        data so only objects are loaded into a new group in scene.
func (sc *Scene) ReadScene(ext string, rs []io.Reader, gp *Group) error {
	dt, has := Decoders[ext]
	if !has {
		return fmt.Errorf("gi3d.ReadScene: file extension: %v not found in Decoders list", ext)
	}
	dec := dt.New()
	err := dec.Decode(rs)
	if err != nil {
		return err
	}
	dec.SetScene(sc)
	return nil
}

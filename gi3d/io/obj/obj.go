// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This package is based extensively on https://github.com/g3n/engine :
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package obj is used to parse the Wavefront OBJ file format (*.obj), including
// associated materials (*.mtl). Not all features of the OBJ format are
// supported. Basic format info: https://en.wikipedia.org/wiki/Wavefront_.obj_file
package obj

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goki/gi/gi3d"
	"github.com/goki/gi/gist"
	"github.com/goki/mat32"
)

// note: gimain imports "github.com/goki/gi/gi3d/io/obj" to get this code
func init() {
	gi3d.Decoders[".obj"] = &Decoder{}
}

// Decoder contains all decoded data from the obj and mtl files.
// It also implements the gi3d.Decoder interface and an instance
// is registered to handle .obj files.
type Decoder struct {
	Objfile       string               // .obj filename (without path)
	Objdir        string               // path to .obj file
	Objects       []Object             // decoded objects
	Matlib        string               // name of the material lib
	Materials     map[string]*Material // maps material name to object
	Vertices      mat32.ArrayF32       // vertices positions array
	Normals       mat32.ArrayF32       // vertices normals
	Uvs           mat32.ArrayF32       // vertices texture coordinates
	Warnings      []string             // warning messages
	line          uint                 // current line number
	objCurrent    *Object              // current object
	matCurrent    *Material            // current material
	smoothCurrent bool                 // current smooth state
}

func (dec *Decoder) New() gi3d.Decoder {
	di := new(Decoder)
	di.Objects = make([]Object, 0)
	di.Warnings = make([]string, 0)
	di.Materials = make(map[string]*Material)
	di.Vertices = mat32.NewArrayF32(0, 0)
	di.Normals = mat32.NewArrayF32(0, 0)
	di.Uvs = mat32.NewArrayF32(0, 0)
	di.line = 1
	return di
}

// Destroy deletes data created during loading
func (dec *Decoder) Destroy() {
	for oi := range dec.Objects {
		ob := &dec.Objects[oi]
		ob.Destroy()
	}
	dec.Objects = nil
	dec.Warnings = nil
	dec.Materials = nil
	dec.Vertices = nil
	dec.Normals = nil
	dec.Uvs = nil
	dec.objCurrent = nil
	dec.matCurrent = nil
}

func (dec *Decoder) Desc() string {
	return ".obj = Wavefront OBJ format, including associated materials (.mtl) which can be specified as second file name -- otherwise auto-searched based on .obj filename, or a default material is used.  Only supports Object-level data, not full Scene (camera, lights etc)."
}

func (dec *Decoder) HasScene() bool {
	return false
}

func (dec *Decoder) SetFile(fname string) []string {
	dec.Objdir, dec.Objfile = filepath.Split(fname)
	mtlf := strings.TrimSuffix(fname, ".obj") + ".mtl"
	if _, err := os.Stat(mtlf); !os.IsNotExist(err) {
		return []string{fname, mtlf}
	} else {
		return []string{fname}
	}
}

// Decode reads the given data and decodes into Decoder tmp vars.
// if 2 args are passed, first is .obj and second is .mtl
func (dec *Decoder) Decode(rs []io.Reader) error {
	nf := len(rs)
	if nf == 0 {
		return errors.New("obj.Decoder: no readers passed")
	}
	// Parses obj lines
	err := dec.parse(rs[0], dec.parseObjLine)
	if err != nil {
		return err
	}

	dec.matCurrent = nil
	dec.line = 1
	useDef := nf == 1
	if nf > 1 {
		err = dec.parse(rs[1], dec.parseMtlLine)
		if err != nil {
			useDef = true
		}
	}
	if useDef {
		for key := range dec.Materials {
			dec.Materials[key] = defaultMat
		}
	}
	return nil
}

// Object contains all information about one decoded object
type Object struct {
	Name      string   // Object name
	Faces     []Face   // Faces
	materials []string // Materials used in this object
}

func (ob *Object) Destroy() {
	ob.materials = nil
	for fi := range ob.Faces {
		fc := &ob.Faces[fi]
		fc.Destroy()
	}
	ob.Faces = nil
}

// Face contains all information about an object face
type Face struct {
	Vertices []int  // Indices to the face vertices
	Uvs      []int  // Indices to the face UV coordinates
	Normals  []int  // Indices to the face normals
	Material string // Material name
	Smooth   bool   // Smooth face
}

func (fc *Face) Destroy() {
	fc.Vertices = nil
	fc.Uvs = nil
	fc.Normals = nil
}

// Material contains all information about an object material
type Material struct {
	Name       string      // Material name
	Illum      int         // Illumination model
	Opacity    float32     // Opacity factor
	Refraction float32     // Refraction factor
	Shininess  float32     // Shininess (specular exponent)
	Ambient    gist.Color  // Ambient color reflectivity
	Diffuse    gist.Color  // Diffuse color reflectivity
	Specular   gist.Color  // Specular color reflectivity
	Emissive   gist.Color  // Emissive color
	MapKd      string      // Texture file linked to diffuse color
	Tiling     gi3d.Tiling // Tiling parameters: repeat and offset
}

// Light gray default material used as when other materials cannot be loaded.
var defaultMat = &Material{
	Diffuse:   gist.Color{R: 0xA0, G: 0xA0, B: 0xA0, A: 0xFF},
	Ambient:   gist.Color{R: 0xA0, G: 0xA0, B: 0xA0, A: 0xFF},
	Specular:  gist.Color{R: 0x80, G: 0x80, B: 0x80, A: 0xFF},
	Shininess: 30.0,
}

// Local constants
const (
	blanks   = "\r\n\t "
	invINDEX = math.MaxUint32
	objType  = "obj"
	mtlType  = "mtl"
)

// SetScene sets group with with all the decoded objects.
func (dec *Decoder) SetScene(sc *gi3d.Scene) {
	gp := gi3d.AddNewGroup(sc, sc, dec.Objfile)
	dec.SetGroup(sc, gp)
}

// SetGroup sets group with with all the decoded objects.
// calls Destroy after to free memory
func (dec *Decoder) SetGroup(sc *gi3d.Scene, gp *gi3d.Group) {
	for i := range dec.Objects {
		obj := &dec.Objects[i]
		if len(obj.Faces) == 0 {
			continue
		}
		objgp := gi3d.AddNewGroup(sc, gp, obj.Name)
		dec.SetObject(sc, objgp, obj)
	}
	dec.Destroy()
}

// SetObject sets the object as a group with each gi3d.Solid as a mesh with unique material
func (dec *Decoder) SetObject(sc *gi3d.Scene, objgp *gi3d.Group, ob *Object) {
	matName := ""
	var sld *gi3d.Solid
	var ms *gi3d.GenMesh
	sldidx := 0
	idxs := make([]int, 0, 4)
	for fi := range ob.Faces {
		face := &ob.Faces[fi]
		if face.Material != matName || sld == nil {
			sldnm := fmt.Sprintf("%s_%d", ob.Name, sldidx)
			ms = &gi3d.GenMesh{}
			ms.Nm = sldnm
			sc.AddMeshUnique(ms)
			sld = gi3d.AddNewSolid(sc, objgp, sldnm, ms.Nm)
			matName = face.Material
			dec.SetMat(sc, sld, matName)
			sldidx++
		}
		// Copy face vertices to geometry
		// https://stackoverflow.com/questions/23723993/converting-quadriladerals-in-an-obj-file-into-triangles
		// logic for 0, i, i+1
		// note: the last comment at the end is *incorrect* -- it really is the triangle fans as impl here:

		idxs = idxs[:3]
		idxs[0] = dec.copyVertex(ms, face, 0)
		idxs[1] = dec.copyVertex(ms, face, 1)
		idxs[2] = dec.copyVertex(ms, face, 2)
		dec.addNorms(ms, 0, 1, 2, idxs)
		for idx := 2; idx < len(face.Vertices); idx++ {
			dec.setIndex(ms, face, 0, &idxs)
			dec.setIndex(ms, face, idx-1, &idxs)
			dec.setIndex(ms, face, idx, &idxs)
			dec.addNorms(ms, idx-3, idx-1, idx, idxs)
		}
	}
}

func (dec *Decoder) addNorms(ms *gi3d.GenMesh, ai, bi, ci int, idxs []int) {
	if ms.Norm.Size() >= ms.Vtx.Size() {
		return
	}
	var a, b, c mat32.Vec3
	ms.Vtx.GetVec3(3*idxs[ai], &a)
	ms.Vtx.GetVec3(3*idxs[bi], &b)
	ms.Vtx.GetVec3(3*idxs[ci], &c)
	nrm := mat32.Normal(a, b, c)
	for {
		ms.Norm.AppendVec3(nrm)
		if ms.Norm.Size() >= ms.Vtx.Size() {
			break
		}
	}
}

func (dec *Decoder) setIndex(ms *gi3d.GenMesh, face *Face, idx int, idxs *[]int) {
	if len(*idxs) > idx {
		ms.Idx.Append(uint32((*idxs)[idx]))
	} else {
		*idxs = append(*idxs, dec.copyVertex(ms, face, idx))
	}
}

func (dec *Decoder) copyVertex(ms *gi3d.GenMesh, face *Face, idx int) int {
	var vec3 mat32.Vec3
	var vec2 mat32.Vec2

	vidx := ms.Vtx.Size() / 3
	// Copy vertex position and append to geometry
	dec.Vertices.GetVec3(3*face.Vertices[idx], &vec3)
	ms.Vtx.AppendVec3(vec3)

	// Copy vertex normal and append to geometry
	if face.Normals[idx] != invINDEX {
		i := 3 * face.Normals[idx]
		if dec.Normals.Size() > i {
			dec.Normals.GetVec3(i, &vec3)
		}
		ms.Norm.AppendVec3(vec3)
	}

	// Copy vertex uv and append to geometry
	if face.Uvs[idx] != invINDEX {
		i := 2 * face.Uvs[idx]
		if dec.Uvs.Size() > i {
			dec.Uvs.GetVec2(i, &vec2)
		}
		ms.Tex.AppendVec2(vec2)
	}
	ms.Idx.Append(uint32(vidx))
	return vidx
}

// SetMat sets the material for object
func (dec *Decoder) SetMat(sc *gi3d.Scene, sld *gi3d.Solid, matnm string) {
	mat := dec.Materials[matnm]
	if mat == nil {
		mat = defaultMat
		// log warning
		msg := fmt.Sprintf("could not find material: %s for object %s. using default material.", matnm, sld.Name())
		dec.appendWarn(objType, msg)
	}
	sld.Mat.Defaults()
	sld.Mat.CullBack = false // obj files do not reliably work with this on!
	sld.Mat.Color = mat.Diffuse
	if mat.Opacity > 0 {
		sld.Mat.Color.A = uint8(mat.Opacity * float32(0xFF))
	}
	sld.Mat.Specular = mat.Specular
	if mat.Shininess != 0 {
		sld.Mat.Shiny = mat.Shininess
	}
	// Loads material textures if specified
	dec.loadTex(sc, sld, mat.MapKd, mat)
}

// loadTex loads given texture file
func (dec *Decoder) loadTex(sc *gi3d.Scene, sld *gi3d.Solid, texfn string, mat *Material) {
	if texfn == "" {
		return
	}
	var texPath string
	if filepath.IsAbs(texfn) {
		texPath = texfn
	} else {
		texPath = filepath.Join(dec.Objdir, texfn)
	}
	_, tfn := filepath.Split(texPath)
	tf, err := sc.TextureByNameTry(tfn)
	if err != nil {
		tf = gi3d.AddNewTextureFile(sc, tfn, texPath)
	}
	sld.Mat.SetTexture(sc, tf)
	if mat.Tiling.Repeat.X > 0 {
		sld.Mat.Tiling.Repeat = mat.Tiling.Repeat
	}
	sld.Mat.Tiling.Off = mat.Tiling.Off
}

// parse reads the lines from the specified reader and dispatch them
// to the specified line parser.
func (dec *Decoder) parse(reader io.Reader, parseLine func(string) error) error {
	bufin := bufio.NewReader(reader)
	dec.line = 1
	for {
		// Reads next line and abort on errors (not EOF)
		line, err := bufin.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		// Parses the line
		line = strings.Trim(line, blanks)
		perr := parseLine(line)
		if perr != nil {
			return perr
		}
		// If EOF ends of parsing.
		if err == io.EOF {
			break
		}
		dec.line++
	}
	return nil
}

// Parses obj file line, dispatching to specific parsers
func (dec *Decoder) parseObjLine(line string) error {
	// Ignore empty lines
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	// Ignore comment lines
	ltype := fields[0]
	if strings.HasPrefix(ltype, "#") {
		return nil
	}
	switch ltype {
	// Material library
	case "mtllib":
		return dec.parseMatlib(fields[1:])
	// Object name
	case "o":
		return dec.parseObject(fields[1:])
	// Group names. We are considering "group" the same as "object"
	// This may not be right
	case "g":
		return dec.parseObject(fields[1:])
	// Vertex coordinate
	case "v":
		return dec.parseVertex(fields[1:])
	// Vertex normal coordinate
	case "vn":
		return dec.parseNormal(fields[1:])
	// Vertex texture coordinate
	case "vt":
		return dec.parseTex(fields[1:])
	// Face vertex
	case "f":
		return dec.parseFace(fields[1:])
	// Use material
	case "usemtl":
		return dec.parseUsemtl(fields[1:])
	// Smooth
	case "s":
		return dec.parseSmooth(fields[1:])
	default:
		dec.appendWarn(objType, "field not supported: "+ltype)
	}
	return nil
}

// Parses a mtllib line:
// mtllib <name>
func (dec *Decoder) parseMatlib(fields []string) error {
	if len(fields) < 1 {
		return errors.New("Material library (mtllib) with no fields")
	}
	dec.Matlib = fields[0]
	return nil
}

// Parses an object line:
// o <name>
func (dec *Decoder) parseObject(fields []string) error {
	if len(fields) < 1 {
		return errors.New("Object line (o) with no fields")
	}

	dec.Objects = append(dec.Objects, makeObject(fields[0]))
	dec.objCurrent = &dec.Objects[len(dec.Objects)-1]
	return nil
}

// makes an Object with name.
func makeObject(name string) Object {
	var ob Object
	ob.Name = name
	ob.Faces = make([]Face, 0)
	ob.materials = make([]string, 0)
	return ob
}

// Parses a vertex position line
// v <x> <y> <z> [w]
func (dec *Decoder) parseVertex(fields []string) error {
	if len(fields) < 3 {
		return errors.New("Less than 3 vertices in 'v' line")
	}
	for _, f := range fields[:3] {
		val, err := strconv.ParseFloat(f, 32)
		if err != nil {
			return err
		}
		dec.Vertices.Append(float32(val))
	}
	return nil
}

// Parses a vertex normal line
// vn <x> <y> <z>
func (dec *Decoder) parseNormal(fields []string) error {
	if len(fields) < 3 {
		return errors.New("Less than 3 normals in 'vn' line")
	}
	for _, f := range fields[:3] {
		val, err := strconv.ParseFloat(f, 32)
		if err != nil {
			return err
		}
		dec.Normals.Append(float32(val))
	}
	return nil
}

// Parses a vertex texture coordinate line:
// vt <u> <v> <w>
func (dec *Decoder) parseTex(fields []string) error {
	if len(fields) < 2 {
		return errors.New("Less than 2 texture coords. in 'vt' line")
	}
	for _, f := range fields[:2] {
		val, err := strconv.ParseFloat(f, 32)
		if err != nil {
			return err
		}
		dec.Uvs.Append(float32(val))
	}
	return nil
}

// parseFace parses a face decription line:
// f v1[/vt1][/vn1] v2[/vt2][/vn2] v3[/vt3][/vn3] ...
func (dec *Decoder) parseFace(fields []string) error {
	if dec.objCurrent == nil {
		// if a face line is encountered before a group (g) or object (o),
		// create a new "default" object. This 'handles' the case when
		// a g or o line is not specified (allowed in OBJ format)
		dec.parseObject([]string{fmt.Sprintf("unnamed%d", dec.line)})
	}

	// If current object has no material, appends last material if defined
	if len(dec.objCurrent.materials) == 0 && dec.matCurrent != nil {
		dec.objCurrent.materials = append(dec.objCurrent.materials, dec.matCurrent.Name)
	}

	if len(fields) < 3 {
		return dec.formatError("Face line with less 3 fields")
	}
	var face Face
	face.Vertices = make([]int, len(fields))
	face.Uvs = make([]int, len(fields))
	face.Normals = make([]int, len(fields))
	if dec.matCurrent != nil {
		face.Material = dec.matCurrent.Name
	} else {
		// TODO (quillaja): do something better than spamming warnings for each line
		// dec.appendWarn(objType, "No material defined")
		face.Material = "internal default" // causes error on in NewGeom() if ""
		// dec.matCurrent = defaultMat
	}
	face.Smooth = dec.smoothCurrent

	for pos, f := range fields {
		// Separate the current field in its components: v vt vn
		vfields := strings.Split(f, "/")
		if len(vfields) < 1 {
			return dec.formatError("Face field with no parts")
		}

		// Get the index of this vertex position (must always exist)
		val, err := strconv.ParseInt(vfields[0], 10, 32)
		if err != nil {
			return err
		}

		// Positive index is an absolute vertex index
		if val > 0 {
			face.Vertices[pos] = int(val - 1)
			// Negative vertex index is relative to the last parsed vertex
		} else if val < 0 {
			current := (len(dec.Vertices) / 3) - 1
			face.Vertices[pos] = current + int(val) + 1
			// Vertex index could never be 0
		} else {
			return dec.formatError("Face vertex index value equal to 0")
		}

		// Get the index of this vertex UV coordinate (optional)
		if len(vfields) > 1 && len(vfields[1]) > 0 {
			val, err := strconv.ParseInt(vfields[1], 10, 32)
			if err != nil {
				return err
			}

			// Positive index is an absolute UV index
			if val > 0 {
				face.Uvs[pos] = int(val - 1)
				// Negative vertex index is relative to the last parsed uv
			} else if val < 0 {
				current := (len(dec.Uvs) / 2) - 1
				face.Uvs[pos] = current + int(val) + 1
				// UV index could never be 0
			} else {
				return dec.formatError("Face uv index value equal to 0")
			}
		} else {
			face.Uvs[pos] = invINDEX
		}

		// Get the index of this vertex normal (optional)
		if len(vfields) >= 3 {
			val, err = strconv.ParseInt(vfields[2], 10, 32)
			if err != nil {
				return err
			}

			// Positive index is an absolute normal index
			if val > 0 {
				face.Normals[pos] = int(val - 1)
				// Negative vertex index is relative to the last parsed normal
			} else if val < 0 {
				current := (len(dec.Normals) / 3) - 1
				face.Normals[pos] = current + int(val) + 1
				// Normal index could never be 0
			} else {
				return dec.formatError("Face normal index value equal to 0")
			}
		} else {
			face.Normals[pos] = invINDEX
		}
	}
	// Appends this face to the current object
	dec.objCurrent.Faces = append(dec.objCurrent.Faces, face)
	return nil
}

// parseUsemtl parses a "usemtl" decription line:
// usemtl <name>
func (dec *Decoder) parseUsemtl(fields []string) error {
	if len(fields) < 1 {
		return dec.formatError("Usemtl with no fields")
	}

	// NOTE(quillaja): see similar nil test in parseFace()
	if dec.objCurrent == nil {
		dec.parseObject([]string{fmt.Sprintf("unnamed%d", dec.line)})
	}

	// Checks if this material has already been parsed
	name := fields[0]
	mat := dec.Materials[name]
	// Creates material descriptor
	if mat == nil {
		mat = new(Material)
		mat.Name = name
		dec.Materials[name] = mat
	}
	dec.objCurrent.materials = append(dec.objCurrent.materials, name)
	// Set this as the current material
	dec.matCurrent = mat
	return nil
}

// parseSmooth parses a "s" decription line:
// s <0|1>
func (dec *Decoder) parseSmooth(fields []string) error {
	if len(fields) < 1 {
		return dec.formatError("'s' with no fields")
	}

	if fields[0] == "0" || fields[0] == "off" {
		dec.smoothCurrent = false
		return nil
	}
	if fields[0] == "1" || fields[0] == "on" {
		dec.smoothCurrent = true
		return nil
	}
	return dec.formatError("'s' with invalid value")
}

/******************************************************************************
mtl parse functions
*/

// Parses material file line, dispatching to specific parsers
func (dec *Decoder) parseMtlLine(line string) error {
	// Ignore empty lines
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	// Ignore comment lines
	ltype := fields[0]
	if strings.HasPrefix(ltype, "#") {
		return nil
	}
	switch ltype {
	case "newmtl":
		return dec.parseNewmtl(fields[1:])
	case "d":
		return dec.parseDissolve(fields[1:])
	case "Ka":
		return dec.parseKa(fields[1:])
	case "Kd":
		return dec.parseKd(fields[1:])
	case "Ke":
		return dec.parseKe(fields[1:])
	case "Ks":
		return dec.parseKs(fields[1:])
	case "Ni":
		return dec.parseNi(fields[1:])
	case "Ns":
		return dec.parseNs(fields[1:])
	case "illum":
		return dec.parseIllum(fields[1:])
	case "map_Kd":
		return dec.parseMapKd(fields[1:])
	default:
		dec.appendWarn(mtlType, "field not supported: "+ltype)
	}
	return nil
}

// Parses new material definition
// newmtl <mat_name>
func (dec *Decoder) parseNewmtl(fields []string) error {
	if len(fields) < 1 {
		return dec.formatError("newmtl with no fields")
	}
	// Checks if material has already been seen
	name := fields[0]
	mat := dec.Materials[name]
	// Creates material descriptor
	if mat == nil {
		mat = new(Material)
		mat.Name = name
		dec.Materials[name] = mat
	}
	dec.matCurrent = mat
	return nil
}

// Parses the dissolve factor (opacity)
// d <factor>
func (dec *Decoder) parseDissolve(fields []string) error {
	if len(fields) < 1 {
		return dec.formatError("'d' with no fields")
	}
	val, err := strconv.ParseFloat(fields[0], 32)
	if err != nil {
		return dec.formatError("'d' parse float error")
	}
	dec.matCurrent.Opacity = float32(val)
	return nil
}

// Parses ambient reflectivity:
// Ka r g b
func (dec *Decoder) parseKa(fields []string) error {
	if len(fields) < 3 {
		return dec.formatError("'Ka' with less than 3 fields")
	}
	var colors [3]float32
	for pos, f := range fields[:3] {
		val, err := strconv.ParseFloat(f, 32)
		if err != nil {
			return err
		}
		colors[pos] = float32(val)
	}
	dec.matCurrent.Ambient.SetFloat32(colors[0], colors[1], colors[2], 1)
	return nil
}

// Parses diffuse reflectivity:
// Kd r g b
func (dec *Decoder) parseKd(fields []string) error {
	if len(fields) < 3 {
		return dec.formatError("'Kd' with less than 3 fields")
	}
	var colors [3]float32
	for pos, f := range fields[:3] {
		val, err := strconv.ParseFloat(f, 32)
		if err != nil {
			return err
		}
		colors[pos] = float32(val)
	}
	dec.matCurrent.Diffuse.SetFloat32(colors[0], colors[1], colors[2], 1)
	return nil
}

// Parses emissive color:
// Ke r g b
func (dec *Decoder) parseKe(fields []string) error {
	if len(fields) < 3 {
		return dec.formatError("'Ke' with less than 3 fields")
	}
	var colors [3]float32
	for pos, f := range fields[:3] {
		val, err := strconv.ParseFloat(f, 32)
		if err != nil {
			return err
		}
		colors[pos] = float32(val)
	}
	dec.matCurrent.Emissive.SetFloat32(colors[0], colors[1], colors[2], 1)
	return nil
}

// Parses specular reflectivity:
// Ks r g b
func (dec *Decoder) parseKs(fields []string) error {
	if len(fields) < 3 {
		return dec.formatError("'Ks' with less than 3 fields")
	}
	var colors [3]float32
	for pos, f := range fields[:3] {
		val, err := strconv.ParseFloat(f, 32)
		if err != nil {
			return err
		}
		colors[pos] = float32(val)
	}
	dec.matCurrent.Specular.SetFloat32(colors[0], colors[1], colors[2], 1)
	return nil
}

// Parses optical density, also known as index of refraction
// Ni <optical_density>
func (dec *Decoder) parseNi(fields []string) error {
	if len(fields) < 1 {
		return dec.formatError("'Ni' with no fields")
	}
	val, err := strconv.ParseFloat(fields[0], 32)
	if err != nil {
		return dec.formatError("'d' parse float error")
	}
	dec.matCurrent.Refraction = float32(val)
	return nil
}

// Parses specular exponent
// Ns <specular_exponent>
func (dec *Decoder) parseNs(fields []string) error {
	if len(fields) < 1 {
		return dec.formatError("'Ns' with no fields")
	}
	val, err := strconv.ParseFloat(fields[0], 32)
	if err != nil {
		return dec.formatError("'d' parse float error")
	}
	dec.matCurrent.Shininess = float32(val)
	return nil
}

// Parses illumination model (0 to 10)
// illum <ilum_#>
func (dec *Decoder) parseIllum(fields []string) error {
	if len(fields) < 1 {
		return dec.formatError("'illum' with no fields")
	}
	val, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil {
		return dec.formatError("'d' parse int error")
	}
	dec.matCurrent.Illum = int(val)
	return nil
}

// Parses color texture linked to the diffuse reflectivity of the material
// map_Kd [-options] <filename>
func (dec *Decoder) parseMapKd(fields []string) error {
	if len(fields) < 1 {
		return dec.formatError("No fields")
	}
	nf := len(fields)
	for i := 0; i < nf; i++ {
		f := fields[i]
		if f[0] == '-' {
			switch f {
			case "-s":
				r1, _ := strconv.ParseFloat(fields[i+1], 32)
				r2 := r1
				i++
				if len(fields) > i+2 {
					rt, err := strconv.ParseFloat(fields[i+2], 32)
					if err == nil {
						r2 = rt
						i++
					}
				}
				dec.matCurrent.Tiling.Repeat.Set(float32(r1), float32(r2))
			case "-o":
				r1, _ := strconv.ParseFloat(fields[i+1], 32)
				r2 := r1
				i++
				if len(fields) > i+2 {
					rt, err := strconv.ParseFloat(fields[i+2], 32)
					if err == nil {
						r2 = rt
						i++
					}
				}
				dec.matCurrent.Tiling.Off.Set(float32(r1), float32(r2))
			}
		} else {
			dec.matCurrent.MapKd = f
		}
	}
	return nil
}

func (dec *Decoder) formatError(msg string) error {
	return fmt.Errorf("%s in line:%d", msg, dec.line)
}

func (dec *Decoder) appendWarn(ftype string, msg string) {

	wline := fmt.Sprintf("%s(%d): %s", ftype, dec.line, msg)
	dec.Warnings = append(dec.Warnings, wline)
}

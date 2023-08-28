// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package toml

import (
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"goki.dev/ki/v2/dirs"
)

// OpenFromPaths reads object from given TOML file,
// looking on paths for the file.
func OpenFromPaths(obj any, file string, paths []string) error {
	filename, err := dirs.FindFileOnPaths(paths, file)
	if err != nil {
		log.Println(err)
		return err
	}
	// _, err = toml.DecodeFile(fp, obj)
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return Read(obj, bufio.NewReader(fp))
}

// Open reads object from given TOML file.
func Open(obj any, filename string) error {
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return Read(obj, bufio.NewReader(fp))
}

// OpenFS reads object from given TOML file,
// using the fs.FS filesystem -- e.g., for embed files.
func OpenFS(obj any, fsys fs.FS, file string) error {
	fp, err := fsys.Open(file)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return Read(obj, bufio.NewReader(fp))
}

// Read reads object TOML encoding from given reader,
func Read(obj any, reader io.Reader) error {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(err)
		return err
	}
	return ReadBytes(obj, b)
}

// ReadBytes reads TOML from given bytes,
func ReadBytes(obj any, b []byte) error {
	err := toml.Unmarshal(b, obj)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// Save writes TOML to given file.
func Save(obj any, file string) error {
	fp, err := os.Create(file)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	bw := bufio.NewWriter(fp)
	err = Write(obj, bw)
	if err != nil {
		log.Println(err)
		return err
	}
	err = bw.Flush()
	if err != nil {
		log.Println(err)
	}
	return err
}

// Write writes TOML to given writer.
func Write(obj any, writer io.Writer) error {
	enc := toml.NewEncoder(writer)
	return enc.Encode(obj)
}

// WriteBytes writes TOML returning bytes.
func WriteBytes(obj any) []byte {
	var b bytes.Buffer
	enc := toml.NewEncoder(&b)
	enc.Encode(obj)
	return b.Bytes()
}

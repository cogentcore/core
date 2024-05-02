// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"slices"
)

func ReadFileLines(fn string) ([][]byte, error) {
	nl := []byte("\n")
	buf, err := os.ReadFile(fn)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	lines := bytes.Split(buf, nl)
	return lines, nil
}

// Extracts comment-directive tagged regions from .go files
func ExtractGoFiles(files []string) map[string][]byte {
	sls := map[string][][]byte{}
	key := []byte("//gosl: ")
	start := []byte("start")
	hlsl := []byte("hlsl")
	nohlsl := []byte("nohlsl")
	end := []byte("end")
	nl := []byte("\n")
	include := []byte("#include")

	for _, fn := range files {
		if !strings.HasSuffix(fn, ".go") {
			continue
		}
		lines, err := ReadFileLines(fn)
		if err != nil {
			continue
		}

		inReg := false
		inHlsl := false
		inNoHlsl := false
		var outLns [][]byte
		slFn := ""
		for _, ln := range lines {
			tln := bytes.TrimSpace(ln)
			isKey := bytes.HasPrefix(tln, key)
			var keyStr []byte
			if isKey {
				keyStr = tln[len(key):]
				// fmt.Printf("key: %s\n", string(keyStr))
			}
			switch {
			case inReg && isKey && bytes.HasPrefix(keyStr, end):
				if inHlsl || inNoHlsl {
					outLns = append(outLns, ln)
				}
				sls[slFn] = outLns
				inReg = false
				inHlsl = false
				inNoHlsl = false
			case inReg:
				for pkg := range LoadedPackageNames { // remove package prefixes
					if !bytes.Contains(ln, include) {
						ln = bytes.ReplaceAll(ln, []byte(pkg+"."), []byte{})
					}
				}
				outLns = append(outLns, ln)
			case isKey && bytes.HasPrefix(keyStr, start):
				inReg = true
				slFn = string(keyStr[len(start)+1:])
				outLns = sls[slFn]
			case isKey && bytes.HasPrefix(keyStr, nohlsl):
				inReg = true
				inNoHlsl = true
				slFn = string(keyStr[len(nohlsl)+1:])
				outLns = sls[slFn]
				outLns = append(outLns, ln) // key to include self here
			case isKey && bytes.HasPrefix(keyStr, hlsl):
				inReg = true
				inHlsl = true
				slFn = string(keyStr[len(hlsl)+1:])
				outLns = sls[slFn]
				outLns = append(outLns, ln)
			}
		}
	}

	rsls := make(map[string][]byte)
	for fn, lns := range sls {
		outfn := filepath.Join(*outDir, fn+".go")
		olns := [][]byte{}
		olns = append(olns, []byte("package main"))
		olns = append(olns, []byte(`import "math"`))
		olns = append(olns, lns...)
		res := bytes.Join(olns, nl)
		ioutil.WriteFile(outfn, res, 0644)
		cmd := exec.Command("goimports", "-w", fn+".go") // get imports
		cmd.Dir, _ = filepath.Abs(*outDir)
		out, err := cmd.CombinedOutput()
		_ = out
		// fmt.Printf("\n################\ngoimports output for: %s\n%s\n", outfn, out)
		if err != nil {
			log.Println(err)
		}
		rsls[fn] = bytes.Join(lns, nl)
	}

	return rsls
}

// ExtractHLSL extracts the HLSL code embedded within .Go files.
// Returns true if HLSL contains a void main( function.
func ExtractHLSL(buf []byte) ([]byte, bool) {
	key := []byte("//gosl: ")
	hlsl := []byte("hlsl")
	nohlsl := []byte("nohlsl")
	end := []byte("end")
	nl := []byte("\n")
	stComment := []byte("/*")
	edComment := []byte("*/")
	comment := []byte("// ")
	pack := []byte("package")
	imp := []byte("import")
	main := []byte("void main(")
	lparen := []byte("(")
	rparen := []byte(")")

	lines := bytes.Split(buf, nl)

	mx := min(10, len(lines))
	stln := 0
	gotImp := false
	for li := 0; li < mx; li++ {
		ln := lines[li]
		switch {
		case bytes.HasPrefix(ln, pack):
			stln = li + 1
		case bytes.HasPrefix(ln, imp):
			if bytes.HasSuffix(ln, lparen) {
				gotImp = true
			} else {
				stln = li + 1
			}
		case gotImp && bytes.HasPrefix(ln, rparen):
			stln = li + 1
		}
	}

	lines = lines[stln:] // get rid of package, import

	hasMain := false
	inHlsl := false
	inNoHlsl := false
	noHlslStart := 0
	for li := 0; li < len(lines); li++ {
		ln := lines[li]
		isKey := bytes.HasPrefix(ln, key)
		var keyStr []byte
		if isKey {
			keyStr = ln[len(key):]
			// fmt.Printf("key: %s\n", string(keyStr))
		}
		switch {
		case inNoHlsl && isKey && bytes.HasPrefix(keyStr, end):
			lines = slices.Delete(lines, noHlslStart, li+1)
			li -= ((li + 1) - noHlslStart)
			inNoHlsl = false
		case inHlsl && isKey && bytes.HasPrefix(keyStr, end):
			lines = slices.Delete(lines, li, li+1)
			li--
			inHlsl = false
		case inHlsl:
			del := false
			switch {
			case bytes.HasPrefix(ln, stComment) || bytes.HasPrefix(ln, edComment):
				lines = slices.Delete(lines, li, li+1)
				li--
				del = true
			case bytes.HasPrefix(ln, comment):
				lines[li] = ln[3:]
			}
			if !del {
				if bytes.HasPrefix(lines[li], main) {
					hasMain = true
				}
			}
		case isKey && bytes.HasPrefix(keyStr, hlsl):
			inHlsl = true
			lines = slices.Delete(lines, li, li+1)
			li--
		case isKey && bytes.HasPrefix(keyStr, nohlsl):
			inNoHlsl = true
			noHlslStart = li
		}
	}
	return bytes.Join(lines, nl), hasMain
}

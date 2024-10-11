// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"slices"
)

// ExtractFiles processes all the package files and saves the corresponding
// .go files with simple go header.
func (st *State) ExtractFiles() {
	st.ImportPackages = make(map[string]bool)
	for impath := range st.GoImports {
		_, pkg := filepath.Split(impath)
		if pkg != "math32" {
			st.ImportPackages[pkg] = true
		}
	}

	for fn, fl := range st.GoFiles {
		hasVars := false
		fl.Lines, hasVars = st.ExtractGosl(fl.Lines)
		if hasVars {
			st.GoVarsFiles[fn] = fl
			delete(st.GoFiles, fn)
		}
		WriteFileLines(filepath.Join(st.ImportsDir, fn), st.AppendGoHeader(fl.Lines))
	}
}

// ExtractImports processes all the imported files and saves the corresponding
// .go files with simple go header.
func (st *State) ExtractImports() {
	if len(st.GoImports) == 0 {
		return
	}
	for impath, im := range st.GoImports {
		_, pkg := filepath.Split(impath)
		for fn, fl := range im {
			fl.Lines, _ = st.ExtractGosl(fl.Lines)
			WriteFileLines(filepath.Join(st.ImportsDir, pkg+"-"+fn), st.AppendGoHeader(fl.Lines))
		}
	}
}

// ExtractGosl gosl comment-directive tagged regions from given file.
func (st *State) ExtractGosl(lines [][]byte) (outLines [][]byte, hasVars bool) {
	key := []byte("//gosl:")
	start := []byte("start")
	wgsl := []byte("wgsl")
	nowgsl := []byte("nowgsl")
	end := []byte("end")
	vars := []byte("vars")
	imp := []byte("import")
	kernel := []byte("//gosl:kernel")
	fnc := []byte("func")

	inReg := false
	inHlsl := false
	inNoHlsl := false
	for li, ln := range lines {
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
				outLines = append(outLines, ln)
			}
			inReg = false
			inHlsl = false
			inNoHlsl = false
		case inReg && isKey && bytes.HasPrefix(keyStr, vars):
			hasVars = true
			outLines = append(outLines, ln)
		case inReg:
			for pkg := range st.ImportPackages { // remove package prefixes
				if !bytes.Contains(ln, imp) {
					ln = bytes.ReplaceAll(ln, []byte(pkg+"."), []byte{})
				}
			}
			if bytes.HasPrefix(ln, fnc) && bytes.Contains(ln, kernel) {
				sysnm := strings.TrimSpace(string(ln[bytes.LastIndex(ln, kernel)+len(kernel):]))
				sy := st.System(sysnm)
				fcall := string(ln[5:])
				lp := strings.Index(fcall, "(")
				rp := strings.LastIndex(fcall, ")")
				args := fcall[lp+1 : rp]
				fnm := fcall[:lp]
				funcode := ""
				for ki := li + 1; ki < len(lines); ki++ {
					kl := lines[ki]
					if len(kl) > 0 && kl[0] == '}' {
						break
					}
					funcode += string(kl) + "\n"
				}
				kn := &Kernel{Name: fnm, Args: args, FuncCode: funcode}
				sy.Kernels[fnm] = kn
				if st.Config.Debug {
					fmt.Println("\tAdded kernel:", fnm, "args:", args, "system:", sy.Name)
				}
			}
			outLines = append(outLines, ln)
		case isKey && bytes.HasPrefix(keyStr, start):
			inReg = true
		case isKey && bytes.HasPrefix(keyStr, nowgsl):
			inReg = true
			inNoHlsl = true
			outLines = append(outLines, ln) // key to include self here
		case isKey && bytes.HasPrefix(keyStr, wgsl):
			inReg = true
			inHlsl = true
			outLines = append(outLines, ln)
		}
	}
	return
}

// AppendGoHeader appends Go header
func (st *State) AppendGoHeader(lines [][]byte) [][]byte {
	olns := make([][]byte, 0, len(lines)+10)
	olns = append(olns, []byte("package main"))
	olns = append(olns, []byte(`import (
	"math"
	"cogentcore.org/core/goal/gosl/slbool"
	"cogentcore.org/core/goal/gosl/slrand"
	"cogentcore.org/core/goal/gosl/sltype"
	"cogentcore.org/core/tensor"
`))
	for impath := range st.GoImports {
		if strings.Contains(impath, "core/goal/gosl") {
			continue
		}
		olns = append(olns, []byte("\t\""+impath+"\""))
	}
	olns = append(olns, []byte(")"))
	olns = append(olns, lines...)
	SlBoolReplace(olns)
	return olns
}

// ExtractWGSL extracts the WGSL code embedded within .Go files,
// which is commented out in the Go code -- remove comments.
func (st *State) ExtractWGSL(lines [][]byte) [][]byte {
	key := []byte("//gosl:")
	wgsl := []byte("wgsl")
	nowgsl := []byte("nowgsl")
	end := []byte("end")
	stComment := []byte("/*")
	edComment := []byte("*/")
	comment := []byte("// ")
	pack := []byte("package")
	imp := []byte("import")
	lparen := []byte("(")
	rparen := []byte(")")

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
			switch {
			case bytes.HasPrefix(ln, stComment) || bytes.HasPrefix(ln, edComment):
				lines = slices.Delete(lines, li, li+1)
				li--
			case bytes.HasPrefix(ln, comment):
				lines[li] = ln[3:]
			}
		case isKey && bytes.HasPrefix(keyStr, wgsl):
			inHlsl = true
			lines = slices.Delete(lines, li, li+1)
			li--
		case isKey && bytes.HasPrefix(keyStr, nowgsl):
			inNoHlsl = true
			noHlslStart = li
		}
	}
	return lines
}

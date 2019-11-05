package bytes

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func assertAlmostEqual(t *testing.T, a, b float64, places int) {
	if math.Abs(a-b) > math.Pow10(-places) {
		t.Errorf("%.7f != %.7f", a, b)
	}
}

func assertEqual(t *testing.T, a, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("%v != %v", a, b)
	}
}

func splitChars(s string) [][]byte {
	chars := make([][]byte, 0, len(s))
	// Assume ASCII inputs
	for i := 0; i != len(s); i++ {
		chars = append(chars, []byte{s[i]})
	}
	return chars
}

func stringsToBytes(in ...string) (out [][]byte) {
	out = make([][]byte, len(in))
	for i, s := range(in) {
		out[i] = []byte(s)
	}
	return
}

func bytesToStrings(in ...[]byte) (out []string) {
	out = make([]string, len(in))
	for i, s := range(in) {
		out[i] = string(s)
	}
	return
}

func TestlistifyString(t *testing.T) {
	lst := listifyString([]byte("qwerty"))
	if reflect.DeepEqual(lst, splitChars("qwerty")) != true {
		t.Fatal("listifyString failure:", lst)
	}
}

func TestSequenceMatcherRatio(t *testing.T) {
	s := NewMatcher(splitChars("abcd"), splitChars("bcde"))
	assertEqual(t, s.Ratio(), 0.75)
	assertEqual(t, s.QuickRatio(), 0.75)
	assertEqual(t, s.RealQuickRatio(), 1.0)
}

func TestGetOptCodes(t *testing.T) {
	a := "qabxcd"
	b := "abycdf"
	s := NewMatcher(splitChars(a), splitChars(b))
	w := &bytes.Buffer{}
	for _, op := range s.GetOpCodes() {
		fmt.Fprintf(w, "%s a[%d:%d], (%s) b[%d:%d] (%s)\n", string(op.Tag),
			op.I1, op.I2, a[op.I1:op.I2], op.J1, op.J2, b[op.J1:op.J2])
	}
	result := string(w.Bytes())
	expected := `d a[0:1], (q) b[0:0] ()
e a[1:3], (ab) b[0:2] (ab)
r a[3:4], (x) b[2:3] (y)
e a[4:6], (cd) b[3:5] (cd)
i a[6:6], () b[5:6] (f)
`
	if expected != result {
		t.Errorf("unexpected op codes: \n%s", result)
	}
}

func TestGroupedOpCodes(t *testing.T) {
	a := [][]byte{}
	for i := 0; i != 39; i++ {
		a = append(a, []byte(fmt.Sprintf("%02d", i)))
	}
	b := [][]byte{}
	b = append(b, a[:8]...)
	b = append(b, []byte(" i"))
	b = append(b, a[8:19]...)
	b = append(b, []byte(" x"))
	b = append(b, a[20:22]...)
	b = append(b, a[27:34]...)
	b = append(b, []byte(" y"))
	b = append(b, a[35:]...)
	s := NewMatcher(a, b)
	w := &bytes.Buffer{}
	for _, g := range s.GetGroupedOpCodes(-1) {
		fmt.Fprintf(w, "group\n")
		for _, op := range g {
			fmt.Fprintf(w, "  %s, %d, %d, %d, %d\n", string(op.Tag),
				op.I1, op.I2, op.J1, op.J2)
		}
	}
	result := string(w.Bytes())
	expected := `group
  e, 5, 8, 5, 8
  i, 8, 8, 8, 9
  e, 8, 11, 9, 12
group
  e, 16, 19, 17, 20
  r, 19, 20, 20, 21
  e, 20, 22, 21, 23
  d, 22, 27, 23, 23
  e, 27, 30, 23, 26
group
  e, 31, 34, 27, 30
  r, 34, 35, 30, 31
  e, 35, 38, 31, 34
`
	if expected != result {
		t.Errorf("unexpected op codes: \n%s", result)
	}
}

func ExampleGetUnifiedDiffCode() {
	a := `one
two
three
four
fmt.Printf("%s,%T",a,b)`
	b := `zero
one
three
four`
	diff := UnifiedDiff{
		A:        SplitLines([]byte(a)),
		B:        SplitLines([]byte(b)),
		FromFile: "Original",
		FromDate: "2005-01-26 23:30:50",
		ToFile:   "Current",
		ToDate:   "2010-04-02 10:20:52",
		Context:  3,
	}
	result, _ := GetUnifiedDiffString(diff)
	fmt.Println(strings.Replace(string(result), "\t", " ", -1))
	// Output:
	// --- Original 2005-01-26 23:30:50
	// +++ Current 2010-04-02 10:20:52
	// @@ -1,5 +1,4 @@
	// +zero
	//  one
	// -two
	//  three
	//  four
	// -fmt.Printf("%s,%T",a,b)
}

func ExampleGetContextDiffCode() {
	a := `one
two
three
four
fmt.Printf("%s,%T",a,b)`
	b := `zero
one
tree
four`
	diff := ContextDiff{
		A:        SplitLines([]byte(a)),
		B:        SplitLines([]byte(b)),
		FromFile: "Original",
		ToFile:   "Current",
		Context:  3,
		Eol:      []byte{'\n'},
	}
	result, _ := GetContextDiffString(diff)
	fmt.Print(strings.Replace(string(result), "\t", " ", -1))
	// Output:
	// *** Original
	// --- Current
	// ***************
	// *** 1,5 ****
	//   one
	// ! two
	// ! three
	//   four
	// - fmt.Printf("%s,%T",a,b)
	// --- 1,4 ----
	// + zero
	//   one
	// ! tree
	//   four
}

func ExampleGetContextDiffString() {
	a := `one
two
three
four`
	b := `zero
one
tree
four`
	diff := ContextDiff{
		A:        SplitLines([]byte(a)),
		B:        SplitLines([]byte(b)),
		FromFile: "Original",
		ToFile:   "Current",
		Context:  3,
		Eol:      []byte{'\n'},
	}
	result, _ := GetContextDiffString(diff)
	fmt.Printf(strings.Replace(string(result), "\t", " ", -1))
	// Output:
	// *** Original
	// --- Current
	// ***************
	// *** 1,4 ****
	//   one
	// ! two
	// ! three
	//   four
	// --- 1,4 ----
	// + zero
	//   one
	// ! tree
	//   four
}

func rep(s string, count int) string {
	return strings.Repeat(s, count)
}

func TestWithAsciiOneInsert(t *testing.T) {
	sm := NewMatcher(splitChars(rep("b", 100)),
		splitChars("a"+rep("b", 100)))
	assertAlmostEqual(t, sm.Ratio(), 0.995, 3)
	assertEqual(t, sm.GetOpCodes(),
		[]OpCode{{'i', 0, 0, 0, 1}, {'e', 0, 100, 1, 101}})
	assertEqual(t, len(sm.bPopular), 0)

	sm = NewMatcher(splitChars(rep("b", 100)),
		splitChars(rep("b", 50)+"a"+rep("b", 50)))
	assertAlmostEqual(t, sm.Ratio(), 0.995, 3)
	assertEqual(t, sm.GetOpCodes(),
		[]OpCode{{'e', 0, 50, 0, 50}, {'i', 50, 50, 50, 51}, {'e', 50, 100, 51, 101}})
	assertEqual(t, len(sm.bPopular), 0)
}

func TestWithAsciiOnDelete(t *testing.T) {
	sm := NewMatcher(splitChars(rep("a", 40)+"c"+rep("b", 40)),
		splitChars(rep("a", 40)+rep("b", 40)))
	assertAlmostEqual(t, sm.Ratio(), 0.994, 3)
	assertEqual(t, sm.GetOpCodes(),
		[]OpCode{{'e', 0, 40, 0, 40}, {'d', 40, 41, 40, 40}, {'e', 41, 81, 40, 80}})
}

func TestWithAsciiBJunk(t *testing.T) {
	isJunk := func(s []byte) bool {
		return len(s) == 1 && s[0] == ' '
	}
	sm := NewMatcherWithJunk(splitChars(rep("a", 40)+rep("b", 40)),
		splitChars(rep("a", 44)+rep("b", 40)), true, isJunk)
	assertEqual(t, sm.bJunk, map[lineHash]struct{}{})

	sm = NewMatcherWithJunk(splitChars(rep("a", 40)+rep("b", 40)),
		splitChars(rep("a", 44)+rep("b", 40)+rep(" ", 20)), false, isJunk)
	assertEqual(t, sm.bJunk, map[lineHash]struct{}{_hash(SPACE): struct{}{}})

	isJunk = func(s []byte) bool {
		return len(s) == 1 && (s[0] == ' ' || s[0] == 'b')
	}
	sm = NewMatcherWithJunk(splitChars(rep("a", 40)+rep("b", 40)),
		splitChars(rep("a", 44)+rep("b", 40)+rep(" ", 20)), false, isJunk)
	assertEqual(t, sm.bJunk, map[lineHash]struct{}{_hash(SPACE): struct{}{}, _hash([]byte{'b'}): struct{}{}})
}

func TestSFBugsRatioForNullSeqn(t *testing.T) {
	sm := NewMatcher(nil, nil)
	assertEqual(t, sm.Ratio(), 1.0)
	assertEqual(t, sm.QuickRatio(), 1.0)
	assertEqual(t, sm.RealQuickRatio(), 1.0)
}

func TestSFBugsComparingEmptyLists(t *testing.T) {
	groups := NewMatcher(nil, nil).GetGroupedOpCodes(-1)
	assertEqual(t, len(groups), 0)
	diff := UnifiedDiff{
		FromFile: "Original",
		ToFile:   "Current",
		Context:  3,
	}
	result, err := GetUnifiedDiffString(diff)
	assertEqual(t, err, nil)
	assertEqual(t, string(result), "")
}

func TestOutputFormatRangeFormatUnified(t *testing.T) {
	// Per the diff spec at http://www.unix.org/single_unix_specification/
	//
	// Each <range> field shall be of the form:
	//   %1d", <beginning line number>  if the range contains exactly one line,
	// and:
	//  "%1d,%1d", <beginning line number>, <number of lines> otherwise.
	// If a range is empty, its beginning line number shall be the number of
	// the line just before the range, or 0 if the empty range starts the file.
	fmt := formatRangeUnified
	fm := func (a, b int) string { return string(fmt(a,b)) }
	assertEqual(t, fm(3, 3), "3,0")
	assertEqual(t, fm(3, 4), "4")
	assertEqual(t, fm(3, 5), "4,2")
	assertEqual(t, fm(3, 6), "4,3")
	assertEqual(t, fm(0, 0), "0,0")
}

func TestOutputFormatRangeFormatContext(t *testing.T) {
	// Per the diff spec at http://www.unix.org/single_unix_specification/
	//
	// The range of lines in file1 shall be written in the following format
	// if the range contains two or more lines:
	//     "*** %d,%d ****\n", <beginning line number>, <ending line number>
	// and the following format otherwise:
	//     "*** %d ****\n", <ending line number>
	// The ending line number of an empty range shall be the number of the preceding line,
	// or 0 if the range is at the start of the file.
	//
	// Next, the range of lines in file2 shall be written in the following format
	// if the range contains two or more lines:
	//     "--- %d,%d ----\n", <beginning line number>, <ending line number>
	// and the following format otherwise:
	//     "--- %d ----\n", <ending line number>
	fmt := formatRangeContext
	fm := func (a, b int) string { return string(fmt(a,b)) }
	assertEqual(t, fm(3, 3), "3")
	assertEqual(t, fm(3, 4), "4")
	assertEqual(t, fm(3, 5), "4,5")
	assertEqual(t, fm(3, 6), "4,6")
	assertEqual(t, fm(0, 0), "0")
}

func TestOutputFormatTabDelimiter(t *testing.T) {
	diff := UnifiedDiff{
		A:        splitChars("one"),
		B:        splitChars("two"),
		FromFile: "Original",
		FromDate: "2005-01-26 23:30:50",
		ToFile:   "Current",
		ToDate:   "2010-04-12 10:20:52",
		Eol:      []byte{'\n'},
	}
	ud, err := GetUnifiedDiffString(diff)
	assertEqual(t, err, nil)
	assertEqual(t, SplitLines(ud)[:2], stringsToBytes(
		"--- Original\t2005-01-26 23:30:50\n",
		"+++ Current\t2010-04-12 10:20:52\n",
	))
	cd, err := GetContextDiffString(ContextDiff(diff))
	assertEqual(t, err, nil)
	assertEqual(t, SplitLines(cd)[:2], stringsToBytes(
		"*** Original\t2005-01-26 23:30:50\n",
		"--- Current\t2010-04-12 10:20:52\n",
	))
}

func TestOutputFormatNoTrailingTabOnEmptyFiledate(t *testing.T) {
	diff := UnifiedDiff{
		A:        splitChars("one"),
		B:        splitChars("two"),
		FromFile: "Original",
		ToFile:   "Current",
		Eol:      []byte{'\n'},
	}
	ud, err := GetUnifiedDiffString(diff)
	assertEqual(t, err, nil)
	assertEqual(t, SplitLines(ud)[:2], stringsToBytes("--- Original\n", "+++ Current\n"))

	cd, err := GetContextDiffString(ContextDiff(diff))
	assertEqual(t, err, nil)
	assertEqual(t, SplitLines(cd)[:2], stringsToBytes("*** Original\n", "--- Current\n"))
}

func TestOmitFilenames(t *testing.T) {
	diff := UnifiedDiff{
		A:   SplitLines([]byte("o\nn\ne\n")),
		B:   SplitLines([]byte("t\nw\no\n")),
		Eol: []byte{'\n'},
	}
	ud, err := GetUnifiedDiffString(diff)
	assertEqual(t, err, nil)
	assertEqual(t, SplitLines(ud), stringsToBytes(
		"@@ -0,0 +1,2 @@\n",
		"+t\n",
		"+w\n",
		"@@ -2,2 +3,0 @@\n",
		"-n\n",
		"-e\n",
		"\n",
	))

	cd, err := GetContextDiffString(ContextDiff(diff))
	assertEqual(t, err, nil)
	assertEqual(t, SplitLines(cd), stringsToBytes(
		"***************\n",
		"*** 0 ****\n",
		"--- 1,2 ----\n",
		"+ t\n",
		"+ w\n",
		"***************\n",
		"*** 2,3 ****\n",
		"- n\n",
		"- e\n",
		"--- 3 ----\n",
		"\n",
	))
}

func TestSplitLines(t *testing.T) {
	allTests := []struct {
		input string
		want  [][]byte
	}{
		{"foo", stringsToBytes("foo\n")},
		{"foo\nbar", stringsToBytes("foo\n", "bar\n")},
		{"foo\nbar\n", stringsToBytes("foo\n", "bar\n", "\n")},
	}
	for _, test := range allTests {
		assertEqual(t, SplitLines([]byte(test.input)), test.want)
	}
}

func benchmarkSplitLines(b *testing.B, count int) {
	str := bytes.Repeat([]byte("foo\n"), count)

	b.ResetTimer()

	n := 0
	for i := 0; i < b.N; i++ {
		n += len(SplitLines(str))
	}
}

func BenchmarkSplitLines100(b *testing.B) {
	benchmarkSplitLines(b, 100)
}

func BenchmarkSplitLines10000(b *testing.B) {
	benchmarkSplitLines(b, 10000)
}

func TestDifferCompare(t *testing.T) {
	diff := NewDiffer()
	// Test
	aLst := stringsToBytes("foo\n", "bar\n", "baz\n")
	bLst := stringsToBytes("foo\n", "bar1\n", "asdf\n", "baz\n")
	out, err := diff.Compare(aLst, bLst)
	if err != nil {
		t.Fatal("Differ Compare() error:", err)
	}
	out2 := bytesToStrings(out...)
	if reflect.DeepEqual(out2, []string{
		"  foo\n",
		"- bar\n",
		"+ bar1\n",
		"?    +\n",
		"+ asdf\n",
		"  baz\n",
	}) != true {
		t.Fatal("Differ Compare failure:", out2)
	}
}

func TestDifferStructuredDump(t *testing.T) {
	diff := NewDiffer()
	out := diff.StructuredDump('+',
		stringsToBytes("foo", "bar", "baz", "quux", "qwerty"),
		1, 3)
	expected := []DiffLine{DiffLine{'+', []byte("bar")},
							DiffLine{'+', []byte("baz")}}
	if !reflect.DeepEqual(out, expected) {
		t.Fatal("Differ StructuredDump failure:", out)
	}
}

func TestDifferDump(t *testing.T) {
	diff := NewDiffer()
	out := diff.Dump(PLUS,
		stringsToBytes("foo", "bar", "baz", "quux", "qwerty"),
		1, 3)
	out2 := bytesToStrings(out...)
	if reflect.DeepEqual(out2, []string{"+ bar", "+ baz"}) != true {
		t.Fatal("Differ Dump() failure:", out2)
	}
}

func TestDifferPlainReplace(t *testing.T) {
	diff := NewDiffer()
	aLst := stringsToBytes("one\n", "two\n", "three\n", "four\n", "five\n")
	bLst := stringsToBytes("one\n", "two2\n", "three\n", "extra\n")
	// Test a then b
	out, err := diff.PlainReplace(aLst, 1, 2, bLst, 1, 2)
	if err != nil {
		t.Fatal("Differ PlainReplace() error:", err)
	}
	out2 := bytesToStrings(out...)
	if reflect.DeepEqual(out2, []string{"- two\n", "+ two2\n"}) != true {
		t.Fatal("Differ PlainReplace() failure:", out2)
	}
	// Test b then a
	out, err = diff.PlainReplace(aLst, 3, 5, bLst, 3, 4)
	if err != nil {
		t.Fatal("Differ PlainReplace() error:", err)
	}
	out2 = bytesToStrings(out...)
	if reflect.DeepEqual(out2,
		[]string{"+ extra\n", "- four\n", "- five\n"}) != true {
		t.Fatal("Differ PlainReplace() failure:", out2)
	}
}

func TestDifferFancyReplaceAndHelper(t *testing.T) {
	diff := NewDiffer()
	// Test identical sync point, both full
	aLst := stringsToBytes("one\n", "asdf\n", "three\n")
	bLst := stringsToBytes("one\n", "two2\n", "three\n")
	out, err := diff.FancyReplace(aLst, 0, 3, bLst, 0, 3)
	if err != nil {
		t.Fatal("Differ FancyReplace() error:", err)
	}
	out2 := bytesToStrings(out...)
	if reflect.DeepEqual(out2,
		[]string{"  one\n", "- asdf\n", "+ two2\n", "  three\n"}) != true {
		t.Fatal("Differ FancyReplace() failure:", out2)
	}
	// Test close sync point, both full
	aLst = stringsToBytes("one\n", "two123456\n", "asdf\n", "three\n")
	bLst = stringsToBytes("one\n", "two123457\n", "qwerty\n", "three\n")
	out, err = diff.FancyReplace(aLst, 1, 3, bLst, 1, 3)
	if err != nil {
		t.Fatal("Differ FancyReplace() error:", err)
	}
	out2 = bytesToStrings(out...)
	if reflect.DeepEqual(out2, []string{
		"- two123456\n",
		"?         ^\n",
		"+ two123457\n",
		"?         ^\n",
		"- asdf\n",
		"+ qwerty\n",
	}) != true {
		t.Fatal("Differ FancyReplace() failure:", out2)
	}
	// Test no identical no close
	aLst = stringsToBytes("one\n", "asdf\n", "three\n")
	bLst = stringsToBytes("one\n", "qwerty\n", "three\n")
	out, err = diff.FancyReplace(aLst, 1, 2, bLst, 1, 2)
	if err != nil {
		t.Fatal("Differ FancyReplace() error:", err)
	}
	out2 = bytesToStrings(out...)
	if reflect.DeepEqual(out2, []string{
		"- asdf\n",
		"+ qwerty\n",
	}) != true {
		t.Fatal("Differ FancyReplace() failure:", out2)
	}
}

func TestDifferQFormat(t *testing.T) {
	diff := NewDiffer()
	aStr := []byte("\tfoo2bar\n")
	aTag := []byte("    ^  ^")
	bStr := []byte("\tfoo3baz\n")
	bTag := []byte("    ^  ^")
	out := diff.QFormat(aStr, bStr, aTag, bTag)
	out2 := bytesToStrings(out...)
	if reflect.DeepEqual(out2, []string{
		"- \tfoo2bar\n",
		"? \t   ^  ^\n",
		"+ \tfoo3baz\n",
		"? \t   ^  ^\n",
	}) != true {
		t.Fatal("Differ QFormat() failure:", out2)
	}
}

func TestGetUnifiedDiffString(t *testing.T) {
	A := []byte("one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\nten\n")
	B := []byte("one\ntwo\nthr33\nfour\nfive\nsix\nseven\neight\nnine\nten\n")
	// Build diff
	diff := UnifiedDiff{A: SplitLines(A),
		FromFile: "file", FromDate: "then",
		B: SplitLines(B),
		ToFile: "tile", ToDate: "now", Eol: []byte{}, Context: 1}
	// Run test
	diffStr, err := GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatal("GetUnifiedDiffString error:", err)
	}
	exp := []byte("--- file\tthen\n+++ tile\tnow\n@@ -2,3 +2,3 @@\n two\n-three\n+thr33\n four\n")
	if !bytes.Equal(diffStr, exp) {
		t.Fatal("GetUnifiedDiffString failure:", diffStr)
	}
}

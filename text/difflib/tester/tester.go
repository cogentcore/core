package tester

import (
	"math/rand"
	"time"
)

func prepareStrings(seed int64) (A, B []string) {
	if seed == -1 {
		seed = time.Now().UnixNano()
	}
	rand.Seed(seed)
	// Generate 4000 random lines
	lines := [4000]string{}
	for i := range lines {
		l := rand.Intn(100)
		p := make([]byte, l)
		rand.Read(p)
		lines[i] = string(p)
	}
	// Generate two 4000 lines documents by picking some lines at random
	A = make([]string, 4000)
	B = make([]string, len(A))
	for i := range A {
		// make the first 50 lines more likely to appear
		if rand.Intn(100) < 40 {
			A[i] = lines[rand.Intn(50)]
		} else {
			A[i] = lines[rand.Intn(len(lines))]
		}
		if rand.Intn(100) < 40 {
			B[i] = lines[rand.Intn(50)]
		} else {
			B[i] = lines[rand.Intn(len(lines))]
		}
	}
	// Do some copies from A to B
	maxcopy := rand.Intn(len(A)-1) + 1
	for copied, tocopy := 0, rand.Intn(2*len(A)/3); copied < tocopy; {
		l := rand.Intn(rand.Intn(maxcopy-1) + 1)
		for a, b, n := rand.Intn(len(A)), rand.Intn(len(B)), 0; a < len(A) && b < len(B) && n < l; a, b, n = a+1, b+1, n+1 {
			B[b] = A[a]
			copied++
		}
	}
	// And some from B to A
	for copied, tocopy := 0, rand.Intn(2*len(A)/3); copied < tocopy; {
		l := rand.Intn(rand.Intn(maxcopy-1) + 1)
		for a, b, n := rand.Intn(len(A)), rand.Intn(len(B)), 0; a < len(A) && b < len(B) && n < l; a, b, n = a+1, b+1, n+1 {
			A[a] = B[b]
			copied++
		}
	}
	return
}

func PrepareStringsToDiff(count, seed int) (As, Bs [][]string) {
	As = make([][]string, count)
	Bs = make([][]string, count)
	for i := range As {
		As[i], Bs[i] = prepareStrings(int64(i + seed))
	}
	return
}

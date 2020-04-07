// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// this code is largely verbatim copied from: https://github.com/sajari/fuzzy
// https://www.sajari.com/
// Most of which seems to have been written by Hamish @sajari

// it does not have a copyright notice in the code itself but does have
// an MIT license file.

// it has not been modified in 5 years, and the file format is very inefficient..
// and also hard to edit if you happen to save a mistake..

package spell

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"index/suffixarray"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	SpellDepthDefault              = 2
	SpellThresholdDefault          = 5
	SuffDivergenceThresholdDefault = 100
)

type Pair struct {
	str1 string
	str2 string
}

type Method int

const (
	MethodIsWord                   Method = 0
	MethodSuggestMapsToInput              = 1
	MethodInputDeleteMapsToDict           = 2
	MethodInputDeleteMapsToSuggest        = 3
)

// Potential is a potential match
type Potential struct {
	Term   string // Potential term string
	Score  int    // Score
	Leven  int    // Levenstein distance from the suggestion to the input
	Method Method // How this potential was matched
}

// Counts has the individual word counts
type Counts struct {
	Corpus int `json:"c"`
	Query  int `json:"q"`
}

// Model is the full data model
type Model struct {
	Data                    map[string]*Counts  `json:"data"`
	Maxcount                int                 `json:"maxcount"`
	Suggest                 map[string][]string `json:"suggest"`
	Depth                   int                 `json:"depth"`
	Threshold               int                 `json:"threshold"`
	UseAutocomplete         bool                `json:"autocomplete"`
	SuffDivergence          int                 `json:"-"`
	SuffDivergenceThreshold int                 `json:"suff_threshold"`
	SuffixArr               *suffixarray.Index  `json:"-"`
	SuffixArrConcat         string              `json:"-"`
	sync.RWMutex
}

// For sorting autocomplete suggestions
// to bias the most popular first
type Autos struct {
	Results []string
	Model   *Model
}

func (a Autos) Len() int      { return len(a.Results) }
func (a Autos) Swap(i, j int) { a.Results[i], a.Results[j] = a.Results[j], a.Results[i] }

func (a Autos) Less(i, j int) bool {
	icc := a.Model.Data[a.Results[i]].Corpus
	jcc := a.Model.Data[a.Results[j]].Corpus
	icq := a.Model.Data[a.Results[i]].Query
	jcq := a.Model.Data[a.Results[j]].Query
	if icq == jcq {
		if icc == jcc {
			return a.Results[i] > a.Results[j]
		}
		return icc > jcc
	}
	return icq > jcq
}

func (m Method) String() string {
	switch m {
	case MethodIsWord:
		return "Input in dictionary"
	case MethodSuggestMapsToInput:
		return "Suggest maps to input"
	case MethodInputDeleteMapsToDict:
		return "Input delete maps to dictionary"
	case MethodInputDeleteMapsToSuggest:
		return "Input delete maps to suggest key"
	}
	return "unknown"
}

func (pot *Potential) String() string {
	return fmt.Sprintf("Term: %v\n\tScore: %v\n\tLeven: %v\n\tMethod: %v\n\n", pot.Term, pot.Score, pot.Leven, pot.Method)
}

// Create and initialise a new model
func NewModel() *Model {
	md := new(Model)
	return md.Init()
}

func (md *Model) Init() *Model {
	md.Data = make(map[string]*Counts)
	md.Suggest = make(map[string][]string)
	md.Depth = SpellDepthDefault
	md.Threshold = SpellThresholdDefault // Setting this to 1 is most accurate, but "1" is 5x more memory and 30x slower processing than "4". This is a big performance tuning knob
	md.UseAutocomplete = true            // Default is to include Autocomplete
	md.updateSuffixArr()
	md.SuffDivergenceThreshold = SuffDivergenceThresholdDefault
	return md
}

// WriteTo writes a model to a Writer
func (md *Model) WriteTo(w io.Writer) error {
	md.RLock()
	defer md.RUnlock()

	nline := []byte("\n")
	comma := []byte(",")

	_, err := w.Write([]byte(`{"data": {`))
	if err != nil {
		return err
	}
	w.Write(nline)

	n := len(md.Data)
	keys := make([]string, n)
	idx := 0
	for k := range md.Data {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)
	for i, k := range keys {
		c := md.Data[k]
		w.Write([]byte(fmt.Sprintf(`%q:{"c":%d,"q":%d}`, k, c.Corpus, c.Query)))
		if i < n-1 {
			w.Write(comma)
		}
		_, err = w.Write(nline)
		if err != nil {
			return err
		}
	}
	w.Write([]byte("},\n"))
	w.Write([]byte(fmt.Sprintf(`"maxcount": %d`, md.Maxcount)))
	w.Write([]byte(",\n"))
	w.Write([]byte(`"suggest": {`))
	w.Write([]byte(nline))

	n = len(md.Suggest)
	keys = make([]string, n)
	idx = 0
	for k := range md.Suggest {
		keys[idx] = k
		idx++
	}
	sort.Strings(keys)
	for i, k := range keys {
		s := md.Suggest[k]
		_, err = w.Write([]byte(fmt.Sprintf(`%q:[`, k)))
		if err != nil {
			return err
		}
		ns := len(s)
		for j, g := range s {
			w.Write([]byte(fmt.Sprintf("%q", g)))
			if j < ns-1 {
				w.Write(comma)
			}
		}
		w.Write([]byte("]"))
		if i < n-1 {
			w.Write(comma)
		}
		w.Write(nline)
	}
	_, err = w.Write([]byte("},\n"))
	if err != nil {
		return err
	}
	w.Write([]byte(fmt.Sprintf(`"depth":%d,`, md.Depth)))
	w.Write(nline)
	w.Write([]byte(fmt.Sprintf(`"threshold":%d,`, md.Threshold)))
	w.Write(nline)
	w.Write([]byte(fmt.Sprintf(`"autocomplete":%t,`, md.UseAutocomplete)))
	w.Write(nline)
	w.Write([]byte(fmt.Sprintf(`"suff_threshold":%d`, md.SuffDivergenceThreshold)))
	_, err = w.Write([]byte("\n}\n"))
	if err != nil {
		return err
	}

	return nil
}

// Save a spelling model to disk
func (md *Model) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		log.Println("Fuzzy model:", err)
		return err
	}
	defer f.Close()
	fb := bufio.NewWriter(f) // this makes a HUGE difference in write performance!
	defer fb.Flush()
	err = md.WriteTo(fb)
	if err != nil {
		log.Println("Fuzzy model:", err)
		return err
	}
	return nil
}

// Save a spelling model to disk, but discard all
// entries less than the threshold number of occurences
// Much smaller and all that is used when generated
// as a once off, but not useful for incremental usage
func (md *Model) SaveLight(filename string) error {
	md.Lock()
	for term, count := range md.Data {
		if count.Corpus < md.Threshold {
			delete(md.Data, term)
		}
	}
	md.Unlock()
	return md.Save(filename)
}

// FromReader loads a model from a Reader
func FromReader(r io.Reader) (*Model, error) {
	md := new(Model)
	d := json.NewDecoder(r)
	err := d.Decode(md)
	if err != nil {
		return nil, err
	}
	md.updateSuffixArr()
	return md, nil
}

// Load a saved model from disk
func Load(filename string) (*Model, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	md, err := FromReader(f)
	if err != nil {
		return nil, err
	}
	return md, nil
}

// Change the default depth value of the model. This sets how many
// character differences are indexed. The default is 2.
func (md *Model) SetDepth(val int) {
	md.Lock()
	md.Depth = val
	md.Unlock()
}

// Change the default threshold of the model. This is how many times
// a term must be seen before suggestions are created for it
func (md *Model) SetThreshold(val int) {
	md.Lock()
	md.Threshold = val
	md.Unlock()
}

// Optionally disabled suffixarray based autocomplete support
func (md *Model) SetUseAutocomplete(val bool) {
	md.Lock()
	old := md.UseAutocomplete
	md.Unlock()
	md.UseAutocomplete = val
	if !old && val {
		md.updateSuffixArr()
	}
}

// Optionally set the suffix array divergence threshold. This is
// the number of query training steps between rebuilds of the
// suffix array. A low number will be more accurate but will use
// resources and create more garbage.
func (md *Model) SetDivergenceThreshold(val int) {
	md.Lock()
	md.SuffDivergenceThreshold = val
	md.Unlock()
}

// Calculate the Levenshtein distance between two strings
func Levenshtein(a, b *string) int {
	la := len(*a)
	lb := len(*b)
	d := make([]int, la+1)
	var lastdiag, olddiag, temp int

	for i := 1; i <= la; i++ {
		d[i] = i
	}
	for i := 1; i <= lb; i++ {
		d[0] = i
		lastdiag = i - 1
		for j := 1; j <= la; j++ {
			olddiag = d[j]
			min := d[j] + 1
			if (d[j-1] + 1) < min {
				min = d[j-1] + 1
			}
			if (*a)[j-1] == (*b)[i-1] {
				temp = 0
			} else {
				temp = 1
			}
			if (lastdiag + temp) < min {
				min = lastdiag + temp
			}
			d[j] = min
			lastdiag = olddiag
		}
	}
	return d[la]
}

// Add an array of words to train the model in bulk
func (md *Model) Train(terms []string) {
	for _, term := range terms {
		md.TrainWord(term)
	}
	md.updateSuffixArr()
}

// Manually set the count of a word. Optionally trigger the
// creation of suggestion keys for the term. This function lets
// you build a model from an existing dictionary with word popularity
// counts without needing to run "TrainWord" repeatedly
func (md *Model) SetCount(term string, count int, suggest bool) {
	md.Lock()
	md.Data[term] = &Counts{count, 0} // Note: This may reset a query count? TODO
	if suggest {
		md.createSuggestKeys(term)
	}
	md.Unlock()
}

// Train the model word by word. This is corpus training as opposed
// to query training. Word counts from this type of training are not
// likely to correlate with those of search queries
func (md *Model) TrainWord(term string) {
	md.Lock()
	if t, ok := md.Data[term]; ok {
		t.Corpus++
	} else {
		md.Data[term] = &Counts{1, 0}
	}
	// Set the max
	if md.Data[term].Corpus > md.Maxcount {
		md.Maxcount = md.Data[term].Corpus
		md.SuffDivergence++
	}
	// If threshold is triggered, store delete suggestion keys
	if md.Data[term].Corpus == md.Threshold {
		md.createSuggestKeys(term)
	}
	md.Unlock()
}

// Train using a search query term. This builds a second popularity
// index of terms used to search, as opposed to generally occurring
// in corpus text
func (md *Model) TrainQuery(term string) {
	md.Lock()
	if t, ok := md.Data[term]; ok {
		t.Query++
	} else {
		md.Data[term] = &Counts{0, 1}
	}
	md.SuffDivergence++
	update := md.SuffDivergence > md.SuffDivergenceThreshold
	md.Unlock()
	if update {
		md.updateSuffixArr()
	}
}

// For a given term, create the partially deleted lookup keys
func (md *Model) createSuggestKeys(term string) {
	edits := md.EditsMulti(term, md.Depth)
	for _, edit := range edits {
		skip := false
		for _, hit := range md.Suggest[edit] {
			if hit == term {
				// Already know about this one
				skip = true
				continue
			}
		}
		if !skip && len(edit) > 1 {
			md.Suggest[edit] = append(md.Suggest[edit], term)
		}
	}
}

// Edits at any depth for a given term. The depth of the model is used
func (md *Model) EditsMulti(term string, depth int) []string {
	edits := Edits1(term)
	for {
		depth--
		if depth <= 0 {
			break
		}
		for _, edit := range edits {
			edits2 := Edits1(edit)
			for _, edit2 := range edits2 {
				edits = append(edits, edit2)
			}
		}
	}
	return edits
}

// Edits1 creates a set of terms that are 1 char delete from the input term
func Edits1(word string) []string {

	splits := []Pair{}
	for i := 0; i <= len(word); i++ {
		splits = append(splits, Pair{word[:i], word[i:]})
	}

	total_set := []string{}
	for _, elem := range splits {

		//deletion
		if len(elem.str2) > 0 {
			total_set = append(total_set, elem.str1+elem.str2[1:])
		} else {
			total_set = append(total_set, elem.str1)
		}

	}

	// Special case ending in "ies" or "ys"
	if strings.HasSuffix(word, "ies") {
		total_set = append(total_set, word[:len(word)-3]+"ys")
	}
	if strings.HasSuffix(word, "ys") {
		total_set = append(total_set, word[:len(word)-2]+"ies")
	}

	return total_set
}

func (md *Model) corpusCount(input string) int {
	if score, ok := md.Data[input]; ok {
		return score.Corpus
	}
	return 0
}

// From a group of potentials, work out the most likely result
func best(input string, potential map[string]*Potential) string {
	var best string
	var bestcalc, bonus int
	for i := 0; i < 4; i++ {
		for _, pot := range potential {
			if pot.Leven == 0 {
				return pot.Term
			} else if pot.Leven == i {
				bonus = 0
				// If the first letter is the same, that's a good sign. Bias these potentials
				if pot.Term[0] == input[0] {
					bonus += 100
				}
				if pot.Score+bonus > bestcalc {
					bestcalc = pot.Score + bonus
					best = pot.Term
				}
			}
		}
		if bestcalc > 0 {
			return best
		}
	}
	return best
}

// From a group of potentials, work out the most likely results, in order of
// best to worst
func bestn(input string, potential map[string]*Potential, n int) []string {
	var output []string
	for i := 0; i < n; i++ {
		if len(potential) == 0 {
			break
		}
		b := best(input, potential)
		output = append(output, b)
		delete(potential, b)
	}
	return output
}

// Test an input, if we get it wrong, look at why it is wrong. This
// function returns a bool indicating if the guess was correct as well
// as the term it is suggesting. Typically this function would be used
// for testing, not for production
func (md *Model) CheckKnown(input string, correct string) bool {
	md.RLock()
	defer md.RUnlock()
	suggestions := md.suggestPotential(input, true)
	best := best(input, suggestions)
	if best == correct {
		// This guess is correct
		fmt.Printf("Input correctly maps to correct term")
		return true
	}
	if pot, ok := suggestions[correct]; !ok {

		if md.corpusCount(correct) > 0 {
			fmt.Printf("\"%v\" - %v (%v) not in the suggestions. (%v) best option.\n", input, correct, md.corpusCount(correct), best)
			for _, sugg := range suggestions {
				fmt.Printf("	%v\n", sugg)
			}
		} else {
			fmt.Printf("\"%v\" - Not in dictionary\n", correct)
		}
	} else {
		fmt.Printf("\"%v\" - (%v) suggested, should however be (%v).\n", input, suggestions[best], pot)
	}
	return false
}

// For a given input term, suggest some alternatives. If exhaustive, each of the 4
// cascading checks will be performed and all potentials will be sorted accordingly
func (md *Model) suggestPotential(input string, exhaustive bool) map[string]*Potential {
	input = strings.ToLower(input)
	suggestions := make(map[string]*Potential, 20)

	// 0 - If this is a dictionary term we're all good, no need to go further
	if md.corpusCount(input) > md.Threshold {
		suggestions[input] = &Potential{Term: input, Score: md.corpusCount(input), Leven: 0, Method: MethodIsWord}
		if !exhaustive {
			return suggestions
		}
	}

	// 1 - See if the input matches a "suggest" key
	if sugg, ok := md.Suggest[input]; ok {
		for _, pot := range sugg {
			if _, ok := suggestions[pot]; !ok {
				suggestions[pot] = &Potential{Term: pot, Score: md.corpusCount(pot), Leven: Levenshtein(&input, &pot), Method: MethodSuggestMapsToInput}
			}
		}

		if !exhaustive {
			return suggestions
		}
	}

	// 2 - See if edit1 matches input
	max := 0
	edits := md.EditsMulti(input, md.Depth)
	for _, edit := range edits {
		score := md.corpusCount(edit)
		if score > 0 && len(edit) > 2 {
			if _, ok := suggestions[edit]; !ok {
				suggestions[edit] = &Potential{Term: edit, Score: score, Leven: Levenshtein(&input, &edit), Method: MethodInputDeleteMapsToDict}
			}
			if score > max {
				max = score
			}
		}
	}
	if max > 0 {
		if !exhaustive {
			return suggestions
		}
	}

	// 3 - No hits on edit1 distance, look for transposes and replaces
	// Note: these are more complex, we need to check the guesses
	// more thoroughly, e.g. levals=[valves] in a raw sense, which
	// is incorrect
	for _, edit := range edits {
		if sugg, ok := md.Suggest[edit]; ok {
			// Is this a real transpose or replace?
			for _, pot := range sugg {
				lev := Levenshtein(&input, &pot)
				if lev <= md.Depth+1 { // The +1 doesn't seem to impact speed, but has greater coverage when the depth is not sufficient to make suggestions
					if _, ok := suggestions[pot]; !ok {
						suggestions[pot] = &Potential{Term: pot, Score: md.corpusCount(pot), Leven: lev, Method: MethodInputDeleteMapsToSuggest}
					}
				}
			}
		}
	}
	return suggestions
}

// Return the raw potential terms so they can be ranked externally
// to this package
func (md *Model) Potentials(input string, exhaustive bool) map[string]*Potential {
	md.RLock()
	defer md.RUnlock()
	return md.suggestPotential(input, exhaustive)
}

// For a given input string, suggests potential replacements
func (md *Model) Suggestions(input string, exhaustive bool) []string {
	md.RLock()
	suggestions := md.suggestPotential(input, exhaustive)
	md.RUnlock()
	output := make([]string, 0, 10)
	for _, suggestion := range suggestions {
		output = append(output, suggestion.Term)
	}
	return output
}

// Return the most likely correction for the input term
func (md *Model) SpellCheck(input string) string {
	md.RLock()
	suggestions := md.suggestPotential(input, false)
	md.RUnlock()
	return best(input, suggestions)
}

// Return the most likely corrections in order from best to worst
func (md *Model) SpellCheckSuggestions(input string, n int) []string {
	md.RLock()
	suggestions := md.suggestPotential(input, true)
	md.RUnlock()
	return bestn(input, suggestions, n)
}

func SampleEnglish() []string {
	var out []string
	file, err := os.Open("data/big.txt")
	if err != nil {
		fmt.Println(err)
		return out
	}
	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	// Count the words.
	count := 0
	for scanner.Scan() {
		exp, _ := regexp.Compile("[a-zA-Z]+")
		words := exp.FindAll([]byte(scanner.Text()), -1)
		for _, word := range words {
			if len(word) > 1 {
				out = append(out, strings.ToLower(string(word)))
				count++
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading input:", err)
	}

	return out
}

// Takes the known dictionary listing and creates a suffix array
// model for these terms. If a model already existed, it is discarded
func (md *Model) updateSuffixArr() {
	if !md.UseAutocomplete {
		return
	}
	md.RLock()
	termArr := make([]string, 0, 1000)
	for term, count := range md.Data {
		if count.Corpus > md.Threshold || count.Query > 0 { // TODO: query threshold?
			termArr = append(termArr, term)
		}
	}
	md.SuffixArrConcat = "\x00" + strings.Join(termArr, "\x00") + "\x00"
	md.SuffixArr = suffixarray.New([]byte(md.SuffixArrConcat))
	md.SuffDivergence = 0
	md.RUnlock()
}

// For a given string, autocomplete using the suffix array model
func (md *Model) Autocomplete(input string) ([]string, error) {
	md.RLock()
	defer md.RUnlock()
	if !md.UseAutocomplete {
		return []string{}, errors.New("Autocomplete is disabled")
	}
	if len(input) == 0 {
		return []string{}, errors.New("Input cannot have length zero")
	}
	express := "\x00" + input + "[^\x00]*"
	match, err := regexp.Compile(express)
	if err != nil {
		return []string{}, err
	}
	matches := md.SuffixArr.FindAllIndex(match, -1)
	a := &Autos{Results: make([]string, 0, len(matches)), Model: md}
	for _, m := range matches {
		str := strings.Trim(md.SuffixArrConcat[m[0]:m[1]], "\x00")
		if count, ok := md.Data[str]; ok {
			if count.Corpus > md.Threshold || count.Query > 0 {
				a.Results = append(a.Results, str)
			}
		}
	}
	sort.Sort(a)
	if len(a.Results) >= 10 {
		return a.Results[:10], nil
	}
	return a.Results, nil
}

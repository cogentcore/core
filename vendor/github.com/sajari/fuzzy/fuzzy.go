package fuzzy

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

type Potential struct {
	Term   string // Potential term string
	Score  int    // Score
	Leven  int    // Levenstein distance from the suggestion to the input
	Method Method // How this potential was matched
}

type Counts struct {
	Corpus int `json:"corpus"`
	Query  int `json:"query"`
}

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
	model := new(Model)
	return model.Init()
}

func (model *Model) Init() *Model {
	model.Data = make(map[string]*Counts)
	model.Suggest = make(map[string][]string)
	model.Depth = SpellDepthDefault
	model.Threshold = SpellThresholdDefault // Setting this to 1 is most accurate, but "1" is 5x more memory and 30x slower processing than "4". This is a big performance tuning knob
	model.UseAutocomplete = true            // Default is to include Autocomplete
	model.updateSuffixArr()
	model.SuffDivergenceThreshold = SuffDivergenceThresholdDefault
	return model
}

// WriteTo writes a model to a Writer
func (model *Model) WriteTo(w io.Writer) (int64, error) {
	model.RLock()
	defer model.RUnlock()
	b, err := json.Marshal(model)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(b)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}

// Save a spelling model to disk
func (model *Model) Save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		log.Println("Fuzzy model:", err)
		return err
	}
	defer f.Close()
	_, err = model.WriteTo(f)
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
func (model *Model) SaveLight(filename string) error {
	model.Lock()
	for term, count := range model.Data {
		if count.Corpus < model.Threshold {
			delete(model.Data, term)
		}
	}
	model.Unlock()
	return model.Save(filename)
}

// FromReader loads a model from a Reader
func FromReader(r io.Reader) (*Model, error) {
	model := new(Model)
	d := json.NewDecoder(r)
	err := d.Decode(model)
	if err != nil {
		return nil, err
	}
	model.updateSuffixArr()
	return model, nil
}

// Load a saved model from disk
func Load(filename string) (*Model, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	model, err := FromReader(f)
	if err != nil {
		model = new(Model)
		if err1 := model.convertOldFormat(filename); err1 != nil {
			return model, err1
		}
		return model, nil
	}
	return model, nil
}

// Change the default depth value of the model. This sets how many
// character differences are indexed. The default is 2.
func (model *Model) SetDepth(val int) {
	model.Lock()
	model.Depth = val
	model.Unlock()
}

// Change the default threshold of the model. This is how many times
// a term must be seen before suggestions are created for it
func (model *Model) SetThreshold(val int) {
	model.Lock()
	model.Threshold = val
	model.Unlock()
}

// Optionally disabled suffixarray based autocomplete support
func (model *Model) SetUseAutocomplete(val bool) {
	model.Lock()
	old := model.UseAutocomplete
	model.Unlock()
	model.UseAutocomplete = val
	if !old && val {
		model.updateSuffixArr()
	}
}

// Optionally set the suffix array divergence threshold. This is
// the number of query training steps between rebuilds of the
// suffix array. A low number will be more accurate but will use
// resources and create more garbage.
func (model *Model) SetDivergenceThreshold(val int) {
	model.Lock()
	model.SuffDivergenceThreshold = val
	model.Unlock()
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
func (model *Model) Train(terms []string) {
	for _, term := range terms {
		model.TrainWord(term)
	}
	model.updateSuffixArr()
}

// Manually set the count of a word. Optionally trigger the
// creation of suggestion keys for the term. This function lets
// you build a model from an existing dictionary with word popularity
// counts without needing to run "TrainWord" repeatedly
func (model *Model) SetCount(term string, count int, suggest bool) {
	model.Lock()
	model.Data[term] = &Counts{count, 0} // Note: This may reset a query count? TODO
	if suggest {
		model.createSuggestKeys(term)
	}
	model.Unlock()
}

// Train the model word by word. This is corpus training as opposed
// to query training. Word counts from this type of training are not
// likely to correlate with those of search queries
func (model *Model) TrainWord(term string) {
	model.Lock()
	if t, ok := model.Data[term]; ok {
		t.Corpus++
	} else {
		model.Data[term] = &Counts{1, 0}
	}
	// Set the max
	if model.Data[term].Corpus > model.Maxcount {
		model.Maxcount = model.Data[term].Corpus
		model.SuffDivergence++
	}
	// If threshold is triggered, store delete suggestion keys
	if model.Data[term].Corpus == model.Threshold {
		model.createSuggestKeys(term)
	}
	model.Unlock()
}

// Train using a search query term. This builds a second popularity
// index of terms used to search, as opposed to generally occurring
// in corpus text
func (model *Model) TrainQuery(term string) {
	model.Lock()
	if t, ok := model.Data[term]; ok {
		t.Query++
	} else {
		model.Data[term] = &Counts{0, 1}
	}
	model.SuffDivergence++
	update := model.SuffDivergence > model.SuffDivergenceThreshold
	model.Unlock()
	if update {
		model.updateSuffixArr()
	}
}

// For a given term, create the partially deleted lookup keys
func (model *Model) createSuggestKeys(term string) {
	edits := model.EditsMulti(term, model.Depth)
	for _, edit := range edits {
		skip := false
		for _, hit := range model.Suggest[edit] {
			if hit == term {
				// Already know about this one
				skip = true
				continue
			}
		}
		if !skip && len(edit) > 1 {
			model.Suggest[edit] = append(model.Suggest[edit], term)
		}
	}
}

// Edits at any depth for a given term. The depth of the model is used
func (model *Model) EditsMulti(term string, depth int) []string {
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

func (model *Model) corpusCount(input string) int {
	if score, ok := model.Data[input]; ok {
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
func (model *Model) CheckKnown(input string, correct string) bool {
	model.RLock()
	defer model.RUnlock()
	suggestions := model.suggestPotential(input, true)
	best := best(input, suggestions)
	if best == correct {
		// This guess is correct
		fmt.Printf("Input correctly maps to correct term")
		return true
	}
	if pot, ok := suggestions[correct]; !ok {

		if model.corpusCount(correct) > 0 {
			fmt.Printf("\"%v\" - %v (%v) not in the suggestions. (%v) best option.\n", input, correct, model.corpusCount(correct), best)
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
func (model *Model) suggestPotential(input string, exhaustive bool) map[string]*Potential {
	input = strings.ToLower(input)
	suggestions := make(map[string]*Potential, 20)

	// 0 - If this is a dictionary term we're all good, no need to go further
	if model.corpusCount(input) > model.Threshold {
		suggestions[input] = &Potential{Term: input, Score: model.corpusCount(input), Leven: 0, Method: MethodIsWord}
		if !exhaustive {
			return suggestions
		}
	}

	// 1 - See if the input matches a "suggest" key
	if sugg, ok := model.Suggest[input]; ok {
		for _, pot := range sugg {
			if _, ok := suggestions[pot]; !ok {
				suggestions[pot] = &Potential{Term: pot, Score: model.corpusCount(pot), Leven: Levenshtein(&input, &pot), Method: MethodSuggestMapsToInput}
			}
		}

		if !exhaustive {
			return suggestions
		}
	}

	// 2 - See if edit1 matches input
	max := 0
	edits := model.EditsMulti(input, model.Depth)
	for _, edit := range edits {
		score := model.corpusCount(edit)
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
		if sugg, ok := model.Suggest[edit]; ok {
			// Is this a real transpose or replace?
			for _, pot := range sugg {
				lev := Levenshtein(&input, &pot)
				if lev <= model.Depth+1 { // The +1 doesn't seem to impact speed, but has greater coverage when the depth is not sufficient to make suggestions
					if _, ok := suggestions[pot]; !ok {
						suggestions[pot] = &Potential{Term: pot, Score: model.corpusCount(pot), Leven: lev, Method: MethodInputDeleteMapsToSuggest}
					}
				}
			}
		}
	}
	return suggestions
}

// Return the raw potential terms so they can be ranked externally
// to this package
func (model *Model) Potentials(input string, exhaustive bool) map[string]*Potential {
	model.RLock()
	defer model.RUnlock()
	return model.suggestPotential(input, exhaustive)
}

// For a given input string, suggests potential replacements
func (model *Model) Suggestions(input string, exhaustive bool) []string {
	model.RLock()
	suggestions := model.suggestPotential(input, exhaustive)
	model.RUnlock()
	output := make([]string, 0, 10)
	for _, suggestion := range suggestions {
		output = append(output, suggestion.Term)
	}
	return output
}

// Return the most likely correction for the input term
func (model *Model) SpellCheck(input string) string {
	model.RLock()
	suggestions := model.suggestPotential(input, false)
	model.RUnlock()
	return best(input, suggestions)
}

// Return the most likely corrections in order from best to worst
func (model *Model) SpellCheckSuggestions(input string, n int) []string {
	model.RLock()
	suggestions := model.suggestPotential(input, true)
	model.RUnlock()
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
func (model *Model) updateSuffixArr() {
	if !model.UseAutocomplete {
		return
	}
	model.RLock()
	termArr := make([]string, 0, 1000)
	for term, count := range model.Data {
		if count.Corpus > model.Threshold || count.Query > 0 { // TODO: query threshold?
			termArr = append(termArr, term)
		}
	}
	model.SuffixArrConcat = "\x00" + strings.Join(termArr, "\x00") + "\x00"
	model.SuffixArr = suffixarray.New([]byte(model.SuffixArrConcat))
	model.SuffDivergence = 0
	model.RUnlock()
}

// For a given string, autocomplete using the suffix array model
func (model *Model) Autocomplete(input string) ([]string, error) {
	model.RLock()
	defer model.RUnlock()
	if !model.UseAutocomplete {
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
	matches := model.SuffixArr.FindAllIndex(match, -1)
	a := &Autos{Results: make([]string, 0, len(matches)), Model: model}
	for _, m := range matches {
		str := strings.Trim(model.SuffixArrConcat[m[0]:m[1]], "\x00")
		if count, ok := model.Data[str]; ok {
			if count.Corpus > model.Threshold || count.Query > 0 {
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

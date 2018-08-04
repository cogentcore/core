// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"go/parser"
	"fmt"
	"unicode"
	"strings"
)

// Completer is an implementation of completion for debugging - there will be a real code completer
// that understands golang
////////////////////////////////////////////////////////////////////////////////////////

// Completer is an interface to be implemented for code completion.
type Completer interface {
	// GenCompletions generates a string array of possible completions based on seed and context
	GenCompletions(text string)

	// Count returns the number of possible completions
	Count() int

	// Seed returns the current seed
	Seed()

	// Extend tries to extend the current seed checking possible completions for a longer common seed
	// e.g. if the current seed is "ab" and the completions are "abcde" and "abcdf" then Extend returns "cd"
	// but if the possible completions are "abcde" and "abz" then Extend returns ""
	Extend() string
}

////////////////////////////////////////////////////////////////////////////////////////
// SampleCompleter
type SampleCompleter struct {
	inited		bool
	seed        string // the portion of the text from which completions are generated
	completions []string
	matches		[]string
}

// GenCompletions generates a list of possible completions and returns the number found.
func (sc *SampleCompleter) GenCompletions(text string) {
	if sc.inited == false {
		sc.Init()
	}
	sc.matches = sc.completions[0:0]

	seedStart := 0
	for i := len(text) - 1; i >= 0; i-- {
		r := rune(text[i])
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			seedStart = i + 1
			break
		}
	}
	sc.seed = text[seedStart:]

	match_start := -1
	match_end := -1
	if len(sc.seed) > 0 {
		for i, s := range sc.completions {
			if match_end > -1 {
				break
			}
			if match_start == -1 {
				if strings.HasPrefix(s, sc.seed) {
					match_start = i // first match in sorted list
				}
				continue
			}
			if (match_start > -1) {
				if strings.HasPrefix(s, sc.seed) == false {
					match_end = i
				}
			}
		}
	}

	//fmt.Printf("match start: %d, match_end: %d", match_start, match_end)
	if (match_start > -1 && match_end > -1) {
		sc.matches = sc.completions[match_start:match_end]
	}
}

func (sc *SampleCompleter) Count() int {
	return len(sc.matches)
}

func (sc *SampleCompleter) Seed() string {
	return sc.seed
}

func (sc *SampleCompleter) Extend() string {
	keep_looking := true
	new_seed := sc.seed
	potential_seed := new_seed
	first_match := sc.matches[0]
	for keep_looking {
		if len(first_match) <= len(new_seed) {
			keep_looking = false // ran out of chars
			break
		}

		potential_seed = first_match[0:len(new_seed) + 1]
		for _, s := range sc.matches {
			if !strings.HasPrefix(s, potential_seed) {
				keep_looking = false;
				break;
			}
		}
		if keep_looking {
			new_seed = potential_seed
		}
	}
	return strings.Replace(new_seed, sc.seed, "", 1) // only return the seed extension
}

// Init being used to load words for this sample completer
func (sc *SampleCompleter) Init() {
	sc.completions = []string{"a", "able", "about", "above", "act", "add", "afraid", "after", "again", "against", "age", "ago", "agree", "air", "all",
		"allow", "also", "always", "am", "among", "an", "and", "anger", "animal", "answer", "any", "appear", "apple", "are",
		"area", "arm", "arrange", "arrive", "art", "as", "ask", "at", "atom", "baby", "back", "bad", "ball", "band", "bank",
		"bar", "base", "basic", "bat", "be", "bear", "beat", "beauty", "bed", "been", "before", "began", "begin", "behind",
		"believe", "bell", "best", "better", "between", "big", "bird", "bit", "black", "block", "blood", "blow", "blue",
		"board", "boat", "body", "bone", "book", "born", "both", "bottom", "bought", "box", "boy", "branch", "bread", "break",
		"bright", "bring", "broad", "broke", "brother", "brought", "brown", "build", "burn", "busy", "but", "buy", "by", "call",
		"came", "camp", "can", "capital", "captain", "car", "card", "care", "carry", "case", "cat", "catch", "caught", "cause",
		"cell", "cent", "center", "century", "certain", "chair", "chance", "change", "character", "charge", "chart", "check",
		"chick", "chief", "child", "children", "choose", "chord", "circle", "city", "claim", "class", "clean", "clear", "climb",
		"clock", "close", "clothe", "cloud", "coast", "coat", "cold", "collect", "colony", "color", "column", "come", "common",
		"company", "compare", "complete", "condition", "connect", "consider", "consonant", "contain", "continent", "continue",
		"control", "cook", "cool", "copy", "corn", "corner", "correct", "cost", "cotton", "could", "count", "country", "course",
		"cover", "cow", "crease", "create", "crop", "cross", "crowd", "cry", "current", "cut", "dad", "dance", "danger", "dark",
		"day", "dead", "deal", "dear", "death", "decide", "decimal", "deep", "degree", "depend", "describe", "desert", "design",
		"determine", "develop", "dictionary", "did", "die", "differ", "difficult", "direct", "discuss", "distant", "divide",
		"division", "do", "doctor", "does", "dog", "dollar", "don't", "done", "door", "double", "down", "draw", "dream",
		"dress", "drink", "drive", "drop", "dry", "duck", "during", "each", "ear", "early", "earth", "ease", "east", "eat",
		"edge", "effect", "egg", "eight", "either", "electric", "element", "else", "end", "enemy", "energy", "engine", "enough",
		"enter", "equal", "equate", "especially", "even", "evening", "event", "ever", "every", "exact", "example", "except",
		"excite", "exercise", "expect", "experience", "experiment", "eye", "face", "fact", "fair", "fall", "family", "famous",
		"far", "farm", "fast", "fat", "father", "favor", "fear", "feed", "feel", "feet", "fell", "felt", "few", "field", "fig",
		"fight", "figure", "fill", "final", "find", "fine", "finger", "finish", "fire", "first", "fish", "fit", "five", "flat",
		"floor", "flow", "flower", "fly", "follow", "food", "foot", "for", "force", "forest", "form", "forward", "found",
		"four", "fraction", "free", "fresh", "friend", "from", "front", "fruit", "full", "fun", "game", "garden", "gas",
		"gather", "gave", "general", "gentle", "get", "girl", "give", "glad", "glass", "go", "gold", "gone", "good", "got",
		"govern", "grand", "grass", "gray", "great", "green", "grew", "ground", "group", "grow", "guess", "guide", "gun", "had",
		"hair", "half", "hand", "happen", "happy", "hard", "has", "hat", "have", "he", "head", "hear", "heard", "heart", "heat",
		"heavy", "held", "help", "her", "here", "high", "hill", "him", "his", "history", "hit", "hold", "hole", "home", "hope",
		"horse", "hot", "hot", "hour", "house", "how", "huge", "human", "hundred", "hunt", "hurry", "I", "ice", "idea", "if",
		"imagine", "in", "inch", "include", "indicate", "industry", "insect", "instant", "instrument", "interest", "invent",
		"iron", "is", "island", "it", "job", "join", "joy", "jump", "just", "keep", "kept", "key", "kill", "kind", "king",
		"knew", "know", "lady", "lake", "land", "language", "large", "last", "late", "laugh", "law", "lay", "lead", "learn",
		"least", "leave", "led", "left", "leg", "length", "less", "let", "letter", "level", "lie", "life", "lift", "light",
		"like", "line", "liquid", "list", "listen", "little", "live", "locate", "log", "lone", "long", "look", "lost", "lot",
		"loud", "love", "low", "machine", "made", "magnet", "main", "major", "make", "man", "many", "map", "mark", "market",
		"mass", "master", "match", "material", "matter", "may", "me", "mean", "meant", "measure", "meat", "meet", "melody",
		"men", "metal", "method", "middle", "might", "mile", "milk", "million", "mind", "mine", "minute", "miss", "mix",
		"modern", "molecule", "moment", "money", "month", "moon", "more", "morning", "most", "mother", "motion", "mount",
		"mountain", "mouth", "move", "much", "multiply", "music", "must", "my", "name", "nation", "natural", "nature", "near",
		"necessary", "neck", "need", "neighbor", "never", "new", "next", "night", "nine", "no", "noise", "noon", "nor", "north",
		"nose", "note", "nothing", "notice", "noun", "now", "number", "numeral", "object", "observe", "occur", "ocean", "of",
		"off", "offer", "office", "often", "oh", "oil", "old", "on", "once", "one", "only", "open", "operate", "opposite", "or",
		"order", "organ", "original", "other", "our", "out", "over", "own", "oxygen", "page", "paint", "pair", "paper",
		"paragraph", "parent", "part", "particular", "party", "pass", "past", "path", "pattern", "pay", "people", "perhaps",
		"period", "person", "phrase", "pick", "picture", "piece", "pitch", "place", "plain", "plan", "plane", "planet", "plant",
		"play", "please", "plural", "poem", "point", "poor", "populate", "port", "pose", "position", "possible", "post",
		"pound", "power", "practice", "prepare", "present", "press", "pretty", "print", "probable", "problem", "process",
		"produce", "product", "proper", "property", "protect", "prove", "provide", "pull", "push", "put", "quart", "question",
		"quick", "quiet", "quite", "quotient", "race", "radio", "rail", "rain", "raise", "ran", "range", "rather", "reach",
		"read", "ready", "real", "reason", "receive", "record", "red", "region", "remember", "repeat", "reply", "represent",
		"require", "rest", "result", "rich", "ride", "right", "ring", "rise", "river", "road", "rock", "roll", "room", "root",
		"rope", "rose", "round", "row", "rub", "rule", "run", "safe", "said", "sail", "salt", "same", "sand", "sat", "save",
		"saw", "say", "scale", "school", "science", "score", "sea", "search", "season", "seat", "second", "section", "see",
		"seed", "seem", "segment", "select", "self", "sell", "send", "sense", "sent", "sentence", "separate", "serve", "set",
		"settle", "seven", "several", "shall", "shape", "share", "sharp", "she", "sheet", "shell", "shine", "ship", "shoe",
		"shop", "shore", "short", "should", "shoulder", "shout", "show", "side", "sight", "sign", "silent", "silver", "similar",
		"simple", "since", "sing", "single", "sister", "sit", "six", "size", "skill", "skin", "sky", "slave", "sleep", "slip",
		"slow", "small", "smell", "smile", "snow", "so", "soft", "soil", "soldier", "solution", "solve", "some", "son", "song",
		"soon", "sound", "south", "space", "speak", "special", "speech", "speed", "spell", "spend", "spoke", "spot", "spread",
		"spring", "square", "stand", "star", "start", "state", "station", "stay", "stead", "steam", "steel", "step", "stick",
		"still", "stone", "stood", "stop", "store", "story", "straight", "strange", "stream", "street", "stretch", "string",
		"strong", "student", "study", "subject", "substance", "subtract", "success", "such", "sudden", "suffix", "sugar",
		"suggest", "suit", "summer", "sun", "supply", "support", "sure", "surface", "surprise", "swim", "syllable", "symbol",
		"system", "table", "tail", "take", "talk", "tall", "teach", "team", "teeth", "tell", "temperature", "ten", "term",
		"test", "than", "thank", "that", "the", "their", "them", "then", "there", "these", "they", "thick", "thin", "thing",
		"think", "third", "this", "those", "though", "thought", "thousand", "three", "through", "throw", "thus", "tie", "time",
		"tiny", "tire", "to", "together", "told", "tone", "too", "took", "tool", "top", "total", "touch", "toward", "town",
		"track", "trade", "train", "travel", "tree", "triangle", "trip", "trouble", "truck", "true", "try", "tube", "turn",
		"twenty", "two", "type", "under", "unit", "until", "up", "us", "use", "usual", "valley", "value", "vary", "verb",
		"very", "view", "village", "visit", "voice", "vowel", "wait", "walk", "wall", "want", "war", "warm", "was", "wash",
		"watch", "water", "wave", "way", "we", "wear", "weather", "week", "weight", "well", "went", "were", "west", "what",
		"wheel", "when", "where", "whether", "which", "while", "white", "who", "whole", "whose", "why", "wide", "wife", "wild",
		"will", "win", "wind", "window", "wing", "winter", "wire", "wish", "with", "woman", "women", "won't", "wonder", "wood",
		"word", "work", "world", "would", "write", "written", "wrong", "wrote", "yard", "year", "yellow", "yes", "yet", "you",
		"young", "your"}
}
////////////////////////////////////////////////////////////////////////////////////////
// GoCompleter - the completer functions for parsing and offering completions for go lang expressions

type GoCompleter struct {

}

func (sc *GoCompleter) GenCompletions(text string) int {
	return 0
}

func (sc *GoCompleter) Parse(code string) {
	t, err := parser.ParseExpr(code)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%s", t)
}

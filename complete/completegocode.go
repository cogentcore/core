package complete

import (
	"encoding/json"
	"fmt"
	"go/token"
	"os/exec"
	"strconv"
	"strings"
)

type candidate struct {
	Class string `json:"class"`
	Name  string `json:"name"`
	Typ   string `json:"type"`
	Pkg   string `json:"package"`
}

// GetCompletions uses the gocode server to find possible completions at the specified position
// in the src (i.e. the byte slice passed in)
// bytes should be the current in memory version of the file
func GetCompletions(bytes []byte, pos token.Position) []Completion {
	var completions []Completion

	offset := pos.Offset
	offsetString := strconv.Itoa(offset)
	cmd := exec.Command("gocode", "-f=json", "-builtin", "autocomplete", offsetString)
	cmd.Stdin = strings.NewReader(string(bytes)) // use current state of file not disk version - may be stale
	result, _ := cmd.Output()
	var skip int = -1
	for i := 0; i < len(result); i++ {
		if result[i] == 123 { // 123 is 07b is '{'
			skip = i - 1 // stop when '{' is found
			break
		}
	}
	if skip != -1 {
		result = result[skip : len(result)-2] // strip off [N,[ at start (where N is some number) and trailing ]] as well
		data := make([]candidate, 0)
		err := json.Unmarshal(result, &data)
		if err != nil {
			fmt.Printf("%#v", err)
		}
		for _, aCandidate := range data {
			comp := Completion{Text: aCandidate.Name}
			//fmt.Println(aCandidate.Name)
			completions = append(completions, comp)
		}
	}
	return completions
}

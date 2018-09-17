package complete

import (
	"encoding/json"
	"fmt"
	"go/token"
	"os/exec"
	"strconv"
)

type candidate struct {
	Class string `json:"class"`
	Name  string `json:"name"`
	Typ   string `json:"type"`
	Pkg   string `json:"package"`
}

func GetCompletions(pos token.Position) []Completion {
	var completions []Completion

	offset := pos.Offset
	offsetString := strconv.Itoa(offset)
	fileStr := "--in=" + string(pos.Filename)
	cmd := exec.Command("gocode", "-f=json", fileStr, "autocomplete", offsetString)
	result, _ := cmd.Output()
	var skip int = -1
	for i := 0; i < len(result); i++ {
		if result[i] == 123 {
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
			completions = append(completions, comp)
		}
	}
	return completions
}

// Eventually this should be removed. Currently it gives backwards compatability to old
// versions that did not store the query count, which is now used for autocomplete.
package fuzzy

import (
	"encoding/json"
	"os"
)

type OldModel struct {
	Data            map[string]int      `json:"data"`
	Maxcount        int                 `json:"maxcount"`
	Suggest         map[string][]string `json:"suggest"`
	Depth           int                 `json:"depth"`
	Threshold       int                 `json:"threshold"`
	UseAutocomplete bool                `json:"autocomplete"`
}

// Converts the old model format to the new version
func (model *Model) convertOldFormat(filename string) error {
	oldmodel := new(OldModel)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	err = d.Decode(oldmodel)
	if err != nil {
		return err
	}

	// Correct for old models pre divergence measure
	if model.SuffDivergenceThreshold == 0 {
		model.SuffDivergenceThreshold = SuffDivergenceThresholdDefault
	}

	// Convert fields
	model.Maxcount = oldmodel.Maxcount
	model.Suggest = oldmodel.Suggest
	model.Depth = oldmodel.Depth
	model.Threshold = oldmodel.Threshold
	model.UseAutocomplete = oldmodel.UseAutocomplete

	// Convert the old counts
	if len(oldmodel.Data) > 0 {
		model.Data = make(map[string]*Counts, len(oldmodel.Data))
		for term, cc := range oldmodel.Data {
			model.Data[term] = &Counts{cc, 0}
		}
	}
	return nil
}

// in pkg/validation/report.go

package validation

import (
	"encoding/json"
	"os"

	"config-validator/pkg/automata"
)

// Report defines the structure of the final JSON output.
type Report struct {
	Status string   `json:"status"`
	Errors []string `json:"errors,omitempty"` // omitempty hides the field if there are no errors
}

// GenerateReport creates a JSON report file from the FSM's final state.
func GenerateReport(fsm *automata.FSM, outputFile string) error {
	var status string
	if len(fsm.Errors) == 0 {
		status = "success"
	} else {
		status = "failed"
	}

	// The new FSM only has an `Errors` field, which is all we need.
	report := Report{
		Status: status,
		Errors: fsm.Errors,
	}

	// Marshal the report into a nicely formatted JSON string.
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	// Write the JSON data to the specified output file.
	return os.WriteFile(outputFile, data, 0644)
}
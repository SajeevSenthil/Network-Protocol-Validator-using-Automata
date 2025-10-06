package config

import (
	"bufio"
	"fmt"
	"os"

	"config-validator/pkg/automata"
)

// ParseFile loads rules, creates a new Finite State Machine (FSM),
// and processes a configuration file line by line to validate it.
func ParseFile(inputFile string, rulesFile string) (*automata.FSM, error) {
	// Load the raw rules from the YAML file.
	rawRules, err := automata.LoadRules(rulesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules from %s: %v", rulesFile, err)
	}

	// Create a new FSM instance. This now returns an FSM and an error.
	// This is the section that was corrected to fix the compilation error.
	fsm, err := automata.NewFSM(rawRules)
	if err != nil {
		return nil, fmt.Errorf("failed to create FSM with provided rules: %v", err)
	}

	// Open the Cisco configuration file for reading.
	file, err := os.Open(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %v", inputFile, err)
	}
	defer file.Close()

	// Process the file line by line using the FSM.
	scanner := bufio.NewScanner(file)
	lineNum := 1
	for scanner.Scan() {
		fsm.ProcessLine(scanner.Text(), lineNum)
		lineNum++
	}

	// Check for any errors that occurred during the scanning process.
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Return the FSM, which now contains the results of the validation.
	return fsm, nil
}
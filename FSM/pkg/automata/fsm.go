package automata

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FSM is the Finite State Machine for validation.
// It holds the compiled rules, current state, and any errors found.
type FSM struct {
	Rules        map[string][]*regexp.Regexp
	CurrentState string
	Errors       []string
}

// LoadRules loads a YAML file and returns it as a map of strings.
func LoadRules(path string) (map[string][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rawRules map[string][]string
	err = yaml.Unmarshal(data, &rawRules)
	return rawRules, err
}

// NewFSM creates a new FSM instance.
// It takes raw string rules, compiles them into regular expressions for performance,
// and initializes the FSM in the "GLOBAL" state.
func NewFSM(rawRules map[string][]string) (*FSM, error) {
	compiledRules := make(map[string][]*regexp.Regexp)
	for state, patterns := range rawRules {
		for _, pattern := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to compile regex '%s' for state '%s': %v", pattern, state, err)
			}
			compiledRules[state] = append(compiledRules[state], re)
		}
	}

	return &FSM{
		Rules:        compiledRules,
		CurrentState: "GLOBAL",
		Errors:       []string{},
	}, nil
}

// ProcessLine is the core logic engine of the validator. It processes a single line of the configuration.
func (fsm *FSM) ProcessLine(originalLine string, lineNum int) {
	// Trim the line for matching, but keep the original to check for indentation.
	trimmedLine := strings.TrimSpace(originalLine)

	// --- 1. Handle Comments and Blank Lines ---
	// They are ignored but also reset the state to GLOBAL, which is safe behavior.
	if trimmedLine == "" || strings.HasPrefix(trimmedLine, "!") {
		fsm.CurrentState = "GLOBAL"
		return
	}

	// --- 2. Implement IMPLICIT EXIT Logic ---
	// This is the most critical fix. If we are in any sub-state (not GLOBAL) and the
	// current line is NOT indented, it means we have implicitly exited that block.
	if fsm.CurrentState != "GLOBAL" && !strings.HasPrefix(originalLine, " ") {
		fsm.CurrentState = "GLOBAL"
	}

	// --- 3. Implement ENTRY Logic ---
	// Check if the current line is a command that triggers a new state.
	if newState := fsm.findStateTrigger(trimmedLine); newState != "" {
		fsm.CurrentState = newState
		return // The trigger command itself is valid, so we move to the next line.
	}

	// --- 4. Validate the Line Against Rules for the Current State ---
	rulesForState, ok := fsm.Rules[fsm.CurrentState]
	if !ok {
		fsm.addError(lineNum, trimmedLine, fsm.CurrentState)
		return
	}

	isMatch := false
	for _, rule := range rulesForState {
		if rule.MatchString(trimmedLine) {
			isMatch = true
			break
		}
	}

	if !isMatch {
		fsm.addError(lineNum, trimmedLine, fsm.CurrentState)
	}
}

// findStateTrigger checks if a line matches a known pattern that starts a new configuration block.
func (fsm *FSM) findStateTrigger(line string) string {
	// These regex patterns define the commands that change the validator's state.
	triggers := map[string]string{
		`^interface\s+.*`:               "INTERFACE",
		`^aaa\s+group\s+server\s+.*`:     "AAA_GROUP",
		`^aaa\s+cache\s+profile\s+.*`:    "AAA_CACHE_PROFILE",
		`^dot11\s+ssid\s+.*`:             "DOT11_SSID",
		`^archive$`:                       "ARCHIVE_CONFIG",
		`^crypto\s+pki\s+.*`:             "CRYPTO_PKI",
		`^tacacs\s+server\s+.*`:          "SERVER_CONFIG",
		`^radius\s+server\s+.*`:          "SERVER_CONFIG",
		`^ip\s+access-list\s+standard\s+.*`: "IP_ACL_STANDARD",
		`^line\s+.*`:                      "LINE",
		`^router\s+.*`:                    "ROUTER", // Added for completeness
		`^vlan\s+[0-9]+`:                  "VLAN",   // Added for completeness
	}

	for pattern, state := range triggers {
		// We can ignore the error here because we know the patterns are valid.
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return state
		}
	}
	return "" // No state change was triggered
}

// addError formats and records a validation error.
func (fsm *FSM) addError(lineNum int, line, state string) {
	fsm.Errors = append(fsm.Errors,
		fmt.Sprintf("Line %d: invalid command '%s' in state %s", lineNum, line, state))
}
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"protocol-validator/pkg/automata"
	"protocol-validator/pkg/validation"
	"regexp"
	"time"
)

type DetailedError struct {
	ErrorType  string   `json:"error_type"`
	Line       int      `json:"line"`
	Position   int      `json:"position"`
	StackState []string `json:"pda_stack_state"`
	Suggestion string   `json:"suggestion"`
}

func main() {
	// CLI flags
	var outDir string
	var rootDir string
	flag.StringVar(&outDir, "outdir", ".", "directory where report files will be saved")
	flag.StringVar(&rootDir, "root", ".", "root directory to resolve relative input paths (helps locate files in nested workspaces)")
	flag.Parse()

	// Determine JSON path from remaining args (after flags)
	var jsonPath string
	args := flag.Args()
	if len(args) > 0 {
		jsonPath = args[0]
	} else {
		jsonPath = "protocol-validator/protocol-validator/request2.json"
	}

	// Resolve absolute paths for root, outdir
	if absRoot, err := filepath.Abs(rootDir); err == nil {
		rootDir = absRoot
	}
	if absOut, err := filepath.Abs(outDir); err == nil {
		outDir = absOut
	}

	// Resolve input path robustly:
	// 1) if absolute and exists -> use
	// 2) if relative and exists relative to cwd -> use
	// 3) try rootDir + jsonPath
	// 4) fallback: search recursively under rootDir for matching basename
	var resolvedInput string
	if filepath.IsAbs(jsonPath) {
		if _, err := os.Stat(jsonPath); err == nil {
			resolvedInput = jsonPath
		}
	} else {
		// try cwd-relative
		if cwdPath, err := filepath.Abs(jsonPath); err == nil {
			if _, err := os.Stat(cwdPath); err == nil {
				resolvedInput = cwdPath
			}
		}
		// try rootDir + jsonPath
		if resolvedInput == "" {
			cand := filepath.Join(rootDir, jsonPath)
			if _, err := os.Stat(cand); err == nil {
				resolvedInput = cand
			}
		}
		// fallback: search by basename under rootDir
		if resolvedInput == "" {
			base := filepath.Base(jsonPath)
			found := ""
			filepath.WalkDir(rootDir, func(p string, d os.DirEntry, err error) error {
				if found != "" || err != nil {
					return nil
				}
				if !d.IsDir() && filepath.Base(p) == base {
					found = p
				}
				return nil
			})
			if found != "" {
				resolvedInput = found
			}
		}
	}

	if resolvedInput == "" {
		// last resort: use provided jsonPath as-is (will likely error when reading)
		resolvedInput = jsonPath
	}
	jsonPath = resolvedInput

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		fmt.Printf("Failed to read %s: %v\n", jsonPath, err)
		return
	}

	httpInput := string(data)
	// Capture all printed output so we can save it to a file in the current directory
	var out bytes.Buffer
	fmt.Fprintf(&out, "Raw input received from %s : %s\n\n", jsonPath, httpInput)
	// Also print raw input to stdout for immediate feedback
	fmt.Print(out.String())

	// Run PDA-based JSON validation
	vErrs := validation.ValidateJSON(httpInput)
	if len(vErrs) > 0 {
		fmt.Println("==================== ERRORS DETECTED ====================")
		fmt.Fprintln(&out, "==================== ERRORS DETECTED ====================")
		var dErrs []DetailedError
		for _, vErr := range vErrs {
			line := findLineNumber(httpInput, vErr.Position)
			dErrs = append(dErrs, DetailedError{
				ErrorType:  vErr.ErrorType,
				Line:       line,
				Position:   vErr.Position,
				StackState: vErr.StackState,
				Suggestion: vErr.Suggestion,
			})
		}
		b, _ := json.MarshalIndent(dErrs, "", "  ")
		// Print to stdout and buffer
		fmt.Println(string(b))
		fmt.Fprintln(&out, string(b))
		fmt.Println("================== END OF ERRORS ==================")
		fmt.Fprintln(&out, "================== END OF ERRORS ==================")

		// Save the buffer to a timestamped file in the requested output directory
		saveReport(outDir, jsonPath, out.Bytes())
		return
	}

	// Success: print full JSON report
	type SuccessReport struct {
		Status     string   `json:"status"`
		File       string   `json:"file"`
		PDAStack   []string `json:"pda_stack_state"`
		TokenCount int      `json:"token_count"`
		LineCount  int      `json:"line_count"`
		Message    string   `json:"message"`
	}
	tokens := validation.TokenizeJSONWithLines(httpInput)
	pda := NewPDAForStack(tokens)
	report := SuccessReport{
		Status:     "valid",
		File:       jsonPath,
		PDAStack:   runeSliceToStringSlice(pda.StackSnapshot()),
		TokenCount: len(tokens),
		LineCount:  countLines(httpInput),
		Message:    " HTTP request and JSON body are valid.",
	}
	b, _ := json.MarshalIndent(report, "", "  ")
	// Print to stdout and append to buffer
	fmt.Println(string(b))
	fmt.Fprintln(&out, string(b))

	// Save the buffer to a timestamped file in the requested output directory
	saveReport(outDir, jsonPath, out.Bytes())
}

// findLineNumber maps a position index to line number in the JSON input
func findLineNumber(input string, pos int) int {
	line := 1
	for i, r := range input {
		if i >= pos {
			break
		}
		if r == '\n' {
			line++
		}
	}
	return line
}

// optional: regex-based parser if you feed external errors
func extractLineFromError(errMsg string) string {
	re := regexp.MustCompile(`(?i)line[ :]*([0-9]+)`)
	match := re.FindStringSubmatch(errMsg)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// Helper: create PDA and return stack after processing tokens
func NewPDAForStack(tokens []validation.TokenInfo) *automata.PDA {
	pda := automata.NewPDA()
	for _, t := range tokens {
		switch t.Token {
		case "{", "[":
			pda.Push(rune(t.Token[0]))
		case "}":
			if pda.Peek() == '{' {
				pda.Pop()
			}
		case "]":
			if pda.Peek() == '[' {
				pda.Pop()
			}
		}
	}
	return pda
}

// Helper: count lines in input
func countLines(input string) int {
	count := 1
	for _, r := range input {
		if r == '\n' {
			count++
		}
	}
	return count
}

func runeSliceToStringSlice(runes []rune) []string {
	result := make([]string, len(runes))
	for i, r := range runes {
		result[i] = string(r)
	}
	return result
}

// saveReport writes report bytes into a timestamped file in the current working directory.
func saveReport(outDir string, inputPath string, data []byte) {
	// Ensure the output directory exists
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Printf("Failed to create outdir %s: %v\n", outDir, err)
		return
	}
	// use basename of input to name file
	base := filepath.Base(inputPath)
	ts := time.Now().Format("20060102-150405")
	outName := fmt.Sprintf("validation-output-%s-%s.txt", base, ts)
	outPath := filepath.Join(outDir, outName)
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Printf("Failed to write report to %s: %v\n", outPath, err)
		return
	}
	fmt.Printf("Saved report to: %s\n", outPath)
}

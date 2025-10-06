package main

import (
	"flag"
	"fmt"
	"log"

	"config-validator/pkg/config"
	"config-validator/pkg/validation"
)

func main() {
	// CLI flags
	inputFile := flag.String("input", "test/sample_config.txt", "Cisco config file to validate")
	outputFile := flag.String("out", "test/report.json", "Path to JSON validation report")
	rulesFile := flag.String("rules", "pkg/automata/rules.yaml", "Path to YAML rules file")
	flag.Parse()

	// Parse Cisco config with FSM + rules
	fsm, err := config.ParseFile(*inputFile, *rulesFile)
	if err != nil {
		log.Fatal("❌ Error parsing file:", err)
	}

	// Generate JSON report
	err = validation.GenerateReport(fsm, *outputFile)
	if err != nil {
		log.Fatal("❌ Error generating report:", err)
	}

	fmt.Println("✅ Validation complete. Report written to", *outputFile)
}

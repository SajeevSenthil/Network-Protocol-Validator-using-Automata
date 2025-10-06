# Network Protocol Validator (PDA + FSM)

This repository contains two related validator projects that demonstrate two common validation techniques for network-related text formats:

- PDA-based JSON/HTTP validator (located in `PDA/`) — a structural JSON validator implemented with a tokenizer and a pushdown-like stack to detect bracket/colon/comma errors and basic value correctness. It is primarily intended to validate HTTP-style request objects and JSON payloads.
- FSM-based Cisco configuration validator (located in `FSM/`) — a finite-state-machine validator that applies YAML-defined regular-expression rules to line-oriented Cisco-style configuration files. It models block entry/exit via triggers and indentation and reports per-line errors where rules do not match.

This README explains the repository layout, how to run each validator, expected inputs/outputs, and best practices for working with the code.

## Repository structure

Top-level folders and purpose:

- `PDA/` — PDA-based JSON/HTTP validator
	- `cmd/http-validator/` — CLI entrypoint for the PDA validator
	- `pkg/validation/` — tokenizer and validator logic
	- `pkg/automata/` — minimal PDA stack helper
	- `pkg/http/` — helpers for validating HTTP-style objects
- `FSM/` — FSM-based Cisco config validator
	- `cmd/config-validator/` — CLI entrypoint for the FSM validator
	- `pkg/automata/` — FSM implementation and rule loader (YAML)
	- `pkg/config/` — parser that feeds lines into the FSM
	- `pkg/validation/` — report generator
	- `test/` — example configuration files and sample reports
- `README.md` — this file

## Prerequisites

- Go 1.20+ installed and on the PATH. Both projects use standard Go modules; run commands from the repository root or the specific subproject folder.
- A terminal (bash/PowerShell) with network access for cloning/pushing to the remote repo when needed.

Note: some convenience Windows binaries may be present in the `FSM/` folder; those are optional and can be removed from source control.

## PDA-based JSON/HTTP validator (PDA/)

Purpose
- Structural JSON validation using a tokenizer and a simple pushdown stack approach. The validator detects mismatched braces/brackets, misplaced commas/colons, and invalid value tokens. It also includes a helper to validate a simple HTTP-style wrapper object whose `body` is validated using the same engine.

How to run (development)

From the repository root:

```bash
go run ./PDA/cmd/http-validator [path/to/input.json]
```

If no input file is provided, the CLI defaults to a sample JSON under the project.

Flags and behavior
- `--root <path>`: (optional) base directory used to resolve relative input paths when they are not found in the current working directory.
- `--outdir <path>`: (optional) directory where the validator saves a timestamped report file summarizing the raw input and validation result. If not specified, the report will be written into the current working directory.

Output
- On validation errors: the CLI prints a JSON array of error objects containing `error_type`, `line`, `position`, `pda_stack_state`, and `suggestion`.
- On success: the CLI prints a `SuccessReport` JSON object with `status: "valid"`, token/line counts, and a stack snapshot.

Example

```bash
# validate a JSON sample and save a report into ./reports/
go run ./PDA/cmd/http-validator --root "D:/1. FLA/FLA" --outdir reports path/to/request.json
```

Notes
- The PDA validator is structural, not a JSON Schema validator. It validates the shape and basic token correctness, not semantic constraints.

## FSM-based Cisco config validator (FSM/)

Purpose
- Line-oriented validation of Cisco-style configuration files using a finite-state machine whose rules are declared in `FSM/pkg/automata/rules.yaml`. The FSM recognizes block entry triggers (e.g., `interface`, `router`) and validates subsequent indented lines using state-specific regex rules.

How to run

From the repository root:

```bash
go run ./FSM/cmd/config-validator -input FSM/test/sample_config.txt -out FSM/test/report.json -rules FSM/pkg/automata/rules.yaml
```

Or on Windows you may run the included binary (if present):

```powershell
.\FSM\validator.exe -input FSM\test\sample_config.txt -out FSM\test\report.json -rules FSM\pkg\automata\rules.yaml
```

Output
- A JSON report file with structure `{ "status": "success|failed", "errors": [ ... ] }`. Each error is a formatted string that identifies the line number, the offending text, and the state where validation failed.

Example

```bash
# run validator and print report
go run ./FSM/cmd/config-validator -input FSM/test/sample_config.txt -out FSM/test/report.json -rules FSM/pkg/automata/rules.yaml
cat FSM/test/report.json
```

Notes and limitations
- Rules are regular expressions and must be maintained in `rules.yaml`. Complex semantic checks (for example, IP range checks, subnet validity, numeric bounds) are not performed by regex; consider adding programmatic validators for those cases.
- The FSM uses leading-space indentation to infer implicit block exits. If your config uses tabs or irregular indentation, you may need to normalize input first.

## Development & build

Build a single tool (example: FSM CLI):

```bash
cd FSM
go build -o bin/config-validator ./cmd/config-validator

# run the built binary
./bin/config-validator -input test/sample_config.txt -out test/report.json -rules pkg/automata/rules.yaml
```

Build the PDA CLI similarly:

```bash
cd PDA
go build -o bin/http-validator ./cmd/http-validator
```

## Repository hygiene recommendations

- Add a `.gitignore` to avoid committing generated artifacts and OS-specific binaries. Example entries to add to the repository root `.gitignore`:

```
# Binaries
*.exe
*.exe~

# Reports and generated output
/PDA/reports/
/FSM/test/report.json

# Editor and OS files
.DS_Store
*.swp
```

- Consider removing tracked binary files from history (they are currently present in `FSM/validator.exe`). To remove an already committed binary without deleting the local copy:

```bash
git rm --cached FSM/validator.exe
git commit -m "Remove tracked binary; add to .gitignore"
git push origin main
```

## Contributing

Contributions are welcome. Suggested steps for changes that affect behavior:

1. Open an issue describing the change or bug.
2. Create a feature branch from `main`.
3. Add unit tests where appropriate (preferably under each package's folder) and ensure `go test ./...` passes.
4. Submit a pull request with a clear description and any relevant test results.

## License

This repository does not include a license file. Add an appropriate open-source license (for example, MIT, Apache 2.0) to make reuse explicit.

If you want, I can:
- add a conservative `.gitignore` and commit it, and
- remove the tracked binary and report files and push the cleanup commit.

---

End of documentation.

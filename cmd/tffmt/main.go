// Package tffmt provides the entry point for the tffmt CLI
package tffmt

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/krewenki/tffmt/pkg/config"
	"github.com/krewenki/tffmt/pkg/formatter"
	"github.com/pmezard/go-difflib/difflib"
)

var (
	cfg           *config.Config
	formatterInst *formatter.Formatter
)

// Main is the entry point for the tffmt CLI
func Main() {
	// Initialize configuration and formatter
	cfg = config.NewConfig()
	formatterInst = formatter.New(cfg)

	// Setup command-line flags
	flag.BoolVar(&cfg.Write, "write", cfg.Write, "write result to source file(s)")
	flag.BoolVar(&cfg.Check, "check", cfg.Check, "check if files are already formatted")
	flag.BoolVar(&cfg.List, "list", cfg.List, "list files whose formatting differs")
	flag.BoolVar(&cfg.Diff, "diff", cfg.Diff, "display diffs")
	flag.BoolVar(&cfg.Recursive, "recursive", cfg.Recursive, "recurse into sub‑directories")
	flag.BoolVar(&cfg.Test, "test", cfg.Test, "run tests")
	flag.BoolVar(&cfg.SortInputs, "sort-inputs", cfg.SortInputs, "alphabetize inputs in resources")
	flag.BoolVar(&cfg.SortVars, "sort-vars", cfg.SortVars, "alphabetize variables in variable blocks")
	flag.Parse()

	// Load settings from config file
	settings, err := config.LoadSettings()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load settings: %v\n", err)
	} else {
		// Track which flags were explicitly set by the user
		passedFlags := make(map[string]bool)
		flag.Visit(func(f *flag.Flag) {
			passedFlags[f.Name] = true
		})

		// Update config with settings from file
		config.ApplySettings(cfg, settings, passedFlags)
	}

	// Get paths from arguments
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Process paths
	exit := 0
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			exit = 1
			continue
		}

		if info.IsDir() {
			if err := walkDir(p); err != nil {
				fmt.Fprintln(os.Stderr, err)
				exit = 1
			}
		} else if filepath.Ext(p) == ".tf" {
			changed, err := processFile(p)
			_ = handleResult(changed, err, &exit)
		}
	}
	os.Exit(exit)
}

// main calls Main for local development
func main() {
	Main()
}

// walkDir recursively processes terraform files in a directory
func walkDir(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !cfg.Recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".tf" {
			changed, err := processFile(path)
			if err != nil {
				return err
			}
			// Don't stop the walk on changed files, let handleResult determine exit code
			_ = handleResult(changed, nil, new(int))
		}
		return nil
	})
}

// processFile formats a single terraform file
func processFile(path string) (changed bool, err error) {
	orig, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	formatted, changed := formatterInst.FormatFile(orig)

	// Handle flags for output
	if cfg.List && changed {
		fmt.Println(path)
	}
	if cfg.Diff && changed {
		showDiff(path, orig, formatted)
	}
	if cfg.Write && changed && !cfg.Check {
		info, err := os.Stat(path)
		if err != nil {
			return changed, err
		}
		err = os.WriteFile(path, formatted, info.Mode().Perm())
		if err != nil {
			return changed, err
		}
	}
	return changed, nil
}

// showDiff displays the formatting changes in unified diff format
func showDiff(path string, a, b []byte) {
	u := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(a)),
		B:        difflib.SplitLines(string(b)),
		FromFile: path + " (orig)",
		ToFile:   path + " (fmt)",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(u)
	fmt.Print(text)
}

// handleResult processes errors and sets exit codes
func handleResult(changed bool, err error, exit *int) error {
	if err != nil {
		fmt.Fprintln(os.Stderr, "tffmt:", err)
		*exit = 1
		return err
	}
	if changed && cfg.Check && *exit == 0 {
		*exit = 3 // terraform fmt's "needs formatting" code
	}
	return nil
}

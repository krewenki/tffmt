package tffmt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/krewenki/tffmt/pkg/config"
)

func TestProcessFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "tffmt-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		content        string
		expectedOutput string
		expectChange   bool
	}{
		{
			name:           "already formatted",
			content:        "resource \"example\" \"test\" {\n  foo = bar\n}\n\n",
			expectedOutput: "resource \"example\" \"test\" {\n  foo = bar\n}\n\n",
			expectChange:   false,
		},
		{
			name:           "needs formatting",
			content:        "resource \"example\" \"test\" {\nfoo = bar\n}",
			expectedOutput: "resource \"example\" \"test\" {\n  foo = bar\n}\n\n",
			expectChange:   true,
		},
		{
			name:           "parens and braces",
			content:        "resource \"example\" \"test\" ({foo = bar})",
			expectedOutput: "resource \"example\" \"test\" (\n  { foo = bar }\n\n)\n\n",
			expectChange:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tmpDir, tt.name+".tf")
			err := os.WriteFile(filePath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatal(err)
			}

			// Save original config and restore it afterwards
			origCfg := cfg
			defer func() { cfg = origCfg }()

			// Set config for test
			cfg = config.NewConfig()
			cfg.Write = true
			cfg.Check = false
			cfg.List = false
			cfg.Diff = false

			// Process the file
			changed, err := processFile(filePath)
			if err != nil {
				t.Fatal(err)
			}

			// Check if the change detection is correct
			if changed != tt.expectChange {
				t.Errorf("processFile() changed = %v, want %v", changed, tt.expectChange)
			}

			// Read the processed file
			output, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatal(err)
			}

			// Check if the output matches the expected result
			if !bytes.Equal(output, []byte(tt.expectedOutput)) {
				t.Errorf("processFile() output = %q, want %q", output, tt.expectedOutput)
			}
		})
	}
}

func TestCheckFlag(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "tffmt-check-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create an unformatted file
	filePath := filepath.Join(tmpDir, "unformatted.tf")
	content := "resource \"example\" \"test\" {foo = bar}"
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Save original config and restore it afterwards
	origCfg := cfg
	defer func() { cfg = origCfg }()

	// Set config for test
	cfg = config.NewConfig()
	cfg.Write = false
	cfg.Check = true

	exit := 0
	changed, err := processFile(filePath)
	handleResult(changed, err, &exit)

	if exit != 3 {
		t.Errorf("handleResult() with check=true and unformatted file should set exit=3, got %d", exit)
	}

	// Check that file wasn't modified
	output, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(output, []byte(content)) {
		t.Errorf("File was modified when check=true: got %q, want %q", output, content)
	}
}

func TestHandleResult(t *testing.T) {
	testCases := []struct {
		name       string
		changed    bool
		checkFlag  bool
		inputError error
		expectExit int
	}{
		{"no change, no error", false, false, nil, 0},
		{"changed, no error, no check", true, false, nil, 0},
		{"changed, check enabled", true, true, nil, 3},
		{"error occurs", false, false, os.ErrNotExist, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save original config and restore it afterwards
			origCfg := cfg
			defer func() { cfg = origCfg }()

			// Set config for test
			cfg = config.NewConfig()
			cfg.Check = tc.checkFlag

			exit := 0
			err := handleResult(tc.changed, tc.inputError, &exit)

			if (err != nil) != (tc.inputError != nil) {
				t.Errorf("handleResult() error = %v, want %v", err, tc.inputError)
			}

			if exit != tc.expectExit {
				t.Errorf("handleResult() exit = %d, want %d", exit, tc.expectExit)
			}
		})
	}
}

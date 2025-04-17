package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPreprocess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "basic transformation",
			input:    "resource \"aws_s3_bucket\" \"example\" ({foo = bar})",
			expected: "resource \"aws_s3_bucket\" \"example\" (\n{foo = bar}\n)",
		},
		{
			name:     "multiple transformations",
			input:    "a({ b }) c({ d }) e",
			expected: "a(\n{ b }\n) c(\n{ d }\n) e",
		},
		{
			name:     "with whitespace",
			input:    "a( { b } ) c( { d } ) e",
			expected: "a(\n{ b }\n) c(\n{ d }\n) e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preprocess([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("preprocess() = %q, want %q", result, tt.expected)
			}
		})
	}
}

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
		{
			name:           "multiple blocks without blank line",
			content:        "block1 {}\nblock2 {}\n",
			expectedOutput: "block1 {}\n\nblock2 {}\n\n",
			expectChange:   true,
		},
		{
			name:           "too many blank lines",
			content:        "block1 {}\n\n\n\nblock2 {}\n",
			expectedOutput: "block1 {}\n\nblock2 {}\n\n",
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

			// Save original flag values
			origWrite := *write
			origCheck := *check
			origList := *list
			origDiff := *diff

			// Modify flags for test
			*write = true
			*check = false
			*list = false
			*diff = false

			// Restore flags after test
			defer func() {
				*write = origWrite
				*check = origCheck
				*list = origList
				*diff = origDiff
			}()

			// Process the file
			changed, err := process(filePath)
			if err != nil {
				t.Fatal(err)
			}

			// Check if the change detection is correct
			if changed != tt.expectChange {
				t.Errorf("process() changed = %v, want %v", changed, tt.expectChange)
			}

			// Read the processed file
			output, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatal(err)
			}

			// Check if the output matches the expected result
			if !bytes.Equal(output, []byte(tt.expectedOutput)) {
				t.Errorf("process() output = %q, want %q", output, tt.expectedOutput)
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

	// Save original flag values
	origWrite := *write
	origCheck := *check

	// Test with check=true, write=false
	*write = false
	*check = true

	exit := 0
	changed, err := process(filePath)
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

	// Restore flags
	*write = origWrite
	*check = origCheck
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
			// Save original check flag
			origCheck := *check
			*check = tc.checkFlag
			defer func() { *check = origCheck }()

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

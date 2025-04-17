package formatter

import (
	"testing"
)

func TestPreprocess(t *testing.T) {
	formatter := New()
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
			result := formatter.Preprocess([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("Preprocess() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	formatter := New()
	tests := []struct {
		name         string
		input        string
		expected     string
		expectChange bool
	}{
		{
			name:         "already formatted",
			input:        "resource \"example\" \"test\" {\n  foo = bar\n}\n\n",
			expected:     "resource \"example\" \"test\" {\n  foo = bar\n}\n\n",
			expectChange: false,
		},
		{
			name:         "needs formatting",
			input:        "resource \"example\" \"test\" {\nfoo = bar\n}",
			expected:     "resource \"example\" \"test\" {\n  foo = bar\n}\n\n",
			expectChange: true,
		},
		{
			name:         "parens and braces",
			input:        "resource \"example\" \"test\" ({foo = bar})",
			expected:     "resource \"example\" \"test\" (\n  { foo = bar }\n\n)\n\n",
			expectChange: true,
		},
		{
			name:         "multiple blocks without blank line",
			input:        "block1 {}\nblock2 {}\n",
			expected:     "block1 {}\n\nblock2 {}\n\n",
			expectChange: true,
		},
		{
			name:         "too many blank lines",
			input:        "block1 {}\n\n\n\nblock2 {}\n",
			expected:     "block1 {}\n\nblock2 {}\n\n",
			expectChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted, changed := formatter.FormatFile([]byte(tt.input))

			if changed != tt.expectChange {
				t.Errorf("FormatFile() changed = %v, want %v", changed, tt.expectChange)
			}

			if string(formatted) != tt.expected {
				t.Errorf("FormatFile() output = %q, want %q", formatted, tt.expected)
			}
		})
	}
}

package formatter

import (
	"testing"

	"github.com/krewenki/tffmt/pkg/config"
)

func TestPreprocess(t *testing.T) {
	cfg := config.NewConfig()
	formatter := New(cfg)
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
	cfg := config.NewConfig()
	formatter := New(cfg)
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

// TestSortInputs verifies the sort-inputs functionality
func TestSortInputs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		sortFlag bool
	}{
		{
			name: "sort inputs enabled",
			input: `resource "aws_instance" "example" {
  zone = "us-west-1a"
  ami = "ami-12345"
  instance_type = "t2.micro"
}`,
			expected: `resource "aws_instance" "example" {
  ami = "ami-12345"
  instance_type = "t2.micro"
  zone = "us-west-1a"
}`,
			sortFlag: true,
		},
		{
			name: "sort inputs disabled",
			input: `resource "aws_instance" "example" {
  zone = "us-west-1a"
  ami = "ami-12345"
  instance_type = "t2.micro"
}`,
			expected: `resource "aws_instance" "example" {
  zone = "us-west-1a"
  ami = "ami-12345"
  instance_type = "t2.micro"
}`,
			sortFlag: false,
		},
		{
			name: "multiple resources with nested blocks",
			input: `resource "aws_instance" "web" {
  vpc_security_group_ids = ["sg-12345"]
  instance_type = "t2.micro"
  ami = "ami-12345"

  tags = {
    Name = "web-server"
  }
}

resource "aws_instance" "db" {
  monitoring = true
  instance_type = "t3.small"
  ami = "ami-67890"
}`,
			expected: `resource "aws_instance" "web" {
  ami = "ami-12345"
  instance_type = "t2.micro"
  vpc_security_group_ids = ["sg-12345"]

  tags = {
    Name = "web-server"
  }
}

resource "aws_instance" "db" {
  ami = "ami-67890"
  instance_type = "t3.small"
  monitoring = true
}`,
			sortFlag: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewConfig()
			cfg.SortInputs = tt.sortFlag
			formatter := New(cfg)

			// We need to format the input and expected to normalize whitespace
			// for a fair comparison after sort
			formatted := formatter.Format([]byte(tt.input))
			expectedFormatted := formatter.Format([]byte(tt.expected))

			if string(formatted) != string(expectedFormatted) {
				t.Errorf("Format() with sort-inputs=%v produced unexpected result.\nGot:\n%s\n\nWant:\n%s",
					tt.sortFlag, formatted, expectedFormatted)
			}
		})
	}
}

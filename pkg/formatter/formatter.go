// Package formatter provides functionality to format Terraform files
// according to custom formatting rules.
package formatter

import (
	"bytes"
	"regexp"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Regex patterns for transformations
var (
	reOpenParenBrace  = regexp.MustCompile(`\(\s*{`)
	reCloseBraceParen = regexp.MustCompile(`}\s*\)`)
	reCollapseBlank   = regexp.MustCompile(`\n{3,}`)     // ≥3 ⇒ 2
	rePadSingle       = regexp.MustCompile(`}\n([^\n])`) // 1 ⇒ 2
)

// Formatter holds configuration for the formatting process
type Formatter struct{}

// New creates a new Formatter instance
func New() *Formatter {
	return &Formatter{}
}

// Format processes a single terraform file and returns the formatted content
func (f *Formatter) Format(content []byte) []byte {
	// 1. custom pre-split
	src := f.Preprocess(content)

	// 2. canonical hcl formatting
	form := hclwrite.Format(src)

	// 3. 2 blank lines between top-level blocks
	form = reCollapseBlank.ReplaceAll(form, []byte("\n\n"))
	form = rePadSingle.ReplaceAll(form, []byte("}\n\n$1"))

	// 4. ensure exactly two trailing newlines
	form = bytes.TrimRight(form, "\n")
	form = append(form, '\n', '\n')

	return form
}

// Preprocess performs initial transformations on terraform content
// such as splitting "({" and "})" into separate lines
func (f *Formatter) Preprocess(in []byte) []byte {
	out := reOpenParenBrace.ReplaceAll(in, []byte("(\n{"))
	out = reCloseBraceParen.ReplaceAll(out, []byte("}\n)"))
	return out
}

// FormatFile formats the content of a terraform file and determines if it changed
func (f *Formatter) FormatFile(content []byte) (formatted []byte, changed bool) {
	formatted = f.Format(content)
	return formatted, !bytes.Equal(content, formatted)
}

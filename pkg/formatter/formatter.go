// Package formatter provides functionality to format Terraform files
// according to custom formatting rules.
package formatter

import (
	"bytes"
	"regexp"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/krewenki/tffmt/pkg/config"
)

// Regex patterns for transformations
var (
	reOpenParenBrace  = regexp.MustCompile(`\(\s*{`)
	reCloseBraceParen = regexp.MustCompile(`}\s*\)`)
	reCollapseBlank   = regexp.MustCompile(`\n{3,}`)                // ≥3 ⇒ 2
	rePadSingle       = regexp.MustCompile(`}\n([^\n])`)            // 1 ⇒ 2
	reResourceBlocks  = regexp.MustCompile(`}\n{0,2}(resource\s+)`) // Ensure exactly 2 newlines between resource blocks
)

// Formatter holds configuration for the formatting process
type Formatter struct {
	Config *config.Config
}

// New creates a new Formatter instance
func New(cfg *config.Config) *Formatter {
	return &Formatter{
		Config: cfg,
	}
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

	// Ensure exactly two newlines between resource blocks
	form = reResourceBlocks.ReplaceAll(form, []byte("}\n\n$1"))

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

	// Apply additional transformations if SortInputs is enabled
	if f.Config.SortInputs {
		out = f.sortResourceInputs(out)
	}

	// Apply variable sorting if SortVars is enabled
	if f.Config.SortVars {
		out = f.sortVariableBlocks(out)
	}

	return out
}

// sortResourceInputs alphabetically sorts the inputs within resource blocks
func (f *Formatter) sortResourceInputs(in []byte) []byte {
	// Parse the HCL content
	file, err := hclwrite.ParseConfig(in, "", hcl.InitialPos)
	if err != nil {
		// If there's an error parsing, return the original content unchanged
		return in
	}

	// Process all top level blocks
	for _, block := range file.Body().Blocks() {
		// We're looking for resource blocks
		if block.Type() == "resource" {
			// Get the attribute names in the block body
			attributes := block.Body().Attributes()
			if len(attributes) > 0 {
				// Get all attribute names
				attrNames := make([]string, 0, len(attributes))
				for name := range attributes {
					attrNames = append(attrNames, name)
				}

				// Sort the attribute names
				sort.Strings(attrNames)

				// Create a temporary map to hold all attributes
				attrMap := make(map[string]*hclwrite.Attribute)
				for name, attr := range attributes {
					attrMap[name] = attr
				}

				// Remove all attributes from the block
				for name := range attributes {
					block.Body().RemoveAttribute(name)
				}

				// Add them back in sorted order
				for _, name := range attrNames {
					expr := attrMap[name].Expr().BuildTokens(nil)
					block.Body().SetAttributeRaw(name, expr)
				}
			}
		}
	}

	// Return the formatted output
	return file.Bytes()
}

// sortVariableBlocks alphabetically sorts variables within variable blocks
func (f *Formatter) sortVariableBlocks(in []byte) []byte {
	// Parse the HCL content
	file, err := hclwrite.ParseConfig(in, "", hcl.InitialPos)
	if err != nil {
		// If there's an error parsing, return the original content unchanged
		return in
	}

	// Get all variable blocks
	var varBlocks []*hclwrite.Block
	for _, block := range file.Body().Blocks() {
		if block.Type() == "variable" {
			varBlocks = append(varBlocks, block)
		}
	}

	// If there are multiple variable blocks, sort them by their labels
	if len(varBlocks) > 1 {
		// Remove all variable blocks from the file body
		for _, block := range varBlocks {
			file.Body().RemoveBlock(block)
		}

		// Sort variable blocks by their labels (names)
		sort.Slice(varBlocks, func(i, j int) bool {
			if len(varBlocks[i].Labels()) == 0 || len(varBlocks[j].Labels()) == 0 {
				return false
			}
			return string(varBlocks[i].Labels()[0]) < string(varBlocks[j].Labels()[0])
		})

		// Add the blocks back in sorted order
		for i, block := range varBlocks {
			file.Body().AppendBlock(block)

			// Add a newline between blocks except after the last block
			if i < len(varBlocks)-1 {
				file.Body().AppendNewline()
			}
		}
	}

	// Return the formatted output
	return file.Bytes()
}

// FormatFile formats the content of a terraform file and determines if it changed
func (f *Formatter) FormatFile(content []byte) (formatted []byte, changed bool) {
	formatted = f.Format(content)
	return formatted, !bytes.Equal(content, formatted)
}

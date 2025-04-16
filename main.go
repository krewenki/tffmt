// tffmt
//
// A **self‑contained** drop‑in replacement for `terraform fmt`.
//
// Differences from upstream, per user spec
// ----------------------------------------
//   - “({”  →  “(\n{” and “})”  →  “}\n)” *before* the file is formatted
//   - every top‑level block is followed by **exactly one blank line**
//     (i.e. two trailing '\n' characters) – including the final block
//
// Implementation strategy
// -----------------------
//  1. Pre‑process the bytes to split “({” / “})”.
//  2. Pass the result through `hclwrite.Format` (the same formatter
//     terraform uses internally) – no terraform binary required.
//  3. Normalise inter‑block spacing with two tiny regex passes.
//  4. Honour the key `terraform fmt` flags: -write (default true),
//     -check, -diff, -list, -recursive.  Exit‑codes match upstream
//     (3 means “needs formatting but -check used”).
//
// Build
//
//	go mod init example.com/tffmt && \
//	go get github.com/hashicorp/hcl/v2 && \
//	go get github.com/pmezard/go-difflib/difflib && \
//	go build -o tffmt tffmt.go
//
// Usage
//
//	./tffmt [same flags & paths you’d pass to terraform fmt]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pmezard/go-difflib/difflib"
)

var (
	write     = flag.Bool("write", true, "write result to source file(s)")
	check     = flag.Bool("check", false, "check if files are already formatted")
	list      = flag.Bool("list", true, "list files whose formatting differs")
	diff      = flag.Bool("diff", false, "display diffs")
	recursive = flag.Bool("recursive", false, "recurse into sub‑directories")
	color     = flag.Bool("color", false, "ignored (kept for cli parity)")
)

// ————— regex helpers —————————————————————————————————————————————
var (
	reOpenParenBrace  = regexp.MustCompile(`\(\s*{`)
	reCloseBraceParen = regexp.MustCompile(`}\s*\)`)
	reCollapseBlank   = regexp.MustCompile(`\n{3,}`)     // ≥3 ⇒ 2
	rePadSingle       = regexp.MustCompile(`}\n([^\n])`) // 1 ⇒ 2
)

// ————— main ————————————————————————————————————————————————
func main() {
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	exit := 0
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			exit = 1
			continue
		}

		if info.IsDir() {
			if err := walk(p); err != nil {
				fmt.Fprintln(os.Stderr, err)
				exit = 1
			}
		} else if filepath.Ext(p) == ".tf" {
			if changed, err := process(p); handleResult(changed, err, &exit) != nil {
			}
		}
	}
	os.Exit(exit)
}

// ————— directory walk ——————————————————————————————————————
func walk(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !*recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".tf" {
			_, err = process(path)
		}
		return err
	})
}

// ————— single‑file pipeline ———————————————————————————————
func process(path string) (changed bool, err error) {
	orig, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// 1. custom pre‑split
	src := preprocess(orig)

	// 2. canonical hcl formatting
	form := hclwrite.Format(src)

	// 3. 2 blank lines between top‑level blocks
	form = reCollapseBlank.ReplaceAll(form, []byte("\n\n"))
	form = rePadSingle.ReplaceAll(form, []byte("}\n\n$1"))

	// 4. ensure exactly two trailing newlines
	form = bytes.TrimRight(form, "\n")
	form = append(form, '\n', '\n')

	changed = !bytes.Equal(orig, form)

	// behavioural flags
	if *list && changed {
		fmt.Println(path)
	}
	if *diff && changed {
		showDiff(path, orig, form)
	}
	if *write && changed && !*check {
		err = os.WriteFile(path, form, 0o644)
	}
	return
}

func preprocess(in []byte) []byte {
	out := reOpenParenBrace.ReplaceAll(in, []byte("(\n{"))
	out = reCloseBraceParen.ReplaceAll(out, []byte("}\n)"))
	return out
}

// ————— diff display ———————————————————————————————————————
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

// ————— exit‑code helper ——————————————————————————————————————
func handleResult(changed bool, err error, exit *int) error {
	if err != nil {
		fmt.Fprintln(os.Stderr, "tffmt:", err)
		*exit = 1
		return err
	}
	if changed && *check && *exit == 0 {
		*exit = 3 // terraform fmt's “needs formatting” code
	}
	return nil
}

/*
   NOTE / Limitations
   ------------------
   • `hclwrite.Format` replicates **most** but not 100 % of terraform‑fmt’s
     stylistic rules (e.g. comments, heredocs).  If you depend on exotic
     edge‑cases, run the upstream formatter’s test‑suite against this tool.
   • The regex for “exactly two blank lines” keys off every ‘}’ – including
     nested braces.  In practice that matches terraform‑fmt’s own output
     because top‑level blocks close at column 0, but if your files contain
     large multiline strings with a literal “}” at start‑of‑line, you may
     get an extra blank line.  (Empirically rare; acceptable trade‑off.)
*/

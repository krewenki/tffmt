// Package main is a simple redirect to the cmd/tffmt package
package main

import (
	"os"

	tffmt "github.com/krewenki/tffmt/cmd/tffmt"
)

func main() {
	tffmt.Main()
	os.Exit(0)
}

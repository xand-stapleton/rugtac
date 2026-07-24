// Command rugtac-index downloads and converts mathlib's tactic documentation
// to rugtac's local JSON format.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"rugtac/internal/catalog"
	"rugtac/internal/indexer"
)

func main() {
	sourceURL := flag.String("url", indexer.DefaultURL, "URL of the generated mathlib tactics page")
	inputPath := flag.String("input", "", "parse a downloaded HTML file instead of fetching the URL")
	outputPath := flag.String("out", "data/tactics.json", "path for the generated local JSON index")
	flag.Parse()

	entries, err := load(*inputPath, *sourceURL)
	if err != nil {
		fail(err)
	}
	if err := indexer.WriteJSON(*outputPath, entries); err != nil {
		fail(err)
	}

	lean, mathlib := counts(entries)
	fmt.Printf("wrote %d tactics to %s (%d Lean, %d mathlib or ecosystem)\n",
		len(entries), *outputPath, lean, mathlib)
}

func load(inputPath, sourceURL string) ([]catalog.Entry, error) {
	if inputPath != "" {
		file, err := os.Open(inputPath)
		if err != nil {
			return nil, fmt.Errorf("open input: %w", err)
		}
		defer file.Close()
		return indexer.Parse(file, sourceURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return indexer.Download(ctx, &http.Client{Timeout: time.Minute}, sourceURL)
}

func counts(entries []catalog.Entry) (lean, mathlib int) {
	for _, entry := range entries {
		if entry.Source == "Lean" {
			lean++
		} else {
			mathlib++
		}
	}
	return lean, mathlib
}

func fail(err error) {
	if err == io.EOF {
		err = fmt.Errorf("empty input")
	}
	fmt.Fprintln(os.Stderr, "rugtac-index:", err)
	os.Exit(1)
}

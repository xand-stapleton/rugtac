// Command rugtac fuzzy-searches a local documentation index in the terminal.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"rugtac/internal/catalog"
	"rugtac/internal/search"
	"rugtac/internal/tui"
)

func main() {
	dataPath := flag.String("data", catalog.DefaultPath(), "read documentation entries from this local JSON file")
	scope := flag.String("scope", "all", "search all, core, or one library (for example: data)")
	printOnly := flag.Bool("print", false, "print matching docs instead of opening the TUI")
	flag.Parse()

	entries, err := catalog.Load(*dataPath)
	if err != nil {
		fail(err)
	}
	query := strings.Join(flag.Args(), " ")
	scopedEntries := catalog.Filter(entries, *scope)

	if *printOnly {
		printMatches(scopedEntries, query)
		return
	}

	program := tea.NewProgram(tui.New(entries, query, *scope))
	if _, err := program.Run(); err != nil {
		fail(err)
	}
}

func printMatches(entries []catalog.Entry, query string) {
	results := search.Find(entries, query)
	if len(results) == 0 {
		fmt.Println("No matches.")
		return
	}

	for i, result := range results {
		if i == 10 {
			break
		}
		entry := result.Entry
		fmt.Printf("%s [%s]\n%s\n", entry.Name, entry.Source, entry.Description)
		if entry.Usage != "" {
			fmt.Printf("\n%s\n", entry.Usage)
		}
		if entry.URL != "" {
			fmt.Printf("\n%s\n", entry.URL)
		}
		if i+1 < len(results) && i < 9 {
			fmt.Println()
		}
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "rugtac:", err)
	os.Exit(1)
}

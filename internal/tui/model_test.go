package tui

import (
	"strings"
	"testing"

	"rugtac/internal/catalog"
)

func TestScopeControlsFilterResults(t *testing.T) {
	entries := []catalog.Entry{
		{Name: "exact", Library: "core"},
		{Name: "fin_omega", Library: "data"},
		{Name: "ring", Library: "tactic"},
	}
	model := New(entries, "", "all")

	if got := model.currentScope(); got != "all" {
		t.Fatalf("initial scope = %q", got)
	}
	model.cycleScope(1)
	if got := model.currentScope(); got != "core" {
		t.Fatalf("second scope = %q", got)
	}
	if len(model.results) != 1 || model.results[0].Entry.Name != "exact" {
		t.Fatalf("core results = %#v", model.results)
	}
	model.cycleScope(1)
	if got := model.currentScope(); got != "data" {
		t.Fatalf("library scope = %q, want data", got)
	}
	model.cycleLibrary(1)
	if got := model.currentScope(); got != "tactic" {
		t.Fatalf("next library = %q, want tactic", got)
	}
}

func TestNoColorViewContainsNoANSI(t *testing.T) {
	model := New([]catalog.Entry{{Name: "exact", Library: "core"}}, "exact", "all")
	model.color = false

	if view := model.View().Content; strings.ContainsRune(view, '\x1b') {
		t.Fatalf("NO_COLOR view contains an ANSI escape: %q", view)
	}
}

func TestColorViewContainsANSI(t *testing.T) {
	model := New([]catalog.Entry{{Name: "exact", Library: "core"}}, "exact", "all")
	model.color = true

	if view := model.View().Content; !strings.ContainsRune(view, '\x1b') {
		t.Fatal("color view contains no ANSI escape")
	}
}

func TestLibraryPickerFuzzySelectsAndPreservesTacticQuery(t *testing.T) {
	entries := []catalog.Entry{
		{Name: "fin_omega", Library: "data"},
		{Name: "ring", Library: "tactic"},
	}
	model := New(entries, "ring", "data")

	model.openLibraryPicker()
	model.setLibraryQuery("tactc")
	if len(model.libraryResults) == 0 || model.libraryResults[0] != "tactic" {
		t.Fatalf("library results = %#v", model.libraryResults)
	}
	model.closeLibraryPicker(true)

	if got := model.currentScope(); got != "tactic" {
		t.Fatalf("selected library = %q", got)
	}
	if model.query != "ring" {
		t.Fatalf("tactic query = %q, want ring", model.query)
	}
	if len(model.results) != 1 || model.results[0].Entry.Name != "ring" {
		t.Fatalf("tactic results = %#v", model.results)
	}
}

package search

import (
	"testing"

	"rugtac/internal/catalog"
)

func TestFindExactNameFirst(t *testing.T) {
	entries := []catalog.Entry{
		{Name: "container/ring", Source: "Go", Description: "A circular ring."},
		{Name: "ring_nf", Source: "mathlib", Description: "Normal form."},
		{Name: "ring", Source: "mathlib", Description: "Prove polynomial identities."},
	}

	results := Find(entries, "ring")
	if len(results) != 3 {
		t.Fatalf("Find() returned %d results, want 3", len(results))
	}
	if got := results[0].Entry.Name; got != "ring" {
		t.Fatalf("first result = %q, want ring", got)
	}
}

func TestFindMatchesTermsAcrossFields(t *testing.T) {
	entries := []catalog.Entry{
		{Name: "abel_nf", Description: "Normalizes expressions in abelian groups."},
		{Name: "ring_nf", Description: "Normalizes commutative rings."},
	}

	results := Find(entries, "abelian nf")
	if len(results) != 1 || results[0].Entry.Name != "abel_nf" {
		t.Fatalf("Find() = %#v, want only abel_nf", results)
	}
}

func TestFindToleratesTypos(t *testing.T) {
	entries := []catalog.Entry{{Name: "linarith", Description: "Linear arithmetic."}}

	results := Find(entries, "linarth")
	if len(results) != 1 || results[0].Entry.Name != "linarith" {
		t.Fatalf("Find() = %#v, want linarith", results)
	}
}

func TestFindRequiresEveryTerm(t *testing.T) {
	entries := []catalog.Entry{
		{Name: "ring", Description: "Polynomial normalization."},
		{Name: "abel_nf", Description: "Abelian normalization."},
	}

	if results := Find(entries, "ring abelian"); len(results) != 0 {
		t.Fatalf("Find() returned %#v, want no results", results)
	}
}

func TestFindRejectsLettersScatteredAcrossAParagraph(t *testing.T) {
	entries := []catalog.Entry{
		{Name: "induction", Description: "Applies a principle and creates one goal for each case."},
	}

	if results := Find(entries, "ring"); len(results) != 0 {
		t.Fatalf("Find() returned %#v, want no results", results)
	}
}

func TestGeneratedIndexFindsAbelNF(t *testing.T) {
	entries, err := catalog.Load("../../data/tactics.json")
	if err != nil {
		t.Fatal(err)
	}

	results := Find(entries, "abelian nf")
	if len(results) == 0 || results[0].Entry.Name != "abel_nf" {
		t.Fatalf("first result = %#v, want abel_nf", results)
	}
}

func TestFindStringsFuzzyRanksLibraries(t *testing.T) {
	libraries := []string{"category-theory", "data", "measure-theory", "tactic"}

	results := FindStrings(libraries, "categry")
	if len(results) == 0 || results[0] != "category-theory" {
		t.Fatalf("FindStrings() = %#v, want category-theory first", results)
	}
}

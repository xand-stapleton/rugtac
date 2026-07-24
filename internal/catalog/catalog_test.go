package catalog

import "testing"

func TestLocalCatalogHasRequiredExamples(t *testing.T) {
	entries, err := Load("../../data/tactics.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 400 {
		t.Fatalf("local catalog has %d entries; expected a complete generated index", len(entries))
	}

	want := map[string]bool{"ring": false, "abel_nf": false}
	for _, entry := range entries {
		if entry.Library == "" {
			t.Errorf("%q has no library", entry.Name)
		}
		if entry.Source == "Go" {
			t.Errorf("unexpected Go entry %q", entry.Name)
		}
		if _, exists := want[entry.Name]; exists {
			want[entry.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("local catalog does not contain %q", name)
		}
	}
}

func TestLibrariesAndFilter(t *testing.T) {
	entries := []Entry{
		{Name: "exact", Library: "core"},
		{Name: "fin_omega", Library: "data"},
		{Name: "ring", Library: "tactic"},
	}

	libraries := Libraries(entries)
	if len(libraries) != 2 || libraries[0] != "data" || libraries[1] != "tactic" {
		t.Fatalf("Libraries() = %#v", libraries)
	}
	filtered := Filter(entries, "data")
	if len(filtered) != 1 || filtered[0].Name != "fin_omega" {
		t.Fatalf("Filter(data) = %#v", filtered)
	}
	if got := Filter(entries, "all"); len(got) != len(entries) {
		t.Fatalf("Filter(all) returned %d entries", len(got))
	}
}

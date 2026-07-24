package indexer

import (
	"strings"
	"testing"
)

func TestParseExtractsDocsMetadataAndVariants(t *testing.T) {
	const page = `<html><body><main>
<div id="Mathlib.Tactic.Abel.abel"><div id="userName-abel">
  <h2>abel</h2>
  <p><code>abel</code> solves equations in abelian groups.
     <code>abel_nf</code> rewrites expressions into normal form.</p>
  <pre><code>example : a + b = b + a := by abel</code></pre>
  <dl><dt>Tags:</dt><dd></dd><dt>Defined in module:</dt>
      <dd><a href="./Mathlib/Tactic/Abel.html#abel">Mathlib.Tactic.Abel</a></dd></dl>
</div></div>
<div id="Lean.Parser.Tactic.exact"><div id="userName-exact">
  <h2>exact</h2><p><code>exact h</code> closes the goal.</p>
  <dl><dt>Defined in module:</dt>
      <dd><a href="./Lean/Parser/Tactic.html#exact">Lean.Parser.Tactic</a></dd></dl>
</div></div>
</main></body></html>`

	entries, err := Parse(strings.NewReader(page), DefaultURL)
	if err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]int)
	for i, entry := range entries {
		byName[entry.Name] = i
	}

	for _, name := range []string{"abel", "abel_nf", "exact"} {
		if _, exists := byName[name]; !exists {
			t.Fatalf("missing %q in %#v", name, entries)
		}
	}
	abel := entries[byName["abel"]]
	if abel.Source != "mathlib" || abel.Module != "Mathlib.Tactic.Abel" || abel.Library != "tactic" {
		t.Fatalf("abel metadata = %#v", abel)
	}
	if !strings.Contains(abel.URL, "Mathlib/Tactic/Abel.html") {
		t.Fatalf("abel URL = %q", abel.URL)
	}
	if !strings.Contains(abel.Usage, "example") {
		t.Fatalf("abel usage = %q", abel.Usage)
	}
	abelNF := entries[byName["abel_nf"]]
	if !contains(abelNF.Aliases, "abelian nf") {
		t.Fatalf("abel_nf aliases = %#v", abelNF.Aliases)
	}
	exact := entries[byName["exact"]]
	if exact.Source != "Lean" || exact.Library != "core" {
		t.Fatalf("exact metadata = %#v", exact)
	}
}

func TestParseRejectsUnrelatedHTML(t *testing.T) {
	if _, err := Parse(strings.NewReader("<html><body>nothing</body></html>"), DefaultURL); err == nil {
		t.Fatal("Parse() succeeded, want an error")
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

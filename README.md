# rugtac

`rugtac` is a tiny, local fuzzy searcher for mathlib and Lean
tactics described on the [Lean Community Tactics List](https://leanprover-community.github.io/mathlib4_docs/tactics.html).
Type a few characters and rugtac shows the best matching documentation. The
generated index is the ordinary local file `data/tactics.json`, so searching
never needs the network and updating the docs never requires rebuilding the
binary.

For example, `ring` puts the mathlib `ring` tactic first. `abelian nf` finds
`abel_nf` because search words may match the name, aliases, summary, or full
description independently.

## Compiling and Running

Step 0. Ensure you have an up-to-date Go installation, e.g. from [here](https://go.dev/dl/).



### Building and Installing (Recommended)
Or use the Makefile:

```sh
make build
make check
make index
```

### Running on-the-fly
To use Go's interpreted-language like mode:
```sh
go run ./bin/rugtac
```


### Rugtac quick start
You can start with a query or print results without a TUI:

```sh
go run ./bin/rugtac -- "abelian nf"
go run ./bin/rugtac -print -- ring
go run ./bin/rugtac -scope core -print -- exact
go run ./bin/rugtac -scope data -print -- omega
```

or inside the TUI with Charm's Bubble Tea:

- type to search;
- press `tab` or `shift+tab` to switch between all tactics, Lean core, and one
  specific library;
- when library scope is active, press `enter` to open the library picker, type
  to fuzzy-filter names such as `data`, `category-theory`, or `tactic`, then
  press `enter` again to confirm;
- `←`/`→` remain available as a quick way to step through libraries;
- use `↑`/`↓` (or `ctrl+p`/`ctrl+n`) to select a result;
- use `ctrl+u` to clear and `esc` to quit.

The interface uses a small ANSI colour palette without another UI dependency.
Set [`NO_COLOR`](https://no-color.org/) to disable it.

To build the binary:

```sh
go build -o rugtac ./bin/rugtac
./rugtac
```

To install both the executable and its local index for your user:

```sh
make install
```

This defaults to `~/.local/bin/rugtac` and
`~/.local/share/rugtac/tactics.json`. Ensure `~/.local/bin` is on `PATH`.
Override the prefix when needed:

```sh
make install PREFIX=/usr/local
```

To remove the installed binary and index:

```sh
make uninstall
```

Use the same `PREFIX` supplied during installation when overriding the default.

## Refresh the tactic index

The checked-in index is generated from mathlib's
[complete tactic index](https://leanprover-community.github.io/mathlib4_docs/tactics.html).
That page includes both Lean's built-in tactics and the tactics available
through mathlib. To download the current page and convert it to local JSON:

```sh
make index
```

The generator is a separate command. `rugtac-index` uses the network;
`rugtac` does not:

```sh
go run ./bin/rugtac-index -out data/tactics.json
```

For reproducible or offline conversion of an already-downloaded page:

```sh
go run ./bin/rugtac-index \
  -input tactics.html \
  -out data/tactics.json
```

## Use another local index

Run the command from the repository root so it finds `data/tactics.json`, or
pass an explicit path from anywhere:

```sh
rugtac -data /path/to/tactics.json
```

The index is a JSON array using this intentionally small schema:

```json
[
  {
    "name": "my_tactic",
    "source": "my project",
    "summary": "One-line explanation.",
    "description": "The longer searchable documentation.",
    "usage": "by\n  my_tactic",
    "url": "https://example.test/docs",
    "module": "Mathlib.Tactic.Example",
    "library": "tactic",
    "aliases": ["words people may search"]
  }
]
```

The generated data lives in [`data/tactics.json`](data/tactics.json). It is
checked into the repository so the searcher works offline immediately.

## Design

There are three direct dependencies:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) supplies only the
  terminal event loop. Input and rendering stay in one short model.
- [fuzzy](https://github.com/sahilm/fuzzy) supplies Unicode-aware ranking and
  has no dependencies of its own.
- [x/net/html](https://pkg.go.dev/golang.org/x/net/html) parses the generated
  documentation using the HTML5 algorithm.

The non-UI behavior is split into three small packages: `catalog` loads and
validates JSON, `search` ranks it, and `indexer` converts doc-gen4's HTML. Run
all tests with `go test ./...`.


### Naming

The naming of Rugtac was rather silly. I wanted a unique name which was descriptive and memorable. Rugtac is a _fuzzy_ finder, rugs are _fuzzy_, and tac is simply a shortening on _tactic_.

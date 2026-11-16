# AGENTS.md

Guidance for AI coding agents working in this repository. `CLAUDE.md`
imports this file (`@AGENTS.md`), so every agent reads the same
instructions. See [README.md](README.md) / [README.pt-BR.md](README.pt-BR.md)
for the user-facing overview.

## What this is

`biblia` is a command-line Bible lookup for a Portuguese-speaking
reader: verses in several translations side by side, ready-to-paste
output for documents (including the `// conferido` + `#bref` lines the
author's EBD lesson project checks), an editor filter mode, and an
interlinear of the original Hebrew/Greek with extended Strong numbers.

Two data sources, with different rules:

- **Verse text: Bíblia Online, through its MCP server**
  (`https://mcp.biblebox.com/pt`, announced in the site's `llms.txt`).
  The chapter pages load their text from `/api/`, which `robots.txt`
  closes to automated clients, so **never scrape the HTML**. The server
  returns at most 5 verses per call and asks that longer passages send the
  reader to the site: **one call per (reference, translation), never
  chunked**. Whatever the server leaves out is reported as missing, with
  the chapter link. The translations have distinct rights holders: text
  is cached locally only as the reader looks it up, never bulk-filled.
- **Original language, lexicon and grammar legends: STEPBible** (TAHOT,
  TAGNT, TBESH, TBESG, TEGMC, TEHMC; [www.STEPBible.org](https://www.stepbible.org),
  Tyndale House, Cambridge, CC BY 4.0). Downloaded on demand (`biblia
  dados baixar`) at a pinned commit into the reader's data directory and
  indexed into SQLite. STEPBible asks that the data not be
  redistributed: **never vendor, commit or embed these files** (see
  [docs/SOURCES.md](docs/SOURCES.md)).
- **Dictionaries and parallel passages: United Bible Societies**
  (ubsicap/ubs-open-license, CC BY-SA 4.0): the Hebrew dictionary
  (Portuguese edition, partly machine-translated and flagged so per
  sense), the Greek one (no Portuguese edition; Spanish by default) and
  the Paratext parallel passages. Same handling as STEPBible: downloaded
  on demand into `ubs.db`, never committed.

## Architecture

```text
internal/books/         the 66-book canon: PT name/abbreviation, site slug, STEPBible code, chapters
internal/ref/           parses "jo 3.16", "1co13:4-7", "Rm 5.8; 6.23", "Jd 3"; prints "1 Co 13.4-7"
internal/bibliaonline/  MCP client (JSON-RPC over HTTP, SSE answer) + parser of its text answers
internal/cache/         SQLite cache of looked-up verses (including verses a translation omits)
internal/lookup/        cache + at most one server call; partial/absent/stale bookkeeping
internal/render/        terminal view (single, side-by-side, interlinear, lexicon) and templates
internal/step/          STEPBible download, indexing and queries (words, lexicon, grammar legend, versification)
internal/ubs/           UBS dictionaries (senses per Strong number and verse) and parallel passages
internal/grammar/       grammar legend terms → Portuguese description
internal/download/      GitHub downloads pinned to a commit (shared by step and ubs)
internal/config/        XDG paths and ~/.config/biblia/config.json
cmd/biblia/             CLI: lookup, filtro, strong, versoes, formatos, dados, cache
editors/sublime/        Sublime Text plugin (calls the binary)
editors/vscode/         VS Code extension, plain JS, no build step (calls the binary)
```

The dependency graph is enforced by [.go-arch-lint.yml](.go-arch-lint.yml)
(`make arch-lint`): `bibliaonline`, `config`, `download` and `grammar`
import nothing from this module; `step` and `ubs` import only
`download` (they never import `books`: book codes and numbers come from
their own files); only `cmd` wires everything together.

## Non-obvious decisions

- **Versification.** References are Portuguese (Almeida) numbering. The
  site uses it natively. STEPBible's primary reference is the English
  (NRSV/KJV) numbering, which Almeida shares in practically every place
  (Psalm titles are verse 0 there, not verse 1). The Hebrew/Greek-edition
  number is kept in `Word.AltRef` and shown as "no original: 51.12". No
  conversion table exists; add one only with a verified counterexample.
- **Absent vs. cut.** When the server skips a requested verse, it is
  *absent* from that translation (Mt 17.21 in the NVI) unless it lies past
  the cut of a partial answer. Absent verses are cached too, so they are
  never asked again.
- **Joining STEPBible words to UBS senses.** The key is the plain
  four-digit Strong number: STEPBible's letter suffixes (`H2603A`) and
  UBS's (`H7451b`) mean different things, and UBS files Aramaic words as
  `A4430` where Strong numbers them `H4430`. The sense is chosen at verse
  level only (UBS counts words differently): Old Testament words match
  through their Hebrew reference (`Word.AltRef`), New Testament words
  through the translations' numbering. Measured on the full data: 90% of
  OT and 92% of NT words find an entry, and 99% / 91% of those find the
  sense for their verse; the gaps are mostly function words, which fall
  back to STEPBible's English gloss, marked `(en)`.
- **Dictionary preferences.** `idioma` picks the defaults
  (`ubs.Defaults`: the reader's language, then the closest one, then
  English, per testament); `dicionarios` overrides them; `step` always
  ends the chain.
- **Grammar** is translated term by term (`grammar.portuguese`); a term
  missing from the table makes the whole description fall back to the raw
  code. `BIBLIA_STEP_DB=<step.db> go test ./internal/grammar/` checks
  every code of a real index.
- **Parallel passages** are in the original numbering; `cmd` converts
  through `step.OriginalVerse`/`TranslationVerse` (Jl 2.28 ↔ 3.1). `-p`
  only lists (no server call); `-pt` fetches each passage like a lookup.
- **Parallels in the paste formats** travel as template data, not as
  extra output: `render.Data.Parallels` on the passage asked about,
  `render.Data.ParallelOf` on each parallel's text (`render.NewParallelOf`
  writes the relation from that text's point of view). The built-in `typ`
  keeps them in comments and never between `// conferido` and `#bref`,
  where the lesson project's checker expects the two lines together.
- **"jo" is John, "jó" is Job.** The accent decides; see
  `books.accented`. Tests guard every spelling against shadowing.
- **Single-chapter books** print as "Jd 1.3" (the lesson project's
  style); input "Jd 3" means verse 3.
- **Lexicon entries and their relations.** A TBESH/TBESG row gives the
  entry's own extended number and its relation to another entry in the
  second column ("H5162H = a Meaning of"), and that other entry's number
  in the third (H5162G); `step.Entry` keeps them as `EStrong`,
  `Relation` and `Target`. Reading the third column as the entry's own
  number showed H5162H as H5162G. Indexes built before this change lack
  the columns; `Store.Outdated` reports it and queries degrade to the old
  reading.
- **Lexicon lookup order** (`step.Store.Lookup`): the extended number the
  tagged texts use ("H2603A", with the entries tied to it: H2589 "a form
  of" H2603A, H2604 "in Aramaic of" H2603A), then the number as the
  lexicon files it ("H7451b", "H5162"), then every entry with the same
  four digits.
- **SENHOR.** The printed ARA sets the divine name in small caps
  ("SENHOR"); the server returns "Senhor". There is no way to tell from
  the text which "Senhor" stands for YHWH, so the tool does not try.
- **Filter mode never loses text.** A line that is not a reference passes
  through unchanged and silently; a reference that fails is kept and
  reported on stderr. The exception is `filtro -p`, which answers each
  reference with the text of its parallels and writes nothing else:
  editors insert that output after the selection, so echoing the input
  would duplicate it.
- **Editor format `auto`.** The plugins pick the paste format from the
  file (extension map in their settings, then the editor's language) and,
  for any other file, pass no `-f`, so `formato_filtro` decides.

## Commands

```sh
make build       # bin/biblia
make install     # ~/.local/bin/biblia
make check       # lint + arch-lint + tests with the race detector (what CI runs)
make lint-md     # markdown lint (Node.js >= 20)
make sublime     # symlink editors/sublime into Sublime's Packages/Biblia
make vscode      # package and install the VS Code extension
```

Tests never touch the network: the MCP server is faked (`lookup`) or
served by `httptest` (`bibliaonline`), and STEPBible rows are inline
fixtures copied from the real files. To try the real thing without
touching the reader's own cache, point `XDG_CACHE_HOME`,
`XDG_DATA_HOME` and `XDG_CONFIG_HOME` at a scratch directory.

golangci-lint is pinned to v2.15.0 in the Makefile: v2.13.x cannot read
Go 1.27 export data.

## Releases

Three independent tags, each with its own version (steps, in Portuguese:
[docs/PUBLICAR.md](docs/PUBLICAR.md)):

- `v0.1.0`: the CLI. `.github/workflows/release.yml` runs GoReleaser
  (`.goreleaser.yaml`) for Linux, macOS and Windows. The workflow matches
  `v[0-9]*`, because `vscode-v...` also starts with a v, and passes
  `GORELEASER_CURRENT_TAG`, because plugin tags may sit on the same
  commit; the Makefile's `VERSION` uses `git describe --match 'v[0-9]*'`
  for the same reason.
- `vscode-v0.2.0`: the VS Code extension, published by
  `.github/workflows/vscode.yml` through OpenID Connect (Marketplace
  Trusted Publishing, Open VSX trusted publishers), with no stored token.
  The tag must equal `editors/vscode/package.json`'s version.
- `sublime-0.2.0`: created by `make sublime-release V=...`, which pushes
  `editors/sublime` (git subtree split) to `Casara/biblia-sublime`, where
  Package Control reads it. Package Control rules shape the package: no
  key bindings enabled by default (the suggestions live in
  `Example.sublime-keymap`, which Sublime does not load), every command
  in the palette, settings and key bindings opened with `edit_settings`,
  the package name written as "Biblia", without the accent, wherever the
  user types it or Sublime resolves it (palette, menu,
  `Biblia.sublime-settings`, the `Packages/Biblia` folder), `.gitattributes` export-ignore for files
  of no use inside Sublime.

Planned work, in Portuguese for the author: [docs/ROADMAP.md](docs/ROADMAP.md).

## Conventions

Same as the author's other Go projects (numbertext-go's
`docs/coding-style.md`):

- gofmt/goimports; golangci-lint with `default: all`, exceptions only in
  `.golangci.yml`, each with its reason.
- Errors: wrap with context and `%w` (`"step: reading lexicon: %w"`);
  sentinel errors for conditions a caller matches with `errors.Is`; no
  inline `if err := ...; err != nil`.
- Long or growing calls in vertical format; comments explain purpose,
  limits and decisions, not the function name.
- Identifiers and comments in English; everything the reader sees
  (messages, help, flags' descriptions) in Portuguese.
- Conventional Commits (`feat:`, `fix:`, `docs:`, `chore(deps):`).

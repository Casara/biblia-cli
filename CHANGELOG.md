# Changelog

All notable changes to this project are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
the versioning follows [Semantic Versioning](https://semver.org/).

The editor plugins keep their own changelog:
[VS Code](editors/vscode/CHANGELOG.md) and
[Sublime Text](editors/sublime/CHANGELOG.md).

## [Unreleased]

## [0.1.0] - 2026-11-13

### Added

#### Verse lookup

- Verse lookup through Bíblia Online's MCP server, in any of the site's
  translations (Portuguese, English, Spanish and others), with a local
  SQLite cache of what was looked up and an `-offline` mode.
- Reference parser for Portuguese input (`jo 3.16`, `1co13:4-7`,
  `Rm 5.8; 6.23`, `Jd 3`, Roman numerals, full names and unambiguous
  prefixes).
- Honest handling of the server's 5-verse limit (partial results link to
  the chapter) and of verses a translation omits.

#### Output

- Terminal view and side-by-side comparison (`-v ara,naa,nvi`).
- Paste formats `linha`, `md` and `typ` (the `// conferido` + `#bref`
  pair the EBD lesson project checks), plus user templates in
  `~/.config/biblia/config.json`, with every template field and function
  documented in the Portuguese README.

#### Original languages

- Interlinear of the original Hebrew/Greek with extended Strong numbers
  (`-s`), and lexicon lookup (`biblia strong`), from STEPBible data.
- Dictionaries of the United Bible Societies (CC BY-SA 4.0), chosen in
  the config file (`idioma`, `dicionarios`): the Hebrew dictionary in
  Portuguese, the Greek one in Spanish by default. The interlinear shows
  the sense the dictionary assigns to the word in that verse, and marks
  the senses UBS flags as machine-translated (`[IA]`); `biblia strong`
  lists every sense, and `-glosa` searches the dictionaries' glosses.
- Grammar of each interlinear word decoded into Portuguese, from
  STEPBible's morphology legends (TEGMC, TEHMC).
- Lexicon entries tied to another one show the relation in Portuguese
  (`H5162H … (um sentido de H5162G)`), and looking up an extended number
  also lists the entries tied to it.

#### Parallel passages

- Parallel passages (`-p`, or `-pt` with their text) from the UBS
  Paratext database: Old Testament quotations in the New, Gospel
  parallels, Kings and Chronicles, converting the Hebrew verse numbering
  to the translations'.
- The paste formats say what each text is: `typ` in comments that stay
  out of the PDF (`// Citado no NT: Rm 1.17 · Gl 3.11 · Hb 10.38`,
  `// paralelo: Rm 1.17 cita Hc 2.4`), `md` in an italic line, `linha`
  inside its parentheses.
- Template fields `.Parallels` (the list, on the passage asked about) and
  `.ParallelOf` (`.Ref`, `.Kind`, `.Relation`, on each parallel's text),
  and the `parallels` template function.

#### Editors

- `filtro` mode for editors (`-p`: only the text of the parallel
  passages), with a Sublime Text plugin and a VS Code extension.
- Plugin commands to show a reference's parallel passages and to insert
  their text after it; the inserted text's format follows the file type
  (`auto`: `.md` → `md`, `.typ` → `typ`).

#### Distribution

- Release binaries for Linux, macOS and Windows (amd64 and arm64), built
  by GoReleaser when a CLI tag (`v0.1.0`) is pushed; `go install` also
  works.
- The data and indexes live in each system's own folder (Application
  Support on macOS, `%LocalAppData%` on Windows); CI runs the tests on the
  three systems.
- The editor plugins are published on their own: the VS Code extension to
  the Visual Studio Marketplace and Open VSX (`vscode-v*` tags, through
  OpenID Connect), the Sublime package through a mirror repository for
  Package Control (`make sublime-release`). Steps in `docs/PUBLICAR.md`.

#### Data

- `biblia dados baixar [step|ubs]` downloads the STEPBible and UBS data
  pinned to a commit and indexes it; `biblia dados status` shows the
  provenance, the dictionaries in use, the space the downloaded files take
  and whether an index predates the current version.
- `-enxuto` deletes the downloaded files once they are indexed (about
  75 MB kept instead of 220 MB).

[Unreleased]: https://github.com/Casara/biblia-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Casara/biblia-cli/releases/tag/v0.1.0

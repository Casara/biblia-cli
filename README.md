# biblia

Bible verse lookup from the command line or from inside an editor, in
several translations at once, with the original Hebrew/Greek, Strong's
numbers, dictionary senses and parallel passages when you want to dig
deeper. Built around Portuguese input and translations, but any
translation the source carries works; the program's messages and output
labels are in Portuguese.

*Leia isto em [português (Brasil)](README.pt-BR.md).*

```console
$ biblia jo 3.16 -v ara,naa,nvi
Jo 3.16
  ARA  Porque Deus amou ao mundo de tal maneira que deu o seu Filho unigênito, ...
  NAA  Porque Deus amou o mundo de tal maneira que deu o seu Filho unigênito, ...
  NVI  — Porque Deus tanto amou o mundo que deu o seu Filho Unigênito, ...
```

## Install

Pick one:

- **Ready-made binary (no Go needed):** download your system's file from
  the [releases page](https://github.com/Casara/biblia-cli/releases/latest)
  (`linux`, `darwin` for macOS or `windows`; `amd64` or `arm64`), unpack
  it and put `biblia` (`biblia.exe` on Windows) in a folder on your
  `PATH`, such as `~/.local/bin`.
- **With Go 1.26 or newer:**
  `go install github.com/casara/biblia-cli/cmd/biblia@latest`.
- **From the source:** `make install`, which installs
  `~/.local/bin/biblia`.

Then, optionally:

```sh
biblia dados baixar   # original texts, lexicons, dictionaries, parallels (~220 MB)
                      # with -enxuto, deletes the downloaded files once indexed (~75 MB)
```

## Usage

```sh
biblia jo 3.16                    # default translation (ara, or the config file's)
biblia jo 3.16 -v ara,naa,nvi     # side by side
biblia rm 8.28-30 -v kjv,rves     # English, Spanish: any id from "biblia versoes -todas"
biblia "Rm 5.8; 6.23"             # several references; the book carries over
biblia jo 3.16 -s                 # + original words: Strong number, sense, grammar
biblia hc 2.4 -p                  # parallel passages (-pt: with their text)
biblia strong H2617               # every sense of a Strong number
biblia strong G0025 -c            # + each sense's domain and the STEPBible entry
biblia strong -tr agape           # by transliteration (diacritics optional)
biblia strong -glosa amor         # by gloss, in the dictionaries' language
biblia jo 3.16 -f typ             # paste-ready, with the "// conferido" line
biblia jo 3.16 -offline           # cache only
```

References are typed the Brazilian way: `jo 3.16`, `Jo 3:16`,
`1co13.4-7`, `I Co 13.4`, `Sl 23.1-3,6`, `Rm 8` (whole chapter), `Jd 3`
(single-chapter book), full names (`Gênesis`, `genesis`) or an
unambiguous prefix (`roman`). `jo` is John; `jó` (accented) or `job` is
Job.

### Original languages, dictionaries and parallels

`-s` shows each word of the original with its Strong number, the gloss of
the sense the dictionary gives it **in that verse**, and its grammar
spelled out:

```text
  Sl 51.1 · hebraico (no original: 51.3) · glosas: UBS hebraico (pt)
    חָנֵּ֣נִי      choneni      H2603A  ter boa vontade · verbo, qal, imperativo, ...
    כְּחַסְדֶּ֑ךָ  kechasDekha  H2617A  bondade [IA] · preposição + substantivo, ...
```

- **Dictionaries:** those of the United Bible Societies (UBS). The Hebrew
  one exists in Portuguese; the Greek one does not, and comes in Spanish,
  the closest language. Part of the Portuguese Hebrew dictionary was
  machine-translated, and UBS itself flags each such sense: the program
  shows `[IA]`. Words the dictionary does not cover (mostly prepositions
  and particles) keep STEPBible's English gloss, marked `(en)`.
- **Grammar:** STEPBible's codes (`V-AAI-3S`, `HR/Ncfsa`) translated term
  by term.
- **Parallels (`-p`):** from the UBS parallel passages database: Old
  Testament quotations in the New, Gospel parallels, Kings and
  Chronicles. `-pt` also brings their text. The Hebrew verse numbering
  (Jl 3.1) is converted to the translations' (Jl 2.28). In the paste
  formats, the list and each text's relation go into the text itself (see
  *Parallels in the formats*, below).

### Formats

| Format | Output |
| --- | --- |
| `texto` | the terminal view (default) |
| `linha` | `texto (Jo 3.16, ARA)` on a single line |
| `md` | a Markdown quotation with the reference |
| `typ` | `// conferido: ARA · bibliaonline.com.br/ara · 2026-11-12` and `#bref("Jo 3.16", [«...»])` |

### Parallels in the formats

With `-p` or `-pt` (and in the editors' command that inserts the
parallels), the paste formats say what each text is:

```text
// conferido: ARA · bibliaonline.com.br/ara · 2026-11-12
#bref("Hc 2.4", [«Eis o soberbo! ... mas o justo viverá pela sua fé.»])
// Citado no NT: Rm 1.17 · Gl 3.11 · Hb 10.38

// paralelo: Rm 1.17 cita Hc 2.4
// conferido: ARA · bibliaonline.com.br/ara · 2026-11-12
#bref("Rm 1.17", [«visto que a justiça de Deus se revela ...»])
```

| Format | On the passage asked about | On each parallel |
| --- | --- | --- |
| `typ` | a comment with the list, after the `#bref` | a comment with the relation, before the `// conferido` |
| `md` | an italic line with the list | the relation on the reference line (`— Rm 1.17 (ARA) · *cita Hc 2.4*`) |
| `linha` | nothing (it is a single line) | the relation inside the parentheses (`(Rm 1.17, ARA; cita Hc 2.4)`) |

In `typ`, everything stays in comments: none of it reaches the PDF, and
the `// conferido` line stays right above the `#bref`. To show the
relation in the document, use a template of your own (below).

### Your own templates

A template is a format of your own, written as a
[Go template](https://pkg.go.dev/text/template) under the `modelos` key
of the config file and used by name (`-f citacao`; in the editors, in the
`formato` option). A template named after one of the program's formats
(`linha`, `md`, `typ`) replaces it. In the example, `#nota` would be a
function of your Typst document that shows the relation in the PDF.

```json
{
  "modelos": {
    "citacao": "“{{.Text}}” ({{.Ref}}, {{.Bible}}){{with .ParallelOf}} — {{.Relation}}{{end}}",
    "booklet": "{{with .ParallelOf}}#nota[{{.Relation}}]\n{{end}}// conferido: {{.Bible}} · bibliaonline.com.br/{{.BibleID}} · {{.Date}}\n#bref(\"{{.Ref}}\", [«{{typ .Text}}»])"
  }
}
```

**Text fields:**

| Field | What it is | Example |
| --- | --- | --- |
| `.Ref` | the reference of the text obtained | `Gn 3.1-5` |
| `.Requested` | the reference asked for (differs from `.Ref` in a partial passage) | `Gn 3.1-6` |
| `.Bible` / `.BibleID` | the translation, upper case / as the site names it | `ARA` / `ara` |
| `.Text` | the verses joined, without numbers | |
| `.Verses` | the verses, each with `.Number` and `.Text` | |
| `.URL` | the link to the passage (to the chapter, if partial) | |
| `.Date` | when the text came from the site | `2026-11-12` |
| `.Partial` / `.Missing` | whether verses are missing / which | `true` / `6` |
| `.Absent` | verses the translation does not have | `21` |

**Parallel fields** (only with `-p`, `-pt` or `filtro -p`):

| Field | Where it appears | What it is |
| --- | --- | --- |
| `.Parallels` | on the passage asked about | the list of parallels; each item has `.Ref` (`Rm 1.17`) and `.Kind` |
| `.ParallelOf` | on each parallel's text | where it came from: `.Ref` (the passage asked about, `Hc 2.4`), `.Kind` and `.Relation` |

`.Kind` is the type, from the point of view of the passage asked about:
`citado no NT` (Hc 2.4 is quoted in Rm 1.17), `citação do AT` (Rm 3.4
quotes Sl 51.4) or `paralelo` (Mt 6.9 and Lc 11.2). `.Relation` says the
same from the parallel's point of view, as a ready phrase: `cita Hc 2.4`,
`citado em Rm 3.4`, `paralelo de Mt 6.9`. Outside parallels, `.Parallels`
is empty and `.ParallelOf` is absent, so `{{if .Parallels}}` and
`{{with .ParallelOf}}` are enough to test for them.

**Functions:**

| Function | What it does | Example |
| --- | --- | --- |
| `typ` | escapes Typst's special characters | `{{typ .Text}}` |
| `md` | escapes Markdown's special characters | `{{md .Text}}` |
| `parallels` | writes the list grouped by type | `{{parallels .Parallels}}` → `Citado no NT: Rm 1.17 · Gl 3.11 · Hb 10.38` |

Inside `{{with .ParallelOf}}`, the fields are the parallel's; the text is
still reachable through `$`:
`{{with .ParallelOf}}{{$.Ref}} {{.Relation}}{{end}}` → `Rm 1.17 cita Hc 2.4`.

### Configuration

`~/.config/biblia/config.json` (or the path in `BIBLIA_CONFIG`). Every
field is optional:

```json
{
  "versoes": ["ara", "naa"],
  "formato": "texto",
  "formato_filtro": "typ",
  "modelos": {
    "citacao": "“{{.Text}}” ({{.Ref}}, {{.Bible}})"
  },
  "idioma": "pt",
  "dicionarios": ["ubs-hebraico-pt", "ubs-grego-es", "step"]
}
```

`idioma` picks the default dictionaries and the language of the grammar;
`dicionarios` sets the list, in order of preference (`biblia dados
status` shows the one in use). Available: `ubs-hebraico-pt`, `-es`, `-en`,
`-fr`; `ubs-grego-es`, `-en`, `-fr`; and `step` (English, from
STEPBible). After changing the list, run `biblia dados baixar ubs`.

### Limits that come from the source

- **At most 5 verses per lookup.** It is the Bíblia Online service's
  limit. For a longer passage, the output shows the first five, says what
  is missing and gives the chapter's link. Verses looked up before come
  from the cache and do not count.
- **A verse the translation does not have** (e.g. Mt 17.21 in the NVI, or
  a number past the end of the chapter inside a range) shows as *não
  consta nesta tradução*, not as an error.
- **SENHOR.** The printed ARA sets the divine name in small caps
  (SENHOR); the site returns "Senhor". When pasting into a text that is
  checked against the printed magazine, adjust it by hand.

## In editors

Both plugins call the same binary and have the same commands. The
shortcuts come enabled in VS Code; in Sublime they are suggestions to
copy into your own keymap, because Package Control does not accept
packages that enable key bindings.

| Command | Shortcut | What it does |
| --- | --- | --- |
| **consultar** | `Ctrl+Alt+B` | shows the text, leaving the file alone |
| **inserir** | `Ctrl+Alt+Shift+B` | replaces the selected reference with its text |
| **consultar…** | (command palette) | asks for a reference and shows its text |
| **passagens paralelas** | `Ctrl+Alt+L` | shows the text and the list of parallels |
| **inserir passagens paralelas** | `Ctrl+Alt+Shift+L` | keeps the reference and inserts, on the lines after it, the text of each parallel |
| **consultar com o original** | `Ctrl+Alt+O` | opens a tab with the interlinear (`-s`) |
| **Strong…** | (command palette) | the entry of the Strong number under the cursor, or a search by transliteration or gloss |
| **consultar / inserir em outras traduções…** | (command palette) | picks the translations for that one time |

With no selection, the cursor line counts. The inserted text's format
defaults to `auto`: `.md` gets `md`, `.typ` gets `typ` (and, in a file
not saved yet, the editor's language decides); any other file uses the
config file's `formato_filtro`. The extension → format pairs are a plugin
option, and `formato` also accepts a fixed format.

- **Sublime Text:** through Package Control (package **Biblia**) or,
  from the source, `make sublime`, which links `editors/sublime` into the
  Packages folder (`~/.config/sublime-text-3` or `~/.config/sublime-text`).
  Options and shortcuts under *Preferences › Package Settings › Biblia*:
  the suggested shortcuts are on the left; copy the ones you want into
  your own keymap on the right.
- **VS Code:** through the Marketplace (extension **Bíblia
  (biblia-cli)**) or, from the source, `make vscode`, which packages and
  installs the extension (requires Node.js). Options under *Settings ›
  Bíblia*; shortcuts under *Keyboard Shortcuts*. The extension's
  `Ctrl+Alt+B` takes the place of VS Code's shortcut that toggles the
  secondary side bar.

Each plugin has its own README, with step-by-step installation of
`biblia`: [VS Code](editors/vscode/README.md) and
[Sublime Text](editors/sublime/README.md).

The shortcuts avoid those Sublime already uses (`Ctrl+Alt+P`,
`Ctrl+Alt+Shift+P`, `Ctrl+Alt+Q`) and GNOME's (`Ctrl+Alt+D`,
`Ctrl+Alt+T`).

Any other editor that can filter a selection through an external command
works: `biblia filtro -f linha` reads references on its input and returns
their text. A line that is not a reference comes back unchanged. With
`-p`, the filter returns only the text of each reference's parallel
passages, and nothing else.

## In any program (GNOME)

GNOME has no context menu for selected text; instead, global keyboard
shortcuts look up whatever is selected in any program (browser, PDF
reader, terminal): a reference (`Jo 3.16`) or a Strong number (`G5293`,
`H2617`).

| Shortcut | Reference | Strong number |
| --- | --- | --- |
| `Super+B` | shows the text in a window | shows the lexicon entry (`strong -c`) |
| `Super+Alt+B` | the same, with the original Hebrew/Greek (`-s`) | shows the lexicon entry |
| `Super+Shift+B` | copies the text, ready to paste (`formato_filtro`) | copies the entry |

Selecting is enough, no `Ctrl+C`; with nothing selected, the clipboard
counts. In a reference's window, the **Traduções…** button opens the
list of the site's translations, the current ones checked: pick one or
several (side by side) and the window reopens with them. The choice
lasts until the window closes; the next lookup goes back to
`config.json`'s. Needs `zenity` and, on Wayland, `wl-clipboard` (`xclip` on X11).
With `biblia` already on your `PATH`:

```sh
sudo apt install zenity wl-clipboard
biblia atalhos instalar
```

The shortcuts live under *Settings › Keyboard › Custom Shortcuts*, where
they can be changed; your other custom shortcuts are left alone. They
call `biblia` by its full path: if you move it, run `instalar` again.
`biblia atalhos remover` deletes them. To copy in another format, edit
the shortcut's command in GNOME, adding `-f md` (or `typ`, `linha`, a
template of yours) to `selecao copiar`.

## Where the files live

| What | Linux | macOS | Windows |
| --- | --- | --- | --- |
| preferences (`config.json`) | `~/.config/biblia/` | `~/Library/Application Support/biblia/` | `%AppData%\biblia\` |
| verses already looked up (`cache.db`) | `~/.cache/biblia/` | `~/Library/Caches/biblia/` | `%LocalAppData%\biblia\` |
| STEPBible and UBS data and their indexes | `~/.local/share/biblia/` | `~/Library/Application Support/biblia/` | `%LocalAppData%\biblia\` |

`biblia dados status` shows where the indexes are, and `biblia cache
status` where the cache is; `biblia cache limpar [versão]` empties it.
`BIBLIA_CONFIG` points to another config file, and `XDG_DATA_HOME` to
another data folder.

## Sources and credits

- **Bible text:** [Bíblia Online](https://www.bibliaonline.com.br), through
  the service the site publishes for programs. Each translation has its own
  rights holders; this program distributes no Bible text.
- **Original languages and lexicons:** data created by
  [www.STEPBible.org](https://www.stepbible.org) based on work at Tyndale
  House, Cambridge, under CC BY 4.0
  ([STEPBible-Data](https://github.com/STEPBible/STEPBible-Data)).
  Downloaded on demand, never redistributed.
- **Dictionaries and parallel passages:** United Bible Societies
  ([ubs-open-license](https://github.com/ubsicap/ubs-open-license)), under
  CC BY-SA 4.0. Downloaded on demand.

Details and terms in [docs/SOURCES.md](docs/SOURCES.md).

## License

[MIT](LICENSE) for the code. The data carries its sources' licenses.

# Bíblia (biblia-cli)

Look up Bible verses without leaving the editor: in several translations
at once, with the original Hebrew/Greek, Strong's numbers, dictionary
senses and parallel passages, and insert them into Markdown or Typst
ready to cite. Built around Portuguese references and translations
(ARA, NVI, NAA...), but English, Spanish and the other translations of
[Bíblia Online](https://www.bibliaonline.com.br) work too. The commands
and the program's output are in Portuguese.

*Leia isto em [português](https://github.com/Casara/biblia-cli/blob/main/editors/vscode/README.pt-BR.md).*

![In a Typst file, the reference Rm 8.28 becomes the "conferido" comment and the #bref quotation; in a Markdown file, a quotation with the reference][inserir]

[inserir]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/vscode-inserir.gif

## Requirement: the `biblia` program

This extension is a front end: every lookup is done by the **`biblia`**
command-line program, which has to be installed separately. Without it,
the commands report *executável não encontrado* (executable not found).

1. **Download it** from the
   [releases page](https://github.com/Casara/biblia-cli/releases/latest):
   - Linux: `biblia_<version>_linux_amd64.tar.gz` (or `arm64`);
   - macOS: `biblia_<version>_darwin_arm64.tar.gz` (Apple silicon) or
     `darwin_amd64` (Intel);
   - Windows: `biblia_<version>_windows_amd64.zip`.

   Unpack it and put `biblia` (`biblia.exe` on Windows) in a folder on
   your `PATH`, such as `~/.local/bin`.

   Or, with Go installed:
   `go install github.com/casara/biblia-cli/cmd/biblia@latest`.

2. **Check it** in a terminal: `biblia jo 3.16` should print John 3.16.

3. **Optional:** `biblia dados baixar` downloads the original-language
   texts, dictionaries and parallel passages (about 220 MB, or 75 MB with
   `-enxuto`). Without them, verse lookups work, but the parallel-passage
   commands do not.

The extension looks for `biblia` on the `PATH` and then in
`~/.local/bin` and `~/go/bin`. Anywhere else, set its full path in
**`biblia.executavel`**.

## Commands

| Command | Shortcut | What it does |
| --- | --- | --- |
| **Bíblia: consultar referência selecionada** | `Ctrl+Alt+B` | shows the text in the *Bíblia* output panel, leaving the file alone |
| **Bíblia: inserir texto no lugar da referência** | `Ctrl+Alt+Shift+B` | replaces the selected reference with its text |
| **Bíblia: consultar…** | — | asks for a reference and shows its text |
| **Bíblia: passagens paralelas da referência selecionada** | `Ctrl+Alt+L` | shows the text and the list of parallel passages (Old Testament quotations in the New, Gospel parallels...) |
| **Bíblia: inserir o texto das passagens paralelas** | `Ctrl+Alt+Shift+L` | keeps the reference and inserts, on the lines after it, the text of each parallel passage, saying what it is |
| **Bíblia: consultar com o original (hebraico/grego e Strong)** | `Ctrl+Alt+O` | opens a read-only tab with the text and, word by word, the Hebrew/Greek original, transliteration, Strong number, gloss and grammar |
| **Bíblia: Strong…** | — | the lexicon entry of the Strong number under the cursor (`G25`, `H7451b`); otherwise asks for a number, transliteration or gloss |
| **Bíblia: consultar em outras traduções…** | — | asks which translations to use this time, then works like *consultar* |
| **Bíblia: inserir texto em outras traduções…** | — | asks which translations to use this time, then works like *inserir* |

With no selection, the cursor line counts. References are typed the
Brazilian way: `Jo 3.16`, `1 Co 13.4-7`, `Rm 5.8; 6.23`, `Sl 23`.

The Strong numbers in the original tab are links: `Ctrl`+click
(`Cmd`+click on macOS) opens the entry beside it in the same group,
leaving the interlinear open; so does **Bíblia: Strong…** with the cursor
on a number. The gloss search uses the dictionaries' language:
Portuguese for Hebrew, Spanish for Greek (the UBS has no Portuguese
Greek dictionary yet). The translation picker remembers the last choice;
`biblia.versoes` stays as it is. The original and Strong commands need
`biblia dados baixar`.
`Ctrl+Alt+B` replaces VS Code's own shortcut for the secondary side bar;
change any of them under *Keyboard Shortcuts*.

![The interlinear of Jo 3.16 in a tab beside the file: Greek word, transliteration, Strong number, gloss and grammar][original]

![Ctrl+click on G0025 in the interlinear opens the lexicon entry in another tab][strong]

![The translation picker with ara and arc checked][traducoes]

[original]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/vscode-original.png
[strong]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/vscode-strong.gif
[traducoes]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/vscode-traducoes.png

## Settings

| Setting | Default | What it does |
| --- | --- | --- |
| `biblia.executavel` | `biblia` | the program, by name or full path |
| `biblia.versoes` | (empty) | translations, comma-separated (`ara,naa`); empty uses the program's config file |
| `biblia.formato` | `auto` | format of the inserted text: `auto`, `linha`, `md`, `typ` or a template of your own |
| `biblia.formatosPorExtensao` | `.md` → `md`, `.typ` → `typ` | for `auto`: file extension → format |

In `auto`, Markdown files get a quotation with the reference, Typst
files get `#bref(...)` preceded by a `// conferido: ARA · ... · date`
line, and other files use the program's `formato_filtro` setting. The
formats, the template fields and the program's config file are
documented in the
[biblia README](https://github.com/Casara/biblia-cli#readme).

## Sources and credits

- **Bible text:** [Bíblia Online](https://www.bibliaonline.com.br),
  through the service the site publishes for programs, at most 5 verses
  per lookup. Each translation has its own rights holders; neither the
  extension nor the program ships Bible text.
- **Original languages, lexicons and grammar:**
  [STEPBible](https://www.stepbible.org) (Tyndale House, Cambridge, CC BY
  4.0), downloaded on demand.
- **Dictionaries and parallel passages:** United Bible Societies
  ([ubs-open-license](https://github.com/ubsicap/ubs-open-license), CC BY-SA
  4.0), downloaded on demand.

## License

MIT. The data keeps its sources' licenses.

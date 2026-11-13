# Data sources

`biblia` ships no Bible text and no lexicon. Both come from third parties,
under the terms below, and live only on the reader's machine.

## Verse text: Bíblia Online

- Site: <https://www.bibliaonline.com.br>
- Access: the MCP server the site publishes for programs,
  `https://mcp.biblebox.com/pt` (tools `get_passage`, `search_bible`,
  `list_bibles`), announced in <https://www.bibliaonline.com.br/llms.txt>.
- Terms stated by the site (`llms.txt` and the server's own instructions):
  - the translations have distinct rights holders, and the full text is
    served only on the site's pages;
  - the server returns short excerpts (at most 5 verses per call); for
    more, send the reader to the link;
  - cite the passage URL when using the content.
- How `biblia` complies:
  - one server call per (reference, translation), never several calls to
    rebuild a longer passage;
  - output always carries the passage or chapter link (the `typ` format
    names the site and the date in its `// conferido` line);
  - the local cache holds only what the reader looked up, in their own
    `~/.cache/biblia/`; there is no bulk download;
  - `/api/` is never called: `robots.txt` closes it to automated
    clients.

## Original language and lexicons: STEPBible

- Repository: <https://github.com/STEPBible/STEPBible-Data>
- Files: TAHOT (Hebrew OT, 4 files) and TAGNT (Greek NT, 2 files) from
  `Translators Amalgamated OT+NT/`; TBESH and TBESG from `Lexicons/`;
  TEGMC and TEHMC (morphology code legends) from `Morphology codes/`.
- Credit: data created by [www.STEPBible.org](https://www.stepbible.org) based on work at Tyndale
  House, Cambridge, under CC BY 4.0.
- Terms beyond the license:
  - the repository asks users to *refer others to github.com/STEPBible as
    the source of the data* and *not to redistribute it themselves*;
  - the TBESH header notes that the brief Hebrew lexicon is based on the
    Abridged BDB by Online Bible, provided for guidance, with permission
    to be sought from Online Bible before applying the definitions in a
    project.
- How `biblia` complies:
  - `biblia dados baixar` downloads the files from GitHub at the commit
    current at that moment (recorded in the index; `biblia dados status`
    shows it) into `~/.local/share/biblia/`;
  - nothing from STEPBible is committed to this repository or embedded in
    the binary;
  - the lexicon is shown for consultation; definitions are not meant to be
    copied into published material (the EBD lesson project writes its own
    glosses and uses the lexicon only to check them);
  - the Portuguese grammar descriptions translate only the legends'
    grammatical terms, from a table written for this program.

## Dictionaries and parallel passages: United Bible Societies

- Repository: <https://github.com/ubsicap/ubs-open-license>
- License: Creative Commons Attribution-ShareAlike 4.0 International
  (CC BY-SA 4.0).
- Files used:
  - *UBS Dictionary of Biblical Hebrew* © United Bible Societies,
    adapted from the *Semantic Dictionary of Biblical Hebrew*
    © 2000–2023 United Bible Societies (`dictionaries/hebrew/JSON/`, in
    Portuguese, Spanish, English, French);
  - *UBS Dictionary of New Testament Greek* © United Bible Societies,
    2023, adapted from the *Semantic Dictionary of Biblical Greek*
    © 2018–2023, which revises J. P. Louw and E. A. Nida's
    *Greek-English Lexicon of the New Testament Based on Semantic
    Domains* © 1988, 1989 (`dictionaries/greek/JSON/`, in Spanish,
    English, French; there is no Portuguese edition);
  - *UBS Parallel Passage Database* © 2023 United Bible Societies
    (`parallel passages/ParallelPassages.xml`).
- Machine translation: part of the Portuguese Hebrew dictionary (4,552
  of 16,606 senses at the indexed commit) was machine-translated; UBS
  flags each such sense (`AIGenerated`, with human review recommended),
  and the program marks it `[IA]` wherever it shows it.
- Numbering: the dictionaries' references and the parallel passages
  follow the original text (Hebrew Sl 51.3 is the translations' Sl 51.1);
  the program converts with the Hebrew/English pairs of the STEPBible
  index.
- How `biblia` complies:
  - `biblia dados baixar ubs` downloads the configured dictionaries and
    the parallel passages at the commit current at that moment into
    `~/.local/share/biblia/ubs/` and indexes them into `ubs.db`;
  - nothing from UBS is committed to this repository or embedded in the
    binary;
  - the source is credited in `biblia ajuda`, in the READMEs and here,
    and every gloss shown names its dictionary.

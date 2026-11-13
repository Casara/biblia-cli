// VS Code integration for the `biblia` command-line tool.
//
// biblia.consultar         shows the selected reference (or the one on the
//                          cursor line) in the "Bíblia" output panel,
//                          leaving the file alone
// biblia.inserir           replaces each selected reference with its text
// biblia.perguntar         asks for a reference and shows it
// biblia.paralelos         shows the selected reference with the list of
//                          its parallel passages
// biblia.inserirParalelos  keeps each selected reference and inserts the
//                          text of its parallel passages on the lines after
// biblia.original          shows the selected reference with the original
//                          Hebrew/Greek words (-s) in a read-only tab
// biblia.strong            lexicon entry of the Strong number under the
//                          cursor, or a search by transliteration or gloss
// biblia.consultarVersoes  asks for the translations, then works like
//                          biblia.consultar
// biblia.inserirVersoes    asks for the translations, then works like
//                          biblia.inserir
//
// The interlinear and the lexicon open as read-only "biblia:" documents
// rather than in the output panel: the cursor can stand on a Strong number
// there, and biblia.strong (or a click on the number) looks it up.
//
// The text is inserted in the format of biblia.formato; "auto" picks it
// from the file (biblia.formatosPorExtensao, then the file's language)
// and, for any other file, leaves it to the tool's own "formato_filtro".
"use strict";

const vscode = require("vscode");
const { execFile } = require("child_process");
const path = require("path");
const fs = require("fs");
const os = require("os");

// Where "make install" and "go install" put the binary. An editor started
// from the desktop may not have them in its PATH even when the terminal
// finds "biblia".
const fallbackDirs = [path.join(os.homedir(), ".local", "bin"), path.join(os.homedir(), "go", "bin")];

// candidates are the file names a command may have: "biblia.exe" too on
// Windows.
function candidates(name) {
  return process.platform === "win32" && !name.toLowerCase().endsWith(".exe") ? [name + ".exe", name] : [name];
}

function isExecutable(file) {
  try {
    fs.accessSync(file, fs.constants.X_OK);
    return fs.statSync(file).isFile();
  } catch {
    return false;
  }
}

function onPath(name) {
  return (process.env.PATH || "")
    .split(path.delimiter)
    .some((dir) => dir && candidates(name).some((n) => isExecutable(path.join(dir, n))));
}

// executable returns the configured binary, looked up in PATH and then in
// the usual install folders when only a bare name is configured.
function executable() {
  const exe = config().get("executavel") || "biblia";
  if (path.dirname(exe) !== "." || onPath(exe)) return exe;
  for (const dir of fallbackDirs) {
    for (const name of candidates(exe)) {
      const candidate = path.join(dir, name);
      if (isExecutable(candidate)) return candidate;
    }
  }
  return exe;
}

// Language ids of files without an extension in formatosPorExtensao.
const formatByLanguage = { markdown: "md", typst: "typ" };

let output;

function config() {
  return vscode.workspace.getConfiguration("biblia");
}

// run calls the binary. versions picks the "-v" argument: undefined
// takes biblia.versoes, a string overrides it for this run, and null sends
// none ("strong" and "versoes" have no -v flag and would refuse it).
function run(args, input, versions) {
  const exe = executable();
  const v = versions === undefined ? config().get("versoes") : versions;
  const full = v ? [...args, "-v", v] : args;

  return new Promise((resolve) => {
    const child = execFile(exe, full, { timeout: 30000 }, (err, stdout, stderr) => {
      if (err && err.code === "ENOENT") {
        resolve({ stdout: "", stderr: `biblia: executável não encontrado (${exe})` });
        return;
      }
      resolve({ stdout, stderr });
    });
    if (input !== undefined) {
      child.stdin.end(input);
    }
  });
}

function referenceRanges(editor) {
  return editor.selections.map((sel) =>
    sel.isEmpty ? editor.document.lineAt(sel.active.line).range : sel
  );
}

// formatArgs returns the "-f" arguments for a document, or none to let
// the tool use its own default.
function formatArgs(document) {
  const chosen = config().get("formato") || "auto";
  if (chosen !== "auto") return ["-f", chosen];

  const byExt = config().get("formatosPorExtensao") || {};
  const ext = path.extname(document.fileName || "").toLowerCase();
  const fmt = byExt[ext] || formatByLanguage[document.languageId];
  return fmt ? ["-f", fmt] : [];
}

async function show(refs, extra = [], versions) {
  const { stdout, stderr } = await run(["-f", "texto", ...extra, refs], undefined, versions);
  output.clear();
  output.appendLine(stdout.trimEnd() || stderr.trimEnd());
  output.show(true);
}

async function consultar() {
  const editor = vscode.window.activeTextEditor;
  if (!editor) return;
  const refs = selectedRefs(editor);
  if (refs) await show(refs);
}

function selectedRefs(editor) {
  return referenceRanges(editor)
    .map((r) => editor.document.getText(r).trim())
    .filter(Boolean)
    .join("; ");
}

async function inserir(versions) {
  const editor = vscode.window.activeTextEditor;
  if (!editor) return;
  const fmt = formatArgs(editor.document);
  const ranges = referenceRanges(editor);
  const results = await Promise.all(
    ranges.map((r) => run(["filtro", ...fmt], editor.document.getText(r), versions))
  );
  await editor.edit((edit) => {
    ranges.forEach((r, i) => {
      if (results[i].stdout) edit.replace(r, results[i].stdout.replace(/\n$/, ""));
    });
  });
  const errors = results.map((r) => r.stderr.trim()).filter(Boolean);
  if (errors.length) vscode.window.showWarningMessage(errors.join(" | "));
}

async function paralelos() {
  const editor = vscode.window.activeTextEditor;
  if (!editor) return;
  const refs = selectedRefs(editor);
  if (refs) await show(refs, ["-p"]);
}

async function inserirParalelos() {
  const editor = vscode.window.activeTextEditor;
  if (!editor) return;
  const fmt = formatArgs(editor.document);
  const ranges = referenceRanges(editor);
  const results = await Promise.all(
    ranges.map((r) => run(["filtro", "-p", ...fmt], editor.document.getText(r)))
  );
  await editor.edit((edit) => {
    ranges.forEach((r, i) => {
      const text = results[i].stdout.replace(/\n+$/, "");
      if (text) edit.insert(r.end, "\n" + text);
    });
  });
  const errors = results.map((r) => r.stderr.trim()).filter(Boolean);
  if (errors.length) {
    vscode.window.showWarningMessage(errors.join(" | "));
  } else if (results.every((r) => !r.stdout.trim())) {
    vscode.window.showInformationMessage("Bíblia: nenhuma passagem paralela no banco da UBS.");
  }
}

async function perguntar() {
  const refs = await vscode.window.showInputBox({ prompt: "Referência (ex.: Jo 3.16; Rm 8.28)" });
  if (refs && refs.trim()) await show(refs);
}

// Read-only "biblia:" documents: the text of each one, by URI. Kept after
// the tab closes, so reopening it (Ctrl+Shift+T, back) shows the text.
const scheme = "biblia";
const pages = new Map();
const pageChanged = new vscode.EventEmitter();

const pageProvider = {
  onDidChange: pageChanged.event,
  provideTextDocumentContent: (uri) => pages.get(uri.toString()) || "",
};

// showPage opens text as a read-only tab titled name: beside the file
// being edited, or in the same group when the current tab is already one
// of these pages (following a Strong number from the interlinear). The
// interlinear opens pinned (preview false), so the lexicon entries looked
// up from it, which open as previews and replace one another, leave it in
// place.
async function showPage(name, text, preview) {
  const uri = vscode.Uri.from({ scheme, path: "/" + name.replace(/[\\/]/g, "-") });
  const key = uri.toString();
  const known = pages.has(key);
  pages.set(key, text);
  if (known) pageChanged.fire(uri);

  const doc = await vscode.workspace.openTextDocument(uri);
  const current = vscode.window.activeTextEditor;
  const inPage = current && current.document.uri.scheme === scheme;
  await vscode.window.showTextDocument(doc, {
    viewColumn: inPage ? current.viewColumn : vscode.ViewColumn.Beside,
    preview,
  });
}

// showResult opens the answer of a call as a page, or warns when there is
// none. Notes on stderr next to an answer ("glosas em inglês...") become a
// warning too.
async function showResult(name, { stdout, stderr }, preview = true) {
  const err = stderr.trim();
  if (!stdout.trim()) {
    vscode.window.showWarningMessage(err || "Bíblia: nada encontrado.");
    return;
  }
  await showPage(name, stdout, preview);
  if (err) vscode.window.showWarningMessage(err);
}

async function original() {
  const editor = vscode.window.activeTextEditor;
  if (!editor) return;
  const refs = selectedRefs(editor);
  if (refs) await showResult(refs + " (original)", await run(["-f", "texto", "-s", refs]), false);
}

// A Strong number as the tool takes it: G25, g0025, H7451b, H2603A.
const strongPattern = /\b[HGhg]\d{1,4}[A-Za-z]?\b/;
const strongExact = new RegExp("^" + strongPattern.source + "$");

// strongUnderCursor returns the selected text, or the Strong number the
// cursor stands on (a whole interlinear line would carry the word, the
// transliteration and the gloss too).
function strongUnderCursor(editor) {
  const sel = editor.selection;
  if (!sel.isEmpty) return editor.document.getText(sel).trim();
  const range = editor.document.getWordRangeAtPosition(sel.active, strongPattern);
  return range ? editor.document.getText(range) : "";
}

async function strong(arg) {
  let query = typeof arg === "string" ? arg : "";
  const editor = vscode.window.activeTextEditor;
  if (!query && editor) query = strongUnderCursor(editor);

  if (!strongExact.test(query)) {
    query = await vscode.window.showInputBox({
      prompt: "Número de Strong (H2617, G25), transliteração ou glosa",
      placeHolder: "glosas no idioma dos dicionários: hebraico em português, grego em espanhol",
      value: query.split("\n")[0],
    });
    query = query && query.trim();
    if (!query) return;
  }

  if (strongExact.test(query)) {
    await showResult(query.toUpperCase(), await run(["strong", query, "-c"], undefined, null));
    return;
  }

  const how = await vscode.window.showQuickPick(
    [
      { label: "Transliteração", description: `strong -tr ${query}`, flag: "-tr" },
      { label: "Glosa", description: `strong -glosa ${query}`, flag: "-glosa" },
    ],
    { placeHolder: `Buscar “${query}” por…` }
  );
  if (how) await showResult(`${how.label}: ${query}`, await run(["strong", how.flag, query], undefined, null));
}

// strongLinks makes every Strong number in a "biblia:" page clickable.
const strongLinks = {
  provideDocumentLinks(doc) {
    const links = [];
    const all = new RegExp(strongPattern.source, "g");
    for (let i = 0; i < doc.lineCount; i++) {
      const text = doc.lineAt(i).text;
      for (const m of text.matchAll(all)) {
        const range = new vscode.Range(i, m.index, i, m.index + m[0].length);
        const target = vscode.Uri.parse(
          "command:biblia.strong?" + encodeURIComponent(JSON.stringify([m[0]]))
        );
        const link = new vscode.DocumentLink(range, target);
        link.tooltip = "Bíblia: Strong " + m[0];
        links.push(link);
      }
    }
    return links;
  },
};

// pickVersions asks for the translations of one run, starting from the
// last choice (or biblia.versoes). It returns a comma-separated list, or
// undefined when the reader gives up.
async function pickVersions(context) {
  const { stdout, stderr } = await run(["versoes"], undefined, null);
  const items = stdout
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [id, ...rest] = line.split(/\s+/);
      return { label: id, description: rest.join(" ") };
    });
  if (!items.length) {
    vscode.window.showWarningMessage(stderr.trim() || "Bíblia: lista de traduções vazia.");
    return undefined;
  }

  const last = context.globalState.get("versoes") || config().get("versoes") || "";
  const chosen = new Set(last.split(",").map((v) => v.trim().toLowerCase()).filter(Boolean));
  items.sort((a, b) => chosen.has(b.label) - chosen.has(a.label));
  items.forEach((item) => (item.picked = chosen.has(item.label)));

  const picked = await vscode.window.showQuickPick(items, {
    canPickMany: true,
    placeHolder: "Traduções desta consulta",
  });
  if (!picked || !picked.length) return undefined;

  const versions = picked.map((p) => p.label).join(",");
  await context.globalState.update("versoes", versions);
  return versions;
}

function activate(context) {
  output = vscode.window.createOutputChannel("Bíblia");

  const consultarVersoes = async () => {
    const editor = vscode.window.activeTextEditor;
    if (!editor) return;
    const refs = selectedRefs(editor);
    if (!refs) return;
    const versions = await pickVersions(context);
    if (versions) await show(refs, [], versions);
  };
  const inserirVersoes = async () => {
    if (!vscode.window.activeTextEditor) return;
    const versions = await pickVersions(context);
    if (versions) await inserir(versions);
  };

  context.subscriptions.push(
    output,
    pageChanged,
    vscode.workspace.registerTextDocumentContentProvider(scheme, pageProvider),
    vscode.languages.registerDocumentLinkProvider({ scheme }, strongLinks),
    vscode.commands.registerCommand("biblia.original", original),
    vscode.commands.registerCommand("biblia.strong", strong),
    vscode.commands.registerCommand("biblia.consultarVersoes", consultarVersoes),
    vscode.commands.registerCommand("biblia.inserirVersoes", inserirVersoes),
    vscode.commands.registerCommand("biblia.consultar", consultar),
    vscode.commands.registerCommand("biblia.inserir", () => inserir()),
    vscode.commands.registerCommand("biblia.perguntar", perguntar),
    vscode.commands.registerCommand("biblia.paralelos", paralelos),
    vscode.commands.registerCommand("biblia.inserirParalelos", inserirParalelos)
  );
}

function deactivate() {}

module.exports = { activate, deactivate };

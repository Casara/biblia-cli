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

function run(args, input) {
  const exe = executable();
  const versoes = config().get("versoes");
  const full = versoes ? [...args, "-v", versoes] : args;

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

async function show(refs, extra = []) {
  const { stdout, stderr } = await run(["-f", "texto", ...extra, refs]);
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

async function inserir() {
  const editor = vscode.window.activeTextEditor;
  if (!editor) return;
  const fmt = formatArgs(editor.document);
  const ranges = referenceRanges(editor);
  const results = await Promise.all(
    ranges.map((r) => run(["filtro", ...fmt], editor.document.getText(r)))
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

function activate(context) {
  output = vscode.window.createOutputChannel("Bíblia");
  context.subscriptions.push(
    output,
    vscode.commands.registerCommand("biblia.consultar", consultar),
    vscode.commands.registerCommand("biblia.inserir", inserir),
    vscode.commands.registerCommand("biblia.perguntar", perguntar),
    vscode.commands.registerCommand("biblia.paralelos", paralelos),
    vscode.commands.registerCommand("biblia.inserirParalelos", inserirParalelos)
  );
}

function deactivate() {}

module.exports = { activate, deactivate };

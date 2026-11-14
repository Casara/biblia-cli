# Changelog

Mudanças da extensão do VS Code. As do programa `biblia` estão no
`CHANGELOG.md` da raiz do repositório.

## [0.3.1] - 2026-11-13

### Documentação

- README com imagens das extensões funcionando.

## [0.3.0] - 2026-11-13

### Adicionado

- **Bíblia: consultar com o original (hebraico/grego e Strong)**
  (`Ctrl+Alt+O`): o `-s` do programa, numa aba só de leitura ao lado do
  arquivo. Os números de Strong dessa aba são links para o verbete
  (`Ctrl`+clique).
- **Bíblia: Strong…**: o verbete do número sob o cursor ou selecionado;
  sem número, pergunta o número, a transliteração (`-tr`) ou a glosa
  (`-glosa`).
- **Bíblia: consultar em outras traduções…** e **inserir texto em outras
  traduções…**: escolha das traduções só para aquela vez, numa lista com
  seleção múltipla que lembra a última escolha.

### Corrigido

- O `biblia.versoes` não é mais enviado aos subcomandos que não aceitam
  `-v` (`strong`, `versoes`).

## [0.2.0] - 2026-11-13

### Adicionado

- **Bíblia: consultar referência selecionada** (`Ctrl+Alt+B`): mostra o
  texto no painel *Bíblia*, sem mexer no arquivo.
- **Bíblia: inserir texto no lugar da referência** (`Ctrl+Alt+Shift+B`).
- **Bíblia: consultar…**: pergunta a referência.
- **Bíblia: passagens paralelas da referência selecionada**
  (`Ctrl+Alt+L`): mostra o texto e a lista de passagens paralelas.
- **Bíblia: inserir o texto das passagens paralelas**
  (`Ctrl+Alt+Shift+L`): mantém a referência e insere, nas linhas
  seguintes, o texto de cada paralelo, dizendo o que ele é (no `typ`, um
  comentário como `// paralelo: Rm 1.17 cita Hc 2.4`).
- Ícone, README em inglês (o que o Marketplace mostra) e em português,
  com a instalação do programa `biblia` passo a passo.
- O programa é procurado no `PATH` e em `~/.local/bin` e `~/go/bin`; no
  Windows, também como `biblia.exe`.
- Opções `biblia.executavel`, `biblia.versoes`, `biblia.formato` (padrão
  `auto`: o formato vem do tipo do arquivo) e `biblia.formatosPorExtensao`
  (`.md` → `md`, `.typ` → `typ`). Num arquivo sem formato associado, vale
  o `formato_filtro` do `~/.config/biblia/config.json`.

[0.3.1]: https://github.com/Casara/biblia-cli/tree/vscode-v0.3.1
[0.3.0]: https://github.com/Casara/biblia-cli/tree/vscode-v0.3.0
[0.2.0]: https://github.com/Casara/biblia-cli/tree/vscode-v0.2.0

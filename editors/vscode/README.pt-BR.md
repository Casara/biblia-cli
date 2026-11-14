# Bíblia (biblia-cli)

Consulte versículos sem sair do editor: em várias traduções ao mesmo
tempo, com o original hebraico/grego, os números de Strong, os sentidos
dos dicionários e as passagens paralelas, e insira-os em Markdown ou
Typst prontos para citar. Feita para referências e traduções em
português (ARA, NVI, NAA...), mas inglês, espanhol e as outras traduções
da [Bíblia Online](https://www.bibliaonline.com.br) também funcionam.

*Read this in [English](https://github.com/Casara/biblia-cli/blob/main/editors/vscode/README.md).*

![Num arquivo Typst, a referência Rm 8.28 vira o comentário "conferido" e a citação #bref; num arquivo Markdown, uma citação com a referência][inserir]

[inserir]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/vscode-inserir.gif

## Requisito: o programa `biblia`

Esta extensão é uma interface: quem faz cada consulta é o programa de
linha de comando **`biblia`**, que precisa ser instalado à parte. Sem
ele, os comandos avisam *executável não encontrado*.

1. **Baixe-o** na
   [página de versões](https://github.com/Casara/biblia-cli/releases/latest):
   - Linux: `biblia_<versão>_linux_amd64.tar.gz` (ou `arm64`);
   - macOS: `biblia_<versão>_darwin_arm64.tar.gz` (Apple silicon) ou
     `darwin_amd64` (Intel);
   - Windows: `biblia_<versão>_windows_amd64.zip`.

   Descompacte e ponha o `biblia` (`biblia.exe` no Windows) numa pasta
   que esteja no `PATH`, como `~/.local/bin`.

   Ou, com o Go instalado:
   `go install github.com/casara/biblia-cli/cmd/biblia@latest`.

2. **Confira** num terminal: `biblia jo 3.16` deve mostrar João 3.16.

3. **Opcional:** `biblia dados baixar` baixa o texto original, os
   dicionários e as passagens paralelas (cerca de 220 MB, ou 75 MB com
   `-enxuto`). Sem eles, a consulta de versículos funciona, mas os
   comandos de passagens paralelas não.

A extensão procura o `biblia` no `PATH` e depois em `~/.local/bin` e
`~/go/bin`. Se estiver em outro lugar, informe o caminho completo em
**`biblia.executavel`**.

## Comandos

| Comando | Atalho | O que faz |
| --- | --- | --- |
| **Bíblia: consultar referência selecionada** | `Ctrl+Alt+B` | mostra o texto no painel *Bíblia*, sem mexer no arquivo |
| **Bíblia: inserir texto no lugar da referência** | `Ctrl+Alt+Shift+B` | troca a referência selecionada pelo texto |
| **Bíblia: consultar…** | — | pergunta a referência e mostra o texto |
| **Bíblia: passagens paralelas da referência selecionada** | `Ctrl+Alt+L` | mostra o texto e a lista de passagens paralelas (citações do AT no NT, paralelos dos Evangelhos...) |
| **Bíblia: inserir o texto das passagens paralelas** | `Ctrl+Alt+Shift+L` | mantém a referência e insere, nas linhas seguintes, o texto de cada paralelo, dizendo o que ele é |
| **Bíblia: consultar com o original (hebraico/grego e Strong)** | `Ctrl+Alt+O` | abre uma aba só de leitura com o texto e, palavra por palavra, o original hebraico/grego, a transliteração, o número de Strong, a glosa e a gramática |
| **Bíblia: Strong…** | — | o verbete do número de Strong sob o cursor (`G25`, `H7451b`); sem número, pergunta o número, a transliteração ou a glosa |
| **Bíblia: consultar em outras traduções…** | — | pergunta as traduções desta vez e faz o mesmo que o *consultar* |
| **Bíblia: inserir texto em outras traduções…** | — | pergunta as traduções desta vez e faz o mesmo que o *inserir* |

Sem seleção, vale a linha do cursor. As referências são escritas como no
Brasil: `Jo 3.16`, `1 Co 13.4-7`, `Rm 5.8; 6.23`, `Sl 23`.

Na aba do original, os números de Strong são links: `Ctrl`+clique
(`Cmd`+clique no macOS) abre o verbete no mesmo grupo, sem fechar o
interlinear; o mesmo faz **Bíblia: Strong…** com o cursor no número. A busca pela glosa usa o idioma dos dicionários:
português no hebraico, espanhol no grego (a UBS ainda não tem o
dicionário grego em português). A escolha de traduções lembra a última
feita e não muda o `biblia.versoes`. O original e o Strong precisam do
`biblia dados baixar`. O
`Ctrl+Alt+B` toma o lugar do atalho do VS Code para a barra lateral
secundária; troque qualquer um em *Atalhos de teclado*.

![O interlinear de Jo 3.16 numa aba ao lado do arquivo: palavra grega, transliteração, número de Strong, glosa e gramática][original]

![Ctrl+clique no G0025 do interlinear abre o verbete em outra aba][strong]

![A lista de traduções com ara e arc marcadas][traducoes]

[original]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/vscode-original.png
[strong]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/vscode-strong.gif
[traducoes]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/vscode-traducoes.png

## Configurações

| Configuração | Padrão | O que faz |
| --- | --- | --- |
| `biblia.executavel` | `biblia` | o programa, pelo nome ou pelo caminho completo |
| `biblia.versoes` | (vazio) | traduções separadas por vírgula (`ara,naa`); vazio usa o arquivo de configuração do programa |
| `biblia.formato` | `auto` | formato do texto inserido: `auto`, `linha`, `md`, `typ` ou um modelo seu |
| `biblia.formatosPorExtensao` | `.md` → `md`, `.typ` → `typ` | no `auto`: extensão do arquivo → formato |

No `auto`, arquivos Markdown recebem uma citação com a referência,
arquivos Typst recebem o `#bref(...)` precedido da linha
`// conferido: ARA · ... · data`, e os demais usam o `formato_filtro` do
programa. Os formatos, os campos dos modelos e o arquivo de configuração
do programa estão no
[README do biblia](https://github.com/Casara/biblia-cli/blob/main/README.pt-BR.md).

## Fontes e créditos

- **Texto bíblico:** [Bíblia Online](https://www.bibliaonline.com.br),
  pelo serviço que o site publica para programas, até 5 versículos por
  consulta. Cada tradução tem seus titulares de direitos; nem a extensão
  nem o programa trazem texto bíblico.
- **Original, léxicos e gramática:** [STEPBible](https://www.stepbible.org)
  (Tyndale House, Cambridge, CC BY 4.0), baixados sob demanda.
- **Dicionários e passagens paralelas:** Sociedade Bíblica Unida
  ([ubs-open-license](https://github.com/ubsicap/ubs-open-license), CC
  BY-SA 4.0), baixados sob demanda.

## Licença

MIT. Os dados mantêm as licenças das suas fontes.

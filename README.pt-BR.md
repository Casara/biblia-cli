# biblia

Consulta de versículos pela linha de comando ou de dentro do editor, em
várias traduções ao mesmo tempo, com o original hebraico/grego, os
números de Strong, os sentidos dos dicionários e as passagens paralelas
quando se quer ir mais fundo.

*Read this in [English](README.md).*

```console
$ biblia jo 3.16 -v ara,naa,nvi
Jo 3.16
  ARA  Porque Deus amou ao mundo de tal maneira que deu o seu Filho unigênito, ...
  NAA  Porque Deus amou o mundo de tal maneira que deu o seu Filho unigênito, ...
  NVI  — Porque Deus tanto amou o mundo que deu o seu Filho Unigênito, ...
```

## Instalação

Escolha uma das formas:

- **Binário pronto (sem precisar de Go):** baixe na
  [página de versões](https://github.com/Casara/biblia-cli/releases/latest)
  o arquivo do seu sistema (`linux`, `darwin` para macOS ou `windows`;
  `amd64` ou `arm64`), descompacte e ponha o `biblia` (`biblia.exe` no
  Windows) numa pasta do `PATH`, como `~/.local/bin`.
- **Com Go 1.26 ou mais novo:**
  `go install github.com/casara/biblia-cli/cmd/biblia@latest`.
- **A partir do código:** `make install`, que instala em
  `~/.local/bin/biblia`.

Depois, opcionalmente:

```sh
biblia dados baixar   # original, léxicos, dicionários e paralelos (~220 MB)
                      # com -enxuto, apaga os arquivos baixados após indexar (~75 MB)
```

## Uso

```sh
biblia jo 3.16                    # tradução padrão (ara, ou a do arquivo de configuração)
biblia jo 3.16 -v ara,naa,nvi     # lado a lado
biblia rm 8.28-30 -v kjv,rves     # inglês, espanhol: qualquer id de "biblia versoes -todas"
biblia "Rm 5.8; 6.23"             # várias referências; o livro vale para o trecho seguinte
biblia jo 3.16 -s                 # + palavras do original: Strong, sentido e gramática
biblia hc 2.4 -p                  # passagens paralelas (-pt: com o texto)
biblia strong H2617               # todos os sentidos de um número
biblia strong G0025 -c            # + domínio de cada sentido e o verbete da STEPBible
biblia strong -tr agape           # pela transliteração (sem acentos também)
biblia strong -glosa amor         # pela glosa, no idioma dos dicionários
biblia jo 3.16 -f typ             # pronto para colar, com a linha "// conferido"
biblia jo 3.16 -offline           # só o cache
```

A referência aceita `jo 3.16`, `Jo 3:16`, `1co13.4-7`, `I Co 13.4`,
`Sl 23.1-3,6`, `Rm 8` (o capítulo), `Jd 3` (livro de um capítulo só), o
nome inteiro (`Gênesis`, `genesis`) ou um começo que não deixe dúvida
(`roman`). `jo` é João; `jó` (com acento) ou `job` é Jó.

### Original, dicionários e paralelos

O `-s` mostra cada palavra do original com o número de Strong, a glosa
do sentido que o dicionário dá a ela **naquele versículo** e a gramática
por extenso:

```text
  Sl 51.1 · hebraico (no original: 51.3) · glosas: UBS hebraico (pt)
    חָנֵּ֣נִי      choneni      H2603A  ter boa vontade · verbo, qal, imperativo, ...
    כְּחַסְדֶּ֑ךָ  kechasDekha  H2617A  bondade [IA] · preposição + substantivo, ...
```

- **Dicionários:** os da Sociedade Bíblica Unida (UBS). O hebraico existe
  em português; o grego não, e vem em espanhol, o mais próximo. Parte do
  hebraico em português foi traduzida por IA, e a própria UBS marca cada
  sentido assim: o programa mostra `[IA]`. Palavras que o dicionário não
  traz (sobretudo preposições e partículas) ficam com a glosa da
  STEPBible, em inglês, marcada `(en)`.
- **Gramática:** os códigos da STEPBible (`V-AAI-3S`, `HR/Ncfsa`)
  traduzidos termo a termo.
- **Paralelos (`-p`):** do banco de passagens paralelas da UBS: citações
  do AT no NT, paralelos dos Evangelhos, Reis e Crônicas. O `-pt` traz
  também o texto. A numeração hebraica (Jl 3.1) é convertida para a das
  traduções (Jl 2.28). Nos formatos de colar, a lista e a relação de cada
  texto entram no próprio texto (veja *Paralelos nos formatos*, abaixo).

### Formatos

| Formato | Saída |
| --- | --- |
| `texto` | a visão do terminal (padrão) |
| `linha` | `texto (Jo 3.16, ARA)` numa linha só |
| `md` | citação em Markdown com a referência |
| `typ` | `// conferido: ARA · bibliaonline.com.br/ara · 2026-11-12` e `#bref("Jo 3.16", [«...»])` |

### Paralelos nos formatos

Com `-p` ou `-pt` (e no comando dos editores que insere os paralelos), os
formatos de colar dizem o que é cada texto:

```text
// conferido: ARA · bibliaonline.com.br/ara · 2026-11-12
#bref("Hc 2.4", [«Eis o soberbo! ... mas o justo viverá pela sua fé.»])
// Citado no NT: Rm 1.17 · Gl 3.11 · Hb 10.38

// paralelo: Rm 1.17 cita Hc 2.4
// conferido: ARA · bibliaonline.com.br/ara · 2026-11-12
#bref("Rm 1.17", [«visto que a justiça de Deus se revela ...»])
```

| Formato | Na passagem pedida | Em cada paralelo |
| --- | --- | --- |
| `typ` | comentário com a lista, depois do `#bref` | comentário com a relação, antes do `// conferido` |
| `md` | linha em itálico com a lista | a relação na linha da referência (`— Rm 1.17 (ARA) · *cita Hc 2.4*`) |
| `linha` | nada (é uma linha só) | a relação nos parênteses (`(Rm 1.17, ARA; cita Hc 2.4)`) |

No `typ`, tudo fica em comentários: não aparece no PDF, e a linha
`// conferido` continua logo acima do `#bref`. Para mostrar a relação no
documento, use um modelo próprio (abaixo).

### Modelos próprios

Um modelo é um formato seu, escrito como
[template do Go](https://pkg.go.dev/text/template) na chave `modelos` do
arquivo de configuração e usado pelo nome (`-f citacao`; nos editores, na
opção `formato`). Um modelo com o nome de um formato do programa (`linha`,
`md`, `typ`) o substitui. No exemplo, `#nota` seria uma função do seu
documento Typst para mostrar a relação no PDF.

```json
{
  "modelos": {
    "citacao": "“{{.Text}}” ({{.Ref}}, {{.Bible}}){{with .ParallelOf}} — {{.Relation}}{{end}}",
    "booklet": "{{with .ParallelOf}}#nota[{{.Relation}}]\n{{end}}// conferido: {{.Bible}} · bibliaonline.com.br/{{.BibleID}} · {{.Date}}\n#bref(\"{{.Ref}}\", [«{{typ .Text}}»])"
  }
}
```

**Campos do texto:**

| Campo | O que é | Exemplo |
| --- | --- | --- |
| `.Ref` | a referência do texto obtido | `Gn 3.1-5` |
| `.Requested` | a referência pedida (difere de `.Ref` num trecho parcial) | `Gn 3.1-6` |
| `.Bible` / `.BibleID` | a tradução, em maiúsculas / como o site a chama | `ARA` / `ara` |
| `.Text` | os versículos juntos, sem números | |
| `.Verses` | os versículos, cada um com `.Number` e `.Text` | |
| `.URL` | o link da passagem (do capítulo, se parcial) | |
| `.Date` | quando o texto veio do site | `2026-11-12` |
| `.Partial` / `.Missing` | se faltaram versículos / quais | `true` / `6` |
| `.Absent` | versículos que a tradução não tem | `21` |

**Campos dos paralelos** (só com `-p`, `-pt` ou `filtro -p`):

| Campo | Onde aparece | O que é |
| --- | --- | --- |
| `.Parallels` | na passagem pedida | a lista dos paralelos; cada item tem `.Ref` (`Rm 1.17`) e `.Kind` |
| `.ParallelOf` | no texto de cada paralelo | de onde ele veio: `.Ref` (a passagem pedida, `Hc 2.4`), `.Kind` e `.Relation` |

`.Kind` é o tipo, do ponto de vista da passagem pedida: `citado no NT`
(Hc 2.4 é citado em Rm 1.17), `citação do AT` (Rm 3.4 cita Sl 51.4) ou
`paralelo` (Mt 6.9 e Lc 11.2). `.Relation` diz o mesmo do ponto de vista
do paralelo, numa frase pronta: `cita Hc 2.4`, `citado em Rm 3.4`,
`paralelo de Mt 6.9`. Fora dos paralelos, `.Parallels` é vazio e
`.ParallelOf` não existe, então `{{if .Parallels}}` e
`{{with .ParallelOf}}` bastam para testar.

**Funções:**

| Função | O que faz | Exemplo |
| --- | --- | --- |
| `typ` | protege os caracteres especiais do Typst | `{{typ .Text}}` |
| `md` | protege os caracteres especiais do Markdown | `{{md .Text}}` |
| `parallels` | escreve a lista agrupada por tipo | `{{parallels .Parallels}}` → `Citado no NT: Rm 1.17 · Gl 3.11 · Hb 10.38` |

Dentro de `{{with .ParallelOf}}`, os campos são os do paralelo; o texto
continua acessível com `$`: `{{with .ParallelOf}}{{$.Ref}} {{.Relation}}{{end}}`
→ `Rm 1.17 cita Hc 2.4`.

### Configuração

`~/.config/biblia/config.json` (ou o caminho em `BIBLIA_CONFIG`). Todos os
campos são opcionais:

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

`idioma` escolhe os dicionários padrão e o idioma da gramática;
`dicionarios` fixa a lista, em ordem de preferência (`biblia dados
status` mostra a que está em uso). Disponíveis: `ubs-hebraico-pt`,
`-es`, `-en`, `-fr`; `ubs-grego-es`, `-en`, `-fr`; e `step` (inglês, da
STEPBible). Depois de mudar a lista, rode `biblia dados baixar ubs`.

### Limites que vêm da fonte

- **Até 5 versículos por consulta.** É o limite do serviço do Bíblia
  Online. Para um trecho maior, a saída mostra os 5 primeiros, avisa o que
  faltou e dá o link do capítulo. Versículos já consultados antes saem do
  cache e não contam.
- **Versículo que a tradução não tem** (ex.: Mt 17.21 na NVI, ou um número
  além do fim do capítulo num intervalo) aparece como *não consta nesta
  tradução*, não como erro.
- **SENHOR.** A ARA impressa usa versalete (SENHOR) para o nome divino; o
  site devolve "Senhor". Ao colar num texto que é conferido contra a
  revista, ajuste à mão.

## Nos editores

Os dois plugins chamam o mesmo binário e têm os mesmos comandos. Os
atalhos vêm ativos no VS Code; no Sublime, são sugestões a copiar para o
seu keymap, porque o Package Control não aceita pacotes que ativem
atalhos.

| Comando | Atalho | O que faz |
| --- | --- | --- |
| **consultar** | `Ctrl+Alt+B` | mostra o texto, sem mexer no arquivo |
| **inserir** | `Ctrl+Alt+Shift+B` | troca a referência selecionada pelo texto |
| **consultar…** | (paleta de comandos) | pergunta a referência e mostra o texto |
| **passagens paralelas** | `Ctrl+Alt+L` | mostra o texto e a lista de paralelos |
| **inserir passagens paralelas** | `Ctrl+Alt+Shift+L` | mantém a referência e insere, nas linhas seguintes, o texto de cada paralelo |
| **consultar com o original** | `Ctrl+Alt+O` | abre uma aba com o interlinear (`-s`) |
| **Strong…** | (paleta de comandos) | o verbete do número sob o cursor, ou uma busca por transliteração ou glosa |
| **consultar / inserir em outras traduções…** | (paleta de comandos) | escolhe as traduções só daquela vez |

Sem seleção, vale a linha do cursor. O formato do que é inserido é, por
padrão, `auto`: `.md` recebe `md`, `.typ` recebe `typ` (e, num arquivo
ainda sem nome, vale a linguagem do editor); qualquer outro arquivo usa o
`formato_filtro` do arquivo de configuração. Os pares extensão → formato
são uma opção do plugin, e `formato` também aceita um formato fixo.

- **Sublime Text:** pelo Package Control (pacote **Biblia**) ou, a
  partir do código, `make sublime`, que liga `editors/sublime` à pasta de
  pacotes (`~/.config/sublime-text-3` ou `~/.config/sublime-text`).
  Opções e atalhos em *Preferences › Package Settings › Biblia*: os
  atalhos sugeridos estão à esquerda; copie os que quiser para o seu
  keymap, à direita.
- **VS Code:** pelo Marketplace (extensão **Bíblia (biblia-cli)**) ou, a
  partir do código, `make vscode`, que empacota e instala a extensão
  (requer Node.js). Opções em *Configurações › Bíblia*; atalhos em
  *Atalhos de teclado*. O `Ctrl+Alt+B` da extensão toma o lugar do atalho
  do VS Code que abre e fecha a barra lateral secundária.

Cada plugin tem o seu README, com a instalação do `biblia` passo a passo:
[VS Code](editors/vscode/README.pt-BR.md) e
[Sublime Text](editors/sublime/README.pt-BR.md).

Os atalhos evitam os que o Sublime já usa (`Ctrl+Alt+P`,
`Ctrl+Alt+Shift+P`, `Ctrl+Alt+Q`) e os do GNOME (`Ctrl+Alt+D`,
`Ctrl+Alt+T`).

Qualquer outro editor que filtre a seleção por um comando externo serve:
`biblia filtro -f linha` lê referências na entrada e devolve o texto. Linha
que não é referência volta igual. Com `-p`, o filtro devolve só o texto
das passagens paralelas de cada referência, e nada mais.

## Em qualquer programa (GNOME)

O GNOME não tem menu de contexto para texto selecionado; no lugar dele,
atalhos de teclado globais consultam o que estiver selecionado em
qualquer programa (navegador, leitor de PDF, terminal): uma referência
(`Jo 3.16`) ou um número de Strong (`G5293`, `H2617`).

| Atalho | Referência | Número de Strong |
| --- | --- | --- |
| `Super+B` | mostra o texto numa janela | mostra o verbete (`strong -c`) |
| `Super+Alt+B` | o mesmo, com o original hebraico/grego (`-s`) | mostra o verbete |
| `Super+Shift+B` | copia o texto, pronto para colar (`formato_filtro`) | copia o verbete |

![Janela do Super+B com Ef 1.22-23 na ARA e o botão Traduções…][gnome-consultar]

Basta selecionar, sem `Ctrl+C`; sem seleção, vale o que estiver na área
de transferência. Na janela de uma referência, o botão **Traduções…**
abre a lista das traduções do site, com as atuais marcadas: escolha uma
ou várias (lado a lado) e a janela reabre com elas. A escolha vale até
fechar a janela; a próxima consulta volta às do `config.json`.

![Lista de traduções com caixas de marcar, a ARA marcada][gnome-traducoes]

![Janela do Super+Alt+B: Ef 1.22-23 e, palavra por palavra, o grego com transliteração, Strong, glosa e gramática][gnome-original]

![Janela do Super+B sobre o número H7706: o verbete de Shaddai na STEPBible e no dicionário hebraico da UBS][gnome-strong]

[gnome-consultar]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/gnome-consultar.png
[gnome-traducoes]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/gnome-traducoes.png
[gnome-original]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/gnome-original.png
[gnome-strong]: https://raw.githubusercontent.com/Casara/biblia-cli/main/docs/images/gnome-strong.png

Requer o `zenity` e, no Wayland, o `wl-clipboard` (no X11, o `xclip`).
Com o `biblia` já no `PATH`:

```sh
sudo apt install zenity wl-clipboard
biblia atalhos instalar
```

Os atalhos ficam em *Configurações › Teclado › Atalhos personalizados*,
onde podem ser trocados; os que você já tinha ficam como estão. Eles
chamam o `biblia` pelo caminho completo: se mudá-lo de lugar, rode o
`instalar` de novo. `biblia atalhos remover` os apaga. Para copiar em
outro formato, edite o comando do atalho no GNOME, acrescentando
`-f md` (ou `typ`, `linha`, um modelo seu) ao `selecao copiar`.

## Onde ficam os arquivos

| O quê | Linux | macOS | Windows |
| --- | --- | --- | --- |
| preferências (`config.json`) | `~/.config/biblia/` | `~/Library/Application Support/biblia/` | `%AppData%\biblia\` |
| versículos já consultados (`cache.db`) | `~/.cache/biblia/` | `~/Library/Caches/biblia/` | `%LocalAppData%\biblia\` |
| dados da STEPBible e da UBS e os índices | `~/.local/share/biblia/` | `~/Library/Application Support/biblia/` | `%LocalAppData%\biblia\` |

`biblia dados status` mostra onde estão os índices, e `biblia cache
status`, onde está o cache; `biblia cache limpar [versão]` o esvazia. `BIBLIA_CONFIG` aponta
outro arquivo de configuração, e `XDG_DATA_HOME` outra pasta de dados.

## Fontes e créditos

- **Textos bíblicos:** [Bíblia Online](https://www.bibliaonline.com.br), pelo
  serviço que o site publica para programas. Cada tradução tem seus titulares
  de direitos; o programa não distribui texto bíblico.
- **Original e léxicos:** dados criados por [www.STEPBible.org](https://www.stepbible.org)
  a partir do trabalho da Tyndale House, Cambridge, sob CC BY 4.0
  ([STEPBible-Data](https://github.com/STEPBible/STEPBible-Data)). Baixados
  sob demanda, nunca redistribuídos.
- **Dicionários e paralelos:** Sociedade Bíblica Unida (United Bible
  Societies, [ubs-open-license](https://github.com/ubsicap/ubs-open-license)),
  sob CC BY-SA 4.0. Baixados sob demanda.

Detalhes e termos em [docs/SOURCES.md](docs/SOURCES.md).

## Licença

[MIT](LICENSE), para o código. Os dados têm as licenças das suas fontes.

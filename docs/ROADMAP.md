# Próximos passos

O que está planejado e ainda não foi feito: correções, funcionalidades e
melhorias do programa. Cada item diz como as coisas são hoje, o que
mudaria e o que precisa ser resolvido antes. Quando um item for feito,
apague-o daqui e registre-o no [CHANGELOG](../CHANGELOG.md).

## Dicionários

### Formato próprio de dicionário

**Hoje.** As glosas e os sentidos vêm só de duas fontes embutidas no
código: os dicionários da UBS e o léxico da STEPBible. A lista
`dicionarios` do `config.json` escolhe entre elas, mas não aceita outra.

**O que mudaria.** Um formato de arquivo documentado para qualquer
dicionário, que o programa leria da pasta de dados e trataria como os da
UBS: entraria na lista `dicionarios`, na ordem de preferência, e
apareceria no `-s` e no `strong`. O formato teria:

- **a chave**, que pode ser o número de Strong (`H2617`) ou a palavra
  original (`חֶסֶד`, `ἀγάπη`). A palavra serve para os dicionários que não
  são numerados pelo Strong, como Vine e Gingrich; o programa casaria pela
  palavra que a STEPBible dá a cada número;
- **um ou mais sentidos**, cada um com glosas e definição;
- **um cabeçalho** com nome, idioma, autor, licença e a marca de tradução
  automática, se houver, como a UBS faz por sentido.

**Antes de fazer.** Só vale a pena quando existir um dicionário que o
usuário tenha direito de usar. Até agora, os dicionários Strong em
português são comerciais (Sociedade Bíblica do Brasil, CPAD, Vida Nova) e
não são vendidos em arquivo aberto.

### Importadores

**Hoje.** Não há como trazer um dicionário de outro programa.

**O que mudaria.** Um comando, por exemplo
`biblia dados importar <arquivo>`, que converte um dicionário de outro
formato para o formato próprio do item anterior. Um importador por
formato, escrito quando aparecer um arquivo real:

- **módulos do e-Sword, MySword e theWord**, que são bancos SQLite ou
  arquivos de texto com marcação própria;
- **JSON**, como o glossário da Bíblia Belém (CC BY-NC: uso não comercial).

**Limite.** Os dicionários comprados no Logos ou no Olive Tree ficam
presos aos aplicativos, sem exportação de texto; não há importador
possível para eles.

### Grego em português

**Hoje.** O dicionário grego da UBS não tem edição em português; o
programa usa a espanhola, a mais próxima. A única versão em português
conhecida, a do NepeSearch, é tradução por IA sem revisão humana, e o
projeto não aceita essa fonte.

**O que mudaria.** Quando a UBS publicar a edição em português (o README
do repositório promete outros idiomas "quando a localização terminar"),
basta acrescentar o arquivo à lista `ubs.Dictionaries`. O padrão do
`idioma: "pt"` passaria a escolhê-la sozinho, porque `ubs.Defaults` prefere
o idioma do leitor ao mais próximo.

**O que acompanhar.** A pasta `dictionaries/greek/` do repositório
`ubsicap/ubs-open-license`.

### Gramática em espanhol e inglês

**Hoje.** A gramática do `-s` é traduzida termo a termo por uma tabela
própria, que só existe para o português (`grammar.portuguese`). Com
`idioma` em qualquer outra língua, a descrição sai com os termos da
legenda da STEPBible em inglês, em minúsculas (*verb, aorist, active*).

**O que mudaria.** Uma tabela por idioma, com os mesmos cerca de 200
termos, escolhida pelo `idioma`. O espanhol é o caso provável, porque o
grego já sai em espanhol. O inglês ganharia uma tabela de verdade
(*aorist active indicative, 3rd person singular*) no lugar dos valores
crus da legenda.

**Como conferir.** O teste que já existe (`BIBLIA_STEP_DB=<step.db> go
test ./internal/grammar/`) passaria a cobrir cada tabela: todo código do
texto precisa de descrição completa.

## Original

### Sentido por palavra, não só por versículo

**Hoje.** No `-s`, a glosa de cada palavra é a do sentido que a UBS
associa àquele versículo. A ligação é por versículo, não por palavra. Se a
mesma palavra aparece duas vezes no versículo com sentidos diferentes,
as duas ocorrências mostram as duas glosas juntas (*a / b*), sem dizer
qual é qual.

**Por que é assim.** A UBS conta as palavras de outro jeito que a
STEPBible: no hebraico, separa os prefixos (בְּרֵאשִׁית são duas palavras) e
usa só números pares; a STEPBible junta prefixo e palavra numa linha só.
As posições das duas fontes não se correspondem diretamente.

**O que mudaria.** Um alinhamento entre as duas contagens, palavra por
palavra, guardado no índice. Cada ocorrência mostraria só o seu sentido.

**Antes de fazer.** Medir quantas ocorrências têm esse problema, como foi
feito para a ligação por versículo (90% e 92% das palavras acham verbete;
99% e 91% delas acham o sentido). Se forem poucas, o custo não compensa.

### Passagens paralelas palavra a palavra

**Hoje.** O `-p` diz quais passagens são paralelas, mas não o que elas
têm em comum.

**O que mudaria.** O banco da UBS marca, para cada palavra de cada
passagem, se ela coincide com a outra passagem por inteiro, em parte ou
não coincide (`HEB="0020052212222"`: 2 é inteira, 1 é em parte, 0 é não).
Com o `-s`, o interlinear das duas passagens poderia destacar as palavras
que coincidem, por exemplo em cor ou com uma marca. Isso mostra, em Hc 2.4
e Rm 1.17, exatamente o que Paulo tomou do profeta.

**Limites.**

- Nas citações do AT no NT, as marcas valem para a Septuaginta, a
  tradução grega que o NT cita, e não para o hebraico. O interlinear do
  AT é hebraico, então a marca do lado do AT não se aplicaria a ele.
- As marcas seguem a contagem de palavras da UBS, com o mesmo problema de
  alinhamento do item anterior.

## Referências

### Partes de versículo (a, b)

**Hoje.** O parser recusa qualquer letra depois do número do versículo:
`Jo 3.16a`, `Rm 8.28b` e `Jo 3.16-17a` dão "referência mal formada", no
terminal, no `filtro` dos editores e nos atalhos do GNOME.

**O que mudaria.** Duas coisas independentes:

1. **Aceitar a notação.** O parser guarda a letra (a, b, c) no início e
   no fim de cada trecho, e a saída a repete onde hoje imprime a
   referência: `Jo 3.16a` no terminal, nos formatos de colar
   (`{{.Ref}}`) e no `#bref`. A consulta ao servidor não muda: ele só
   conhece versículos inteiros. É simples e não tem risco.
2. **Decidir onde o versículo se divide.** Nenhuma fonte define o "a" e o
   "b" do texto em português; a divisão é convenção de quem cita, em
   geral no sinal de pontuação principal. No AT, o hebraico tem uma
   divisão de verdade, o acento *atnach*, que a STEPBible traz; mas ela
   vale para o hebraico, e levá-la à ARA seria palpite. As opções:

   | | Texto | Risco |
   | --- | --- | --- |
   | **A** | o versículo inteiro, com a referência `Jo 3.16a`; o autor corta à mão | nenhum |
   | **B** | corte automático no sinal mais forte (`;`, `:`, `.`, `—`) perto do meio | errar o corte numa citação, sem que ninguém perceba |
   | **C** | o versículo inteiro, com a parte provável destacada no terminal; nos formatos de colar, como em A | baixo: o destaque só sugere |

   A recomendação é o item 1 com A, e talvez o destaque de C no terminal.

**Antes de fazer.** Saber o que o verificador do projeto de lições faz
com o par `// conferido` + `#bref`:

- se compara o texto do `#bref` com o versículo inteiro, um texto
  cortado à mão é reprovado, e a referência com letra precisa de
  tratamento lá também;
- se só confere que o texto está contido no versículo, cortar à mão já
  funciona, e A basta.

## Fora do editor

### Atalhos da seleção em outros sistemas

**Hoje.** O `biblia selecao` consulta o texto selecionado em qualquer
programa (referência ou número de Strong), e o `biblia atalhos instalar`
cadastra `Super+B`, `Super+Alt+B` e `Super+Shift+B`. As duas coisas só
existem para o Linux: o `selecao` lê a seleção com o `wl-paste` ou o
`xclip` e responde com o `zenity`; o `atalhos` cadastra só no GNOME, pelo
`gsettings`. Em outro sistema, `biblia atalhos` diz que só existe para o
GNOME.

**O que mudaria, por sistema:**

| Sistema | Como funcionaria | O que falta |
| --- | --- | --- |
| **macOS** | uma **Ação Rápida** (Serviços) aparece no clique direito sobre texto selecionado, em qualquer aplicativo: é o único sistema com menu de contexto de verdade | `selecao` lendo o texto da entrada padrão (o Serviço o entrega assim), janela pelo `osascript` (`display dialog`), cópia pelo `pbcopy`; a Ação Rápida em si se cria no Automator e pode ir como instrução no README ou como arquivo `.workflow` |
| **KDE e outros Linux** (XFCE, Cinnamon) | o `selecao` já funciona: o `wl-paste` e o `xclip` existem ali, e o `zenity` também | o `atalhos` só cadastra no GNOME; nos outros, o atalho se cria à mão em *Configurações › Atalhos* (README). No KDE, as ações do Klipper poderiam oferecer "consultar no biblia" quando o texto copiado parece uma referência |
| **Windows** | não há menu de contexto para texto nem seleção primária; um atalho global (AutoHotkey ou PowerToys) manda `Ctrl+C` e chama o `biblia` | `selecao` lendo a área de transferência (PowerShell `Get-Clipboard`) e mostrando o resultado (janela do PowerShell ou notificação); o atalho fica como instrução no README |

**Por onde começar.** Pelo macOS, que dá o menu de contexto pedido
originalmente; o primeiro passo, útil para todos, é o `selecao` aceitar
o texto pela entrada padrão (por exemplo, `biblia selecao -`), sem
depender de ler a seleção.

**Antes de fazer.** Ter onde testar: nenhum dos três foi experimentado.

### Janelas melhores que as do zenity

**Hoje.** O `biblia selecao` responde em janelas do `zenity`: um texto
com os botões *Traduções…* e *Fechar*, e uma lista de marcar para
escolher as traduções. Trocar de tradução fecha a janela e abre outra.
Não há como ter um combobox dentro da janela de texto: no `zenity`, o
combobox só existe no diálogo de formulário, que não mostra texto longo.

**O que mudaria.** Uma janela que faça mais sem reabrir: escolher a
tradução num combobox ao lado do texto, alternar entre o texto e o
original, abrir um número de Strong do interlinear. As opções:

| Opção | O que ganha | O que custa |
| --- | --- | --- |
| **yad** (*Yet Another Dialog*) | fork do `zenity` com muito mais: combobox e texto no mesmo diálogo, botões com ações próprias, janela que se atualiza sem fechar | não vem instalado no Ubuntu (`apt install yad`), e o projeto tem um só mantenedor. É o passo menor: a troca fica dentro do `internal/desktop`, com o `zenity` como alternativa quando o `yad` faltar |
| **Interface de terminal** (Bubble Tea) num `gnome-terminal` | combobox, atalhos de teclado e rolagem em Go puro; a mesma nos três sistemas | a janela é um terminal, menos com cara de aplicativo; cada sistema tem o seu terminal a abrir |
| **Interface gráfica em Go** (Fyne, Gio) | janela de verdade, do jeito que se quiser, nos três sistemas | exige CGO e OpenGL: executável bem maior, e a compilação cruzada de hoje (`CGO_ENABLED=0`, os seis binários num só runner) deixa de funcionar |
| **Extensão do GNOME Shell** | a integração mais nativa: um popup junto do cursor, sem janela à parte | só GNOME, em JavaScript, e costuma quebrar a cada versão do GNOME |
| **kdialog** | o equivalente do `zenity` no KDE | os mesmos limites do `zenity` |

**Qual escolher.** Para melhorar a janela no Linux, o **yad**. Se a meta
for levar a seleção ao macOS e ao Windows (item anterior), a
**interface de terminal**, que mantém o binário único e sem CGO.

**Antes de fazer.** Que a janela de hoje se mostre insuficiente no uso:
o botão *Traduções…* já resolve a troca de tradução.

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

# Como publicar

O passo a passo para lançar o programa e as extensões. A parte que se
faz **uma vez só** (criar contas e dar permissões) vem separada da que
se repete **a cada versão**.

Cada peça tem a sua versão e a sua tag, independentes:

| Peça | Tag | O que a tag dispara |
| --- | --- | --- |
| programa `biblia` | `v0.1.0` | binários para Linux, macOS e Windows na página de *Releases* (`.github/workflows/release.yml`) |
| extensão do VS Code | `vscode-v0.2.0` | publicação no Visual Studio Marketplace e no Open VSX (`.github/workflows/vscode.yml`) |
| pacote do Sublime | `sublime-0.2.0` | nada: é criada pelo `make sublime-release`, que publica o pacote no repositório espelho |

**A ordem importa na primeira vez:** o programa primeiro. As extensões
mandam baixar o `biblia` da página de *Releases*; se elas saírem antes,
quem instalar não terá de onde tirá-lo.

## 1. O programa

### A cada versão do programa

1. Atualize o `CHANGELOG.md`: troque `[Unreleased]` pela versão e pela
   data e abra um `[Unreleased]` novo, vazio.
2. Commit e push.
3. Crie e envie a tag:

   ```sh
   git tag -a v0.1.0 -m "biblia 0.1.0"
   git push origin v0.1.0
   ```

4. Acompanhe a Action **Release** na aba *Actions* do GitHub (cerca de 10
   minutos: são seis binários). No fim, a página
   [Releases](https://github.com/Casara/biblia-cli/releases) tem a versão
   com seis arquivos (`linux`, `darwin` e `windows`, cada um em `amd64` e
   `arm64`) e o `checksums.txt`.

Para testar a geração sem publicar nada:
`go run github.com/goreleaser/goreleaser/v2@v2.19.0 release --snapshot --clean`
(os arquivos ficam em `dist/`, que o git ignora).

## 2. Extensão do VS Code

A publicação é feita pela Action, sem token guardado no GitHub: o
Marketplace e o Open VSX confiam na Action por OpenID Connect (*Trusted
Publishing*). Os tokens pessoais do Azure DevOps, o caminho antigo
(`vsce login`), deixam de funcionar em 1º de dezembro de 2026.

### Uma vez só: Visual Studio Marketplace

1. **Crie o publisher.** Entre em
   <https://marketplace.visualstudio.com/manage> com uma conta Microsoft
   e clique em **Create publisher**.
   - **ID:** `casara`. Precisa ser igual ao campo `publisher` do
     `editors/vscode/package.json`, e não pode ser mudado depois. Se
     `casara` já estiver em uso, escolha outro e troque também no
     `package.json`.
   - **Name:** o nome que aparece na página da extensão, como
     `Casara`.
2. **Autorize a Action.** Na página do publisher, abra **Trusted
   Publishing**, expanda **Create** e preencha:

   | Campo | Valor |
   | --- | --- |
   | Policy name | `github-biblia-cli` (qualquer nome) |
   | Repository owner | `Casara` |
   | Repository | `biblia-cli` |
   | Workflow file | `vscode.yml` |
   | Environment | `vscode-marketplace` |

3. **Crie o ambiente no GitHub.** No repositório: **Settings ›
   Environments › New environment**, nome `vscode-marketplace`. Em
   **Required reviewers**, ponha você mesmo: a publicação passa a esperar
   um clique seu de aprovação na Action, o que evita publicar por engano.

### Uma vez só: Open VSX

O Open VSX é a loja das variantes abertas do VS Code (VSCodium, Cursor,
Gitpod). É opcional; se não quiser, apague o job `open-vsx` do
`.github/workflows/vscode.yml`.

1. **Conta.** Entre em <https://open-vsx.org> com o GitHub. No seu
   perfil, ligue uma conta da Eclipse Foundation, criada em
   <https://accounts.eclipse.org> **com o mesmo usuário do GitHub**.
2. **Acordo.** No perfil do open-vsx.org aparece **Show Publisher
   Agreement**: leia e assine. Sem isso não se publica.
3. **Namespace.** Crie um token em **Settings › Access Tokens** e rode:

   ```sh
   npx ovsx@1.3.0 create-namespace casara -p <token>
   ```

   O namespace tem de ser igual ao `publisher` do `package.json`. Criar
   não dá exclusividade: para ser o único a publicar nele, peça a posse
   abrindo uma *issue* em
   <https://github.com/EclipseFdn/open-vsx.org/issues> (escolha o modelo
   de pedido de posse de namespace). Depois, apague o token.
4. **Autorize a Action.** Nas configurações do namespace no
   open-vsx.org, cadastre a Action como *trusted publisher*: repositório
   `Casara/biblia-cli`, workflow `vscode.yml`, ambiente `open-vsx`. Os
   nomes exatos dos campos estão no
   [guia de publicação do Open VSX](https://github.com/eclipse/openvsx/wiki/Publishing-Extensions);
   eles podem mudar.
5. **Crie o ambiente no GitHub:** `open-vsx`, como no passo 3 do
   Marketplace.

### A cada versão da extensão

1. Suba a versão em `editors/vscode/package.json` (`"version"`) e anote
   as mudanças em `editors/vscode/CHANGELOG.md`.
2. Commit e push.
3. Crie e envie a tag, com a **mesma versão** do `package.json` (a
   Action confere e para se forem diferentes):

   ```sh
   git tag -a vscode-v0.2.0 -m "extensão do VS Code 0.2.0"
   git push origin vscode-v0.2.0
   ```

4. Na aba *Actions*, a Action **VS Code extension** empacota a extensão
   e, depois da sua aprovação (se pôs revisor), publica nas duas lojas.
   A página aparece em alguns minutos:
   <https://marketplace.visualstudio.com/items?itemName=casara.biblia>.

### Se a publicação pela Action falhar

O *Trusted Publishing* do Marketplace é recente, e pode não aceitar a
**primeira** publicação de uma extensão nova. Não é preciso token para
contornar:

1. Na página da execução da Action, baixe o artefato **vsix** (um `.zip`
   com o `biblia-0.2.0.vsix` dentro).
2. Em <https://marketplace.visualstudio.com/manage>, clique em **New
   extension › Visual Studio Code** e envie o `.vsix`. Para uma versão
   nova de uma extensão que já existe, use **⋯ › Update** na linha dela.
3. No Open VSX, sem a Action, publica-se com um token:
   `npx ovsx@1.3.0 publish biblia-0.2.0.vsix -p <token>`.

## 3. Pacote do Sublime Text

O Package Control exige o pacote **na raiz de um repositório próprio**,
com uma tag de versão. O código continua sendo editado aqui, em
`editors/sublime`; o `make sublime-release` copia essa pasta, com o
histórico dela, para o repositório espelho `Casara/biblia-sublime`.

### Uma vez só: repositório espelho e Package Control

1. **Crie o repositório espelho** no GitHub: `Casara/biblia-sublime`,
   **público** e **vazio** (sem README, sem licença: tudo vem daqui).
   Descrição sugerida: *Bíblia for Sublime Text: Bible verses, Strong's
   numbers and parallel passages through the biblia CLI.*
2. **Publique a primeira versão** (*A cada versão do pacote*, abaixo).
3. **Peça a inclusão no Package Control:**
   1. Faça um *fork* de
      <https://github.com/sublimehq/package_control_channel>.
   2. No fork, edite `repository/b.json` e acrescente, **em ordem
      alfabética** pelo nome:

      ```json
      {
        "name": "Biblia",
        "details": "https://github.com/Casara/biblia-sublime",
        "labels": ["bible", "text manipulation", "markdown", "typst"],
        "releases": [
          {
            "sublime_text": ">=4000",
            "tags": true
          }
        ]
      },
      ```

      `>=4000` é o Sublime Text 4: o plugin roda no Python 3.8, que o
      Sublime 3 não tem.
   3. Abra o *pull request* e marque a lista de verificação do modelo.
      O pacote já atende a todos os itens: não ativa atalhos nem menu de
      contexto; todos os comandos estão na paleta; configurações e atalhos
      abrem pelo menu e pela paleta, lado a lado com os padrões; o
      `.gitattributes` deixa de fora o que não serve dentro do Sublime; e
      há uma tag de versão.
   4. Aguarde a revisão, que costuma levar de dias a semanas. Os
      revisores podem pedir ajustes nos comentários do PR.

### A cada versão do pacote

1. Anote as mudanças em `editors/sublime/CHANGELOG.md`.
2. Commit e push.
3. Rode, com a versão nova:

   ```sh
   make sublime-release V=0.2.0
   ```

   O alvo confere se `editors/sublime` não tem mudanças sem commit, envia
   a pasta para o `main` do `Casara/biblia-sublime`, cria lá a tag
   `0.2.0` e cria aqui a tag `sublime-0.2.0`. O Package Control percebe a
   tag nova sozinho, em algumas horas.

## Conferência depois de publicar

- **Programa:** baixe o arquivo do seu sistema na página de *Releases*,
  descompacte e rode `./biblia versao`: deve mostrar a versão da tag.
- **VS Code:** instale a extensão pela loja num perfil limpo
  (**Arquivo › Novo perfil**) e teste o `Ctrl+Alt+B` numa linha com
  `Jo 3.16`.
- **Sublime:** depois de aceito, **Package Control: Install Package ›
  Biblia**; antes disso, dá para testar o espelho com **Package Control:
  Add Repository** › `https://github.com/Casara/biblia-sublime`.

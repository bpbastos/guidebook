# Como manter e evoluir

Este é o caminho de uma mudança no guidebook, do clone à versão
publicada. As regras completas estão no `AGENTS.md`.

## Prepare o ambiente

Requisitos: Go 1.27.2 ou mais novo, git e make. O Docker só é preciso
para construir a imagem de container.

No clone, ative o hook de mensagens de commit:

```sh
make hooks
```

O hook recusa a mensagem fora da convenção e lista todas as violações
de uma vez. Uma mensagem como `Adicionado o recurso.` sai com código 1.

## Rode os testes

```sh
make test
```

O alvo roda `go test ./...`. A CI roda também `gofmt -l .` e
`go vet ./...`; rode os dois antes do push.

## Compile e veja os guias

```sh
make build
```

O binário fica em `bin/guidebook`. Para ver os guias no navegador:

```sh
make preview
```

O alvo gera os guias do próprio guidebook em `out/guidebook/` e os do
projeto de exemplo em `out/example/`. No fim, imprime os quatro
caminhos `file://`. O guia do desenvolvedor existe só como esse arquivo
local; o `guidebook serve` serve só o guia do usuário.

Para construir a imagem de container no seu computador:

```sh
docker build -t guidebook:local .
```

## Fluxo de trabalho

O trabalho segue as fases da spec: uma fase, uma branch, um pull
request.

1. A branch nasce da `main` atualizada, com prefixo `chore/`, `docs/`,
   `feat/` ou `fix/`.
2. Os commits seguem a convenção do `AGENTS.md`. Em fase com código, o
   teste falha pelo motivo certo antes da implementação.
3. O push é só da branch, por SSH, nunca da `main` e nunca com força.
4. Depois do push, o agente avisa o conselheiro por `SendMessage`, com
   branch, SHA, resumo, testes e decisões.
5. O conselheiro revisa, abre o pull request e mescla por squash. A
   `main` é protegida e só recebe assim.

A `main` local só anda com `git fetch origin`, `git switch main` e
`git merge --ff-only origin/main`.

## Acrescente uma funcionalidade

1. Confira se a spec cobre o comportamento. Se não cobrir, a spec muda
   no mesmo pull request.
2. Escreva o teste que falha e rode `make test` para ver o vermelho.
3. Implemente até o verde.
4. Atualize a página do guia do usuário do guidebook, em `docs/user/`.
   A CI roda o `check` sobre o próprio repositório.
5. Registre a mudança em "Não publicado" no `CHANGELOG.md`.

Decisão com alternativa real vira um ADR curto em
`docs/adr/NNNN-titulo.md`, com contexto, decisão e consequência.
Dependência nova, ferramenta nova e mudança de convenção são decisões
do Bruno.

## A CI

O workflow `ci.yml` roda em todo pull request e na `main`. O job `test`
roda gofmt, vet, os testes e o `check` do exemplo e do repositório. O
job `image` constrói a imagem e confere que ela serve o guia do usuário
do exemplo com o sistema de arquivos somente leitura.

## Publique uma versão

A versão nasce de uma tag `vX.Y.Z`, criada pelo Bruno. Antes da tag, o
marco de "Não publicado" no `CHANGELOG.md` ganha número e data.

A tag dispara o workflow `release.yml`. Ele confere o formato da tag,
roda os testes e publica a imagem `ghcr.io/bpbastos/guidebook:<tag>` no
GitHub Container Registry. A autenticação usa o `GITHUB_TOKEN` do
próprio workflow, sem segredo no repositório.

# Como manter e evoluir

## Suba o ambiente

Com o Docker instalado, rode `make up` na raiz do repositório.

## Fluxo de trabalho

Cada mudança nasce numa branch, passa pela revisão e entra por pull
request. A CI roda os testes em todo pull request.

## Acrescente uma funcionalidade

Escreva primeiro o teste que falha. Depois, a regra de negócio e a
página do guia do usuário.

## Rode os testes

Rode `make test`.

## Publique uma versão

Uma tag `vX.Y.Z` na `main` publica a versão.

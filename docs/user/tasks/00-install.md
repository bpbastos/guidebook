# Instalar o guidebook

Há três jeitos de usar o guidebook. Escolha o que combina com o
projeto.

## Instale com o Go

Com o Go 1.27.2 ou mais novo, instale uma versão fixa:

```sh
go install github.com/bpbastos/guidebook/cmd/guidebook@v0.1.0
```

O programa fica em `$(go env GOPATH)/bin`. Confira a versão:

```sh
guidebook version
```

## Compile a partir do código

No clone do guidebook, rode:

```sh
make build
```

O programa fica em `bin/guidebook`.

## Use a imagem de container

Cada versão publica a imagem `ghcr.io/bpbastos/guidebook:<versão>`, por
exemplo `ghcr.io/bpbastos/guidebook:v0.1.0`. A imagem roda o `serve`.
Veja como pôr no compose do projeto em
[Servir os guias](04-serve-guides.md).

Use sempre uma versão fixa, nunca `latest`.

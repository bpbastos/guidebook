# guidebook

Ferramenta em Go que gera guias interativos em HTML a partir de
Markdown: o guia do usuário e o guia do desenvolvedor. Serve para vários
projetos e abre sem servidor e sem internet.

O próprio guidebook se documenta assim. Para gerar e abrir os guias
dele e os do projeto de exemplo:

```sh
make preview
```

O comando imprime os caminhos dos HTML para abrir no navegador.

## Depois de clonar

```sh
make hooks
```

Isso ativa o hook de mensagem de commit em `.githooks/`.

## Onde está cada coisa

- Como usar: guia do usuário, a partir de `docs/user/`.
- Como funciona por dentro: guia do desenvolvedor, a partir de
  `docs/developer/`, `docs/spec.md` e `docs/adr/`.
- Regras do projeto: [AGENTS.md](AGENTS.md).

## Licença

[MIT](LICENSE).

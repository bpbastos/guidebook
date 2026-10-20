# Seu primeiro guia

Neste tutorial você cria um guia do usuário com uma página. No fim, abre
o guia no navegador.

## Instale o guidebook

Siga [Instalar o guidebook](../tasks/00-install.md). Confira com:

```sh
guidebook version
```

## Crie o guidebook.yaml

Na raiz do seu projeto, crie o arquivo `guidebook.yaml`:

```yaml
version: 1

product:
  name: Pedido Fácil

user:
  dir: docs/user
```

## Escreva a primeira página

Crie o arquivo `docs/user/tasks/01-register-order.md`:

```markdown
# Registrar um pedido

1. Escolha o cliente.
2. Adicione os itens.
3. Clique em Salvar.
```

A pasta `tasks` diz que a página é uma [tarefa](../glossary.md#tarefa).
O título vem da linha que começa com `#`.

## Gere o guia

Na raiz do projeto, confira e gere o guia:

```sh
guidebook check
guidebook build
```

O `check` responde "nenhum problema encontrado". O guia fica em
`out/user-guide.html`. Abra esse arquivo com dois cliques.

## Próximos passos

Veja como [escrever as páginas](../tasks/01-write-pages.md), como
[conferir a documentação](../tasks/02-check-docs.md) e como
[integrar com o e2e](../tasks/05-integrate-e2e.md).

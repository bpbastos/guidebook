# Escrever as páginas

As páginas são arquivos Markdown comuns. O guidebook usa poucas
convenções, todas descritas aqui.

## Escolha a pasta pelo tipo da página

Cada página fica numa das quatro pastas de `docs/user/`:

| Pasta | Tipo | Para quê |
|---|---|---|
| `tutorials/` | tutorial | aprender fazendo, do começo ao fim |
| `tasks/` | tarefa | resolver um problema conhecido |
| `reference/` | referência | consultar um detalhe |
| `explanation/` | explicação | entender o porquê |

Na raiz de `docs/user/` ficam só `index.md`, `glossary.md` e `media/`.
Veja o que é [Diátaxis](../glossary.md#diátaxis).

## Dê um título e ordene

1. Comece a página com uma linha `# Título`.
2. Use um título só por página.
3. Para ordenar, ponha um número no começo do nome do arquivo, como
   `01-register-order.md`.

O número não entra no endereço da página. O arquivo
`tasks/01-register-order.md` vira a página `tasks/register-order`.

## Pense nos slides

No modo apresentação, cada `##` começa um slide. Para cortar um slide
no meio de uma seção, use uma linha só com `---`. Seções curtas dão
slides melhores.

## Ligue as páginas

Use links relativos comuns, como `[Fechar o dia](02-close-day.md)`. O
destino decide o que acontece:

- link para o glossário, referência, explicação, tarefa ou tutorial abre
  numa janela, sem sair da página;
- na janela de tarefa ou tutorial, o botão "Abrir a página inteira"
  navega até lá;
- depois de navegar, aparece o botão "Voltar para onde eu estava";
- na página inicial (`index.md`), link para tarefa ou tutorial navega
  direto, como o menu.

Veja os detalhes em [Como o guia funciona](../explanation/01-how-the-guide-works.md).

Para levar a uma seção, acrescente a âncora do título, do mesmo jeito
que no GitHub: minúsculas, acentos mantidos, pontuação fora e cada
espaço trocado por hífen. A seção "Configuração — avançada" vira
`#configuração--avançada`, com dois hífens. A forma do GitLab, com os
hífens juntados, também funciona. Título repetido na página ganha
`-1`, `-2`...

## Explique os bloqueios

Quando algo impede o usuário, abra uma seção que começa com "O sistema
não me deixa". Por exemplo:

```markdown
## O sistema não me deixa salvar o pedido

O botão Salvar fica desligado enquanto falta o cliente.
```

Essas seções formam o índice "O sistema não me deixa...". O começo do
título muda em `user.blocked_prefix`.

## Monte o glossário

No `glossary.md`, cada termo é uma seção `##`. Para citar um termo, use
`[pedido](../glossary.md#pedido)`.

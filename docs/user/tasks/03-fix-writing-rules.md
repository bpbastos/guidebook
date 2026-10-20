# Corrigir as recusas das regras de escrita

O guidebook recusa três coisas no Markdown. Cada recusa traz o arquivo e
a linha.

## Frase longa

```
docs/user/tasks/a.md:7: frase com 31 palavras (limite: 25): "Quando o atendente abre a tela de pedidos…"
```

A frase passou do limite de palavras. Quebre a frase em duas ou mais,
sem cortar informação. Listas ajudam quando há vários passos.

O limite muda em `rules.max_sentence_words`. Código e endereços não
contam como palavras. Um número com ponto ou vírgula entre os dígitos,
como `1.234,56`, conta como uma palavra só.

## Citação proibida

```
docs/user/tasks/a.md:3: não cite chamado interno: "ABC-123"
```

O texto casou com um padrão de `rules.forbidden_patterns`. A mensagem
antes dos dois-pontos é a do projeto. Tire a citação ou troque por um
texto para o usuário.

## HTML cru

```
docs/user/tasks/a.md:9: HTML cru não é permitido no Markdown
```

Troque o HTML por Markdown. Para mostrar uma marcação como exemplo,
ponha entre crases.

## O guidebook não me deixa usar um link

O link aponta para um destino que não existe ou que o guia não aceita.
Confira o caminho e a âncora. A âncora segue a regra do GitHub e
mantém os acentos: `#códigos-de-saída`, não `#codigos-de-saida`.

Links externos precisam de `http`, `https` ou `mailto`. Outros
esquemas, como `javascript:` e `data:`, são recusados.

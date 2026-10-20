# Capturas com marcadores

O e2e do projeto grava cada captura com um arquivo de marcadores ao
lado. O guidebook desenha os marcadores e monta a visita guiada.

## Arquivos

Para a captura `docs/user/media/register-order-1.png`, o e2e grava
`docs/user/media/register-order-1.markers.json`:

```json
{
  "version": 1,
  "image": "register-order-1.png",
  "width": 640,
  "height": 400,
  "markers": [
    { "n": 1, "x": 0.266, "y": 0.2, "text": "Campo Cliente" },
    { "n": 2, "x": 0.641, "y": 0.2, "text": "Botão Adicionar item" },
    { "n": 3, "x": 0.867, "y": 0.919, "text": "Botão Salvar" }
  ]
}
```

## Campos

| Campo | Regra |
|---|---|
| `version` | sempre `1` |
| `image` | nome do arquivo da captura |
| `width`, `height` | tamanho da captura em pixels, igual ao do arquivo |
| `n` | número do marcador, de 1 em diante, em ordem |
| `x`, `y` | centro do marcador, de 0 a 1, a partir do canto de cima à esquerda |
| `text` | opcional: rótulo curto, até 80 caracteres; sem ele, vale o texto do passo `n` da lista |

A captura precisa ser PNG ou JPEG.

## Na página

Ponha a imagem sozinha num parágrafo e, logo depois, uma lista
numerada. O marcador 1 corresponde ao item 1 da lista, e assim por
diante.

```markdown
![Tela de novo pedido](../media/register-order-1.png)

1. Escolha o cliente.
2. Adicione os itens.
3. Clique em Salvar.
```

## O guidebook não me deixa usar os marcadores

O `check` diz o que falta. Os casos mais comuns:

- a lista depois da imagem tem outro número de itens;
- falta a lista numerada logo depois da imagem;
- `width` e `height` não batem com a captura, porque ela foi gravada de
  novo sem atualizar os marcadores.

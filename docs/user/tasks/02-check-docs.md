# Conferir a documentação

O guidebook confere a documentação antes de gerar os guias. Rode a
conferência no seu computador e no CI do projeto.

## Rode o check

Na raiz do projeto, rode:

```sh
guidebook check
```

O `check` faz as mesmas conferências do `build`, sem gravar nada. Sem
problemas, ele responde "nenhum problema encontrado".

## Entenda o resultado

Cada problema sai numa linha, com arquivo e linha:

```
docs/user/tasks/register-order.md:12: frase com 31 palavras (limite: 25): "Quando o atendente abre a tela de pedidos…"
```

O guidebook lista todos os problemas de uma vez. Veja como
[corrigir as recusas](03-fix-writing-rules.md).

O código de saída diz o tipo de problema. A lista está nos
[comandos](../reference/01-commands.md#códigos-de-saída).

## Ponha no CI do projeto

Acrescente um passo que roda `guidebook check`. Qualquer problema faz o
passo falhar.

## O guidebook não me deixa gravar na pasta de saída

O `build` só apaga uma pasta vazia ou uma pasta que ele mesmo criou. Ele
reconhece a própria pasta pelo arquivo `.guidebook-output`.

Escolha outra pasta com `-out`, ou esvazie a pasta antes.
